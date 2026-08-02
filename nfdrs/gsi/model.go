package gsi

import (
	"time"

	firewx "alpineworks.io/firewx"
)

// schemaVersion is the on-disk schema version stamped into a Model. Increase it
// when the meaning of a field changes.
const schemaVersion = 1

// FuelType is the kind of live fuel that a Model predicts.
type FuelType int

const (
	// Herbaceous is live herbaceous fuel, such as grass. It has an annual curing
	// state.
	Herbaceous FuelType = iota
	// Woody is live woody fuel, such as shrub foliage.
	Woody
)

// Default limits for the three GSI indicator functions, from firelab/NFDRS4.
const (
	defaultTminMinC   = -2.0
	defaultTminMaxC   = 5.0
	defaultVPDMinPa   = 900.0
	defaultVPDMaxPa   = 4100.0
	defaultDaylMinSec = 36000.0
	defaultDaylMaxSec = 39600.0
)

// Default parameters for the live fuel moisture map, from firelab/NFDRS4.
const (
	defaultMaxGSI           = 1.0
	defaultGreenupThreshold = 0.5
	defaultMAPeriod         = 21

	defaultHerbMinLFM  = 30.0
	defaultHerbMaxLFM  = 250.0
	defaultWoodyMinLFM = 60.0
	defaultWoodyMaxLFM = 200.0
)

// Model is a stateful GSI live fuel moisture model. It carries the running
// average window of the GSI and, for herbaceous fuel, the annual curing state.
//
// The exported fields hold the complete state. A Model marshals to JSON and back
// with an exact round trip, so a caller can persist the model between runs.
//
// A Model is not safe for concurrent use.
type Model struct {
	SchemaVersion int      `json:"schema_version"`
	FuelType      FuelType `json:"fuel_type"`
	Latitude      float64  `json:"latitude"`
	Annual        bool     `json:"annual"`

	// GSI indicator limits.
	TminMinC   float64 `json:"tmin_min_c"`
	TminMaxC   float64 `json:"tmin_max_c"`
	VPDMinPa   float64 `json:"vpd_min_pa"`
	VPDMaxPa   float64 `json:"vpd_max_pa"`
	DaylMinSec float64 `json:"dayl_min_sec"`
	DaylMaxSec float64 `json:"dayl_max_sec"`

	// Live fuel moisture map parameters.
	MaxGSI           float64 `json:"max_gsi"`
	GreenupThreshold float64 `json:"greenup_threshold"`
	MinLFM           float64 `json:"min_lfm"`
	MaxLFM           float64 `json:"max_lfm"`
	Slope            float64 `json:"slope"`
	Intercept        float64 `json:"intercept"`

	// Running average window.
	MAPeriod int       `json:"ma_period"`
	GSIQueue []float64 `json:"gsi_queue"`

	// Herbaceous curing state.
	HasGreenedUp   bool    `json:"has_greened_up"`
	HasExceeded120 bool    `json:"has_exceeded_120"`
	CanIncrease    bool    `json:"can_increase"`
	LastHerbFM     float64 `json:"last_herb_fm"`

	// LastUpdateUnix is the time of the previous update, in Unix seconds. It is
	// zero before the first update. The model uses it to drop stale GSI values
	// when a gap in the daily updates occurs.
	LastUpdateUnix int64 `json:"last_update_unix"`

	// CurrentMoisture is the live fuel moisture from the most recent update.
	CurrentMoisture float64 `json:"current_moisture"`
}

// DailyObs is one day of weather for a Model update.
type DailyObs struct {
	// Date is the observation date. The model takes the day of the year and the
	// gap handling from it.
	Date time.Time

	MinTemperature firewx.Celsius
	MaxTemperature firewx.Celsius
	MinHumidity    firewx.Percent

	// SnowCovered suppresses the green-up when set.
	SnowCovered bool
}

// newModel returns a model with the default limits and the default map
// parameters for the fuel type.
func newModel(fuel FuelType, latitude float64, annual bool) *Model {
	m := &Model{
		SchemaVersion:    schemaVersion,
		FuelType:         fuel,
		Latitude:         latitude,
		Annual:           annual,
		TminMinC:         defaultTminMinC,
		TminMaxC:         defaultTminMaxC,
		VPDMinPa:         defaultVPDMinPa,
		VPDMaxPa:         defaultVPDMaxPa,
		DaylMinSec:       defaultDaylMinSec,
		DaylMaxSec:       defaultDaylMaxSec,
		MaxGSI:           defaultMaxGSI,
		GreenupThreshold: defaultGreenupThreshold,
		MAPeriod:         defaultMAPeriod,
		LastHerbFM:       -1.0,
	}
	if fuel == Herbaceous {
		m.MinLFM = defaultHerbMinLFM
		m.MaxLFM = defaultHerbMaxLFM
	} else {
		m.MinLFM = defaultWoodyMinLFM
		m.MaxLFM = defaultWoodyMaxLFM
	}
	m.setDerived()
	return m
}

