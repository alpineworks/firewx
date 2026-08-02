package nfdrs

import (
	"fmt"
	"math"
	"time"

	firewx "alpineworks.io/firewx"
	"alpineworks.io/firewx/nfdrs/gsi"
	"alpineworks.io/firewx/nfdrs/nelson"
	"alpineworks.io/firewx/simple"
)

// driverSchemaVersion is the schema version stamped into a DriverState. Increase
// it when the meaning of a field changes.
const driverSchemaVersion = 1

// Config holds the site settings for a Driver.
type Config struct {
	// FuelModel is the fuel model of the site.
	FuelModel FuelModel
	// Latitude is the site latitude in decimal degrees, for the day length.
	Latitude float64
	// SlopeClass is the slope steepness class, 1 to 5.
	SlopeClass int
	// KBDIThreshold is the drought index above which the drought fuel load
	// starts.
	KBDIThreshold float64
	// MeanAnnualPrecip is the mean annual precipitation, for the drought index.
	MeanAnnualPrecip firewx.Inches
	// AnnualHerb sets whether the herbaceous fuel is annual.
	AnnualHerb bool

	// RegObsHour is the hour of the daily observation in local standard time.
	// The driver updates the live fuel moisture and the drought index at this
	// hour.
	RegObsHour int
	// LSTOffset is the fixed offset from UTC that defines local standard time.
	LSTOffset time.Duration
}

// Driver runs the National Fire Danger Rating System over a series of hourly
// weather observations. It advances the Nelson dead fuel moisture sticks every
// hour. It advances the live fuel moisture and the drought index once per day,
// at the regular observation hour. It computes the four indices every hour.
//
// A Driver is not safe for concurrent use.
type Driver struct {
	cfg Config

	stick1    *nelson.Stick
	stick10   *nelson.Stick
	stick100  *nelson.Stick
	stick1000 *nelson.Stick

	herb  *gsi.Model
	woody *gsi.Model
	kbdi  simple.KBDIState

	// Nelson-derived outputs, updated every hour.
	mc1, mc10, mc100, mc1000 float64
	fuelTemp                 firewx.Celsius

	// Live fuel and drought outputs, updated once per day.
	gsiValue       float64
	mcHerb, mcWood float64
	kbdiValue      float64

	// Daily accumulators over the local standard day.
	haveDay                          bool
	dayMinTemp, dayMaxTemp, dayMinRH float64
	dayPrecip                        firewx.Inches

	last     time.Time
	haveLast bool
	indices  Indices
}

// NewDriver returns a Driver for the site settings. The Nelson sticks and the
// live fuel moisture models start from their default initial states.
func NewDriver(cfg Config) *Driver {
	d := &Driver{
		cfg:       cfg,
		stick1:    nelson.NewStandardStick(nelson.OneHour),
		stick10:   nelson.NewStandardStick(nelson.TenHour),
		stick100:  nelson.NewStandardStick(nelson.HundredHour),
		stick1000: nelson.NewStandardStick(nelson.ThousandHour),
		herb:      gsi.NewHerbaceous(cfg.Latitude, cfg.AnnualHerb),
		woody:     gsi.NewWoody(cfg.Latitude),
		kbdi:      simple.NewKBDIState(cfg.MeanAnnualPrecip),
	}
	return d
}