// NewHerbaceous returns a herbaceous fuel model for a latitude. The annual flag
// sets whether the herbaceous fuel is annual, which cures once per year.
func NewHerbaceous(latitude float64, annual bool) *Model {
	return newModel(Herbaceous, latitude, annual)
}

// NewWoody returns a woody fuel model for a latitude.
func NewWoody(latitude float64) *Model {
	return newModel(Woody, latitude, false)
}

// setDerived computes the slope and the intercept of the live fuel moisture map
// from the current parameters.
func (m *Model) setDerived() {
	if m.GreenupThreshold == 1.0 {
		m.GreenupThreshold = 0.9999
	}
	m.Slope = (m.MaxLFM - m.MinLFM) / (1.0 - m.GreenupThreshold)
	m.Intercept = m.MaxLFM - m.Slope
}

// dailyGSI computes the GSI for one day from the observation.
func (m *Model) dailyGSI(o DailyObs) float64 {
	tmin := float64(o.MinTemperature)
	tMinInd := tminIndicator(tmin, m.TminMinC, m.TminMaxC)

	// The vapor pressure deficit uses the maximum temperature and the minimum
	// humidity, and the humidity has a floor of 5 percent.
	rh := o.MinHumidity
	if rh < 5 {
		rh = 5
	}
	vpd := VaporPressureDeficit(o.MaxTemperature, rh)
	vpdInd := vpdIndicator(vpd, m.VPDMinPa, m.VPDMaxPa)

	dayl := float64(Daylength(m.Latitude, o.Date.UTC().YearDay())) / float64(time.Second)
	daylInd := daylIndicator(dayl, m.DaylMinSec, m.DaylMaxSec)

	return tMinInd * vpdInd * daylInd
}

// Update advances the model by one day. It computes the GSI for the day, adds it
// to the running average window, then computes the live fuel moisture.
func (m *Model) Update(o DailyObs) {
	gsi := m.dailyGSI(o)

	// Drop stale GSI values when a gap in the daily updates occurs.
	now := o.Date.UTC().Unix()
	if m.LastUpdateUnix != 0 {
		days := int((now - m.LastUpdateUnix) / 86400)
		if days > 1 {
			drop := min(days-1, len(m.GSIQueue))
			m.GSIQueue = m.GSIQueue[drop:]
		}
	}
	// The moving average period is at least one day, so the window keeps at
	// least the current value.
	period := max(m.MAPeriod, 1)
	m.GSIQueue = append(m.GSIQueue, gsi)
	for len(m.GSIQueue) > period {
		m.GSIQueue = m.GSIQueue[1:]
	}
	m.LastUpdateUnix = now

	m.CurrentMoisture = m.moisture(o.SnowCovered)
}

// GSI returns the running average of the Growing Season Index. It has a range of
// 0 to 1.
func (m *Model) GSI() float64 {
	if len(m.GSIQueue) == 0 {
		return 0
	}
	var sum float64
	for _, v := range m.GSIQueue {
		sum += v
	}
	return sum / float64(len(m.GSIQueue))
}

// Moisture returns the live fuel moisture content from the most recent update.
func (m *Model) Moisture() firewx.Percent {
	return firewx.Percent(m.CurrentMoisture)
}

// moisture computes the live fuel moisture from the running average GSI and, for
// herbaceous fuel, advances the annual curing state.
func (m *Model) moisture(snow bool) float64 {
	rescale := m.GSI() / m.MaxGSI
	rescale = min(1.0, rescale)
	rescale = max(0.0, rescale)

	if m.FuelType == Woody {
		if rescale >= m.GreenupThreshold && !snow {
			return m.Slope*rescale + m.Intercept
		}
		return m.MinLFM
	}

	// Herbaceous fuel with the annual curing state.
	ret := m.MinLFM
	if rescale >= m.GreenupThreshold && !snow {
		ret = m.Slope*rescale + m.Intercept
		if !m.HasGreenedUp {
			m.HasGreenedUp = true
			m.CanIncrease = true
		}
	}
	if !m.CanIncrease && m.LastHerbFM >= 0 {
		ret = min(ret, m.LastHerbFM)
	}
	if !m.HasExceeded120 && ret >= 120 {
		m.HasExceeded120 = true
	}
	if m.HasExceeded120 && ret < 120.0 && m.Annual {
		m.CanIncrease = false
	}
	m.LastHerbFM = ret
	return ret
}

// ResetAnnual clears the annual curing state of a herbaceous model. Call it at
// the start of a new growing season.
func (m *Model) ResetAnnual() {
	m.HasGreenedUp = false
	m.CanIncrease = false
	m.HasExceeded120 = false
	m.LastHerbFM = -1.0
}