// Update advances the system by one hourly observation. It returns false if the
// observation lacks the temperature, the humidity, the solar radiation, or the
// precipitation that the Nelson sticks need.
func (d *Driver) Update(o firewx.Obs) bool {
	temp, okT := o.Temperature.Get()
	rh, okH := o.RelativeHumidity.Get()
	solar, okS := o.SolarRadiation.Get()
	precip, okP := o.Precipitation.Get()
	if !okT || !okH || !okS || !okP {
		return false
	}
	snow := o.SnowCovered.Or(false)

	elapsed := time.Hour
	if d.haveLast {
		elapsed = o.Time.Sub(d.last)
	}
	d.last = o.Time
	d.haveLast = true

	// Feed the Nelson sticks. The temperature is rounded to two decimal places,
	// which matches firelab/NFDRS4. Snow cover replaces the weather with a cold,
	// wet, dark, dry input.
	nelTemp := math.Floor(float64(temp)*100+0.5) / 100
	nelRH := rh
	nelSolar := solar
	nelPrecip := precip
	if snow {
		nelTemp = 0
		nelRH = 99.9
		nelSolar = 0
		nelPrecip = 0
	}
	// The barometric pressure is left absent, so the Nelson sticks use their
	// default of 0.0218 cal/cm3, which matches firelab/NFDRS4.
	w := nelson.Weather{
		Elapsed:          elapsed,
		Temperature:      firewx.Celsius(nelTemp),
		RelativeHumidity: nelRH,
		SolarRadiation:   nelSolar,
		Rainfall:         nelPrecip,
	}
	d.stick1.Update(w)
	d.stick10.Update(w)
	d.stick100.Update(w)
	d.stick1000.Update(w)

	d.mc1 = float64(d.stick1.MedianRadialMoisture())
	d.mc10 = float64(d.stick10.MedianRadialMoisture())
	d.mc100 = float64(d.stick100.MedianRadialMoisture())
	d.mc1000 = float64(d.stick1000.MedianRadialMoisture())
	d.fuelTemp = d.stick1.SurfaceTemperature()

	// Accumulate the daily minimum and maximum over the local standard day.
	tF := float64(temp.Fahrenheit())
	rhV := float64(rh)
	if !d.haveDay {
		d.dayMinTemp, d.dayMaxTemp, d.dayMinRH = tF, tF, rhV
		d.dayPrecip = 0
		d.haveDay = true
	} else {
		d.dayMinTemp = math.Min(d.dayMinTemp, tF)
		d.dayMaxTemp = math.Max(d.dayMaxTemp, tF)
		d.dayMinRH = math.Min(d.dayMinRH, rhV)
	}
	d.dayPrecip += precip.Inches()

	// Update the live fuel moisture and the drought index once per day, at the
	// regular observation hour.
	lst := o.Time.Add(d.cfg.LSTOffset)
	if lst.Hour() == d.cfg.RegObsHour {
		d.updateDaily(o.Time)
	}

	d.indices = d.compute(o)
	return true
}

// updateDaily advances the live fuel moisture and the drought index from the
// daily accumulators, then it resets the accumulators.
func (d *Driver) updateDaily(t time.Time) {
	dayObs := gsi.DailyObs{
		Date:           t,
		MinTemperature: firewx.Fahrenheit(d.dayMinTemp).Celsius(),
		MaxTemperature: firewx.Fahrenheit(d.dayMaxTemp).Celsius(),
		MinHumidity:    firewx.Percent(d.dayMinRH),
	}
	d.herb.Update(dayObs)
	d.woody.Update(dayObs)
	d.gsiValue = d.herb.GSI()
	d.mcHerb = float64(d.herb.Moisture())
	d.mcWood = float64(d.woody.Moisture())

	d.kbdi.Step(firewx.Fahrenheit(d.dayMaxTemp), d.dayPrecip)
	d.kbdiValue = float64(d.kbdi.Index)

	d.haveDay = false
}

// compute builds the conditions and returns the indices.
func (d *Driver) compute(o firewx.Obs) Indices {
	wind := o.WindSpeed.Or(0).MilesPerHour()
	c := Conditions{
		Moisture1:        d.mc1,
		Moisture10:       d.mc10,
		Moisture100:      d.mc100,
		Moisture1000:     d.mc1000,
		MoistureHerb:     d.mcHerb,
		MoistureWoody:    d.mcWood,
		KBDI:             d.kbdiValue,
		KBDIThreshold:    d.cfg.KBDIThreshold,
		WindSpeed:        wind,
		SlopeClass:       d.cfg.SlopeClass,
		FuelTemperature:  d.fuelTemp,
		GSI:              d.gsiValue,
		GSIMax:           1.0,
		GreenupThreshold: 0.5,
	}
	return d.cfg.FuelModel.Compute(c)
}

// Indices returns the four indices from the most recent update.
func (d *Driver) Indices() Indices { return d.indices }

// DeadMoistures returns the four dead fuel moisture contents from the most
// recent update, as percentages.
func (d *Driver) DeadMoistures() (mc1, mc10, mc100, mc1000 firewx.Percent) {
	return firewx.Percent(d.mc1), firewx.Percent(d.mc10), firewx.Percent(d.mc100), firewx.Percent(d.mc1000)
}

// DriverState is the full serializable state of a Driver. It holds the state of
// the four Nelson sticks, the two live fuel moisture models, the drought index,
// and the driver's own daily accumulators and last outputs.
//
// A DriverState marshals to JSON and back with an exact round trip, so a caller
// can persist a Driver between runs and resume without the long spin-up that the
// 1000-hour stick needs from a cold start. The caller must construct the Driver
// with the same Config, then restore the state.
//
// The sub-model pointers alias the Driver's live models. Marshal the state
// before the next call to Update.
type DriverState struct {
	SchemaVersion int `json:"schema_version"`

	Stick1    *nelson.Stick    `json:"stick_1hr"`
	Stick10   *nelson.Stick    `json:"stick_10hr"`
	Stick100  *nelson.Stick    `json:"stick_100hr"`
	Stick1000 *nelson.Stick    `json:"stick_1000hr"`
	Herb      *gsi.Model       `json:"herb"`
	Woody     *gsi.Model       `json:"woody"`
	KBDI      simple.KBDIState `json:"kbdi"`

	// Dead fuel moisture and fuel temperature from the most recent update.
	MC1      float64 `json:"mc_1hr"`
	MC10     float64 `json:"mc_10hr"`
	MC100    float64 `json:"mc_100hr"`
	MC1000   float64 `json:"mc_1000hr"`
	FuelTemp float64 `json:"fuel_temp_c"`

	// Live fuel moisture and drought from the most recent daily update.
	GSIValue  float64 `json:"gsi"`
	MCHerb    float64 `json:"mc_herb"`
	MCWood    float64 `json:"mc_woody"`
	KBDIValue float64 `json:"kbdi_value"`

	// Daily accumulators over the current local standard day.
	HaveDay    bool    `json:"have_day"`
	DayMinTemp float64 `json:"day_min_temp_f"`
	DayMaxTemp float64 `json:"day_max_temp_f"`
	DayMinRH   float64 `json:"day_min_rh"`
	DayPrecip  float64 `json:"day_precip_in"`

	Last     time.Time `json:"last"`
	HaveLast bool      `json:"have_last"`
	Indices  Indices   `json:"indices"`
}

// State returns the full state of the Driver. Marshal it to JSON to persist the
// Driver between runs.
func (d *Driver) State() DriverState {
	return DriverState{
		SchemaVersion: driverSchemaVersion,
		Stick1:        d.stick1,
		Stick10:       d.stick10,
		Stick100:      d.stick100,
		Stick1000:     d.stick1000,
		Herb:          d.herb,
		Woody:         d.woody,
		KBDI:          d.kbdi,
		MC1:           d.mc1,
		MC10:          d.mc10,
		MC100:         d.mc100,
		MC1000:        d.mc1000,
		FuelTemp:      float64(d.fuelTemp),
		GSIValue:      d.gsiValue,
		MCHerb:        d.mcHerb,
		MCWood:        d.mcWood,
		KBDIValue:     d.kbdiValue,
		HaveDay:       d.haveDay,
		DayMinTemp:    d.dayMinTemp,
		DayMaxTemp:    d.dayMaxTemp,
		DayMinRH:      d.dayMinRH,
		DayPrecip:     float64(d.dayPrecip),
		Last:          d.last,
		HaveLast:      d.haveLast,
		Indices:       d.indices,
	}
}

// SetState restores the Driver from a saved state. The Driver keeps its Config,
// so construct it with the same Config first. SetState returns an error if the
// schema version does not match or a sub-model is absent.
func (d *Driver) SetState(s DriverState) error {
	if s.SchemaVersion != driverSchemaVersion {
		return fmt.Errorf("nfdrs: driver state schema version %d, want %d", s.SchemaVersion, driverSchemaVersion)
	}
	if s.Stick1 == nil || s.Stick10 == nil || s.Stick100 == nil || s.Stick1000 == nil || s.Herb == nil || s.Woody == nil {
		return fmt.Errorf("nfdrs: driver state has an absent sub-model")
	}
	d.stick1 = s.Stick1
	d.stick10 = s.Stick10
	d.stick100 = s.Stick100
	d.stick1000 = s.Stick1000
	d.herb = s.Herb
	d.woody = s.Woody
	d.kbdi = s.KBDI
	d.mc1 = s.MC1
	d.mc10 = s.MC10
	d.mc100 = s.MC100
	d.mc1000 = s.MC1000
	d.fuelTemp = firewx.Celsius(s.FuelTemp)
	d.gsiValue = s.GSIValue
	d.mcHerb = s.MCHerb
	d.mcWood = s.MCWood
	d.kbdiValue = s.KBDIValue
	d.haveDay = s.HaveDay
	d.dayMinTemp = s.DayMinTemp
	d.dayMaxTemp = s.DayMaxTemp
	d.dayMinRH = s.DayMinRH
	d.dayPrecip = firewx.Inches(s.DayPrecip)
	d.last = s.Last
	d.haveLast = s.HaveLast
	d.indices = s.Indices
	return nil
}
