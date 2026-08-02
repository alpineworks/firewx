package fwi

import (
	"time"

	firewx "alpineworks.io/firewx"
)

// schemaVersion is the on-disk schema version stamped into State. Increase it
// when the meaning of a field changes, so the code can identify an old persisted
// state and change it correctly.
const schemaVersion = 1

// Standard spring start-up values for the three moisture codes (Van Wagner
// 1987). They suit most springtime conditions in Canada, the northern United
// States, and Alaska. They do not suit a very dry winter or other parts of the
// world.
const (
	StartupFFMC FFMC = 85
	StartupDMC  DMC  = 6
	StartupDC   DC   = 15
)

// Weather is one day of noon LST fire weather for the FWI System.
type Weather struct {
	Temperature firewx.Celsius
	Humidity    firewx.Percent
	Wind        firewx.KilometersPerHour
	Rain        firewx.Millimeters

	// Month sets the day-length factor for the Duff Moisture Code and the
	// Drought Code.
	Month time.Month

	// Latitude sets the day-length band, in decimal degrees. It is positive in
	// the northern hemisphere and negative in the southern hemisphere.
	Latitude float64
}

// Output holds the seven FWI System values for one day: the three moisture
// codes, the three derived indices, and the Daily Severity Rating.
type Output struct {
	FFMC FFMC
	DMC  DMC
	DC   DC
	ISI  ISI
	BUI  BUI
	FWI  FWI
	DSR  DSR
}

// State carries the three moisture codes forward from day to day. It is a plain
// struct with exported, JSON-tagged fields, so it can be persisted between daily
// runs and restored exactly. The derived indices are not state; the driver
// computes them fresh each day.
type State struct {
	SchemaVersion int  `json:"schema_version"`
	FFMC          FFMC `json:"ffmc"`
	DMC           DMC  `json:"dmc"`
	DC            DC   `json:"dc"`
}

// NewState returns a state seeded with the standard spring start-up values and
// stamped with the current schema version.
func NewState() State {
	return State{
		SchemaVersion: schemaVersion,
		FFMC:          StartupFFMC,
		DMC:           StartupDMC,
		DC:            StartupDC,
	}
}

// Step advances the three moisture codes by one day and returns the full daily
// output. A relative humidity of 100 percent or more is held at 99.9999, which
// matches the cffdrs R package and avoids the singular case in the codes.
func (s *State) Step(w Weather) Output {
	rh := w.Humidity
	if rh >= 100 {
		rh = 99.9999
	}

	s.FFMC = FineFuelMoistureCode(s.FFMC, w.Temperature, rh, w.Wind, w.Rain)
	s.DMC = DuffMoistureCode(s.DMC, w.Temperature, rh, w.Rain, w.Month, w.Latitude)
	s.DC = DroughtCode(s.DC, w.Temperature, w.Rain, w.Month, w.Latitude)

	isi := InitialSpreadIndex(s.FFMC, w.Wind)
	bui := BuildupIndex(s.DMC, s.DC)
	fwi := FireWeatherIndex(isi, bui)

	return Output{
		FFMC: s.FFMC,
		DMC:  s.DMC,
		DC:   s.DC,
		ISI:  isi,
		BUI:  bui,
		FWI:  fwi,
		DSR:  DailySeverityRating(fwi),
	}
}

// StepObs advances the codes from a firewx.Obs. The month and the latitude come
// from the caller, because the FWI System uses the local standard time date and
// the station latitude. It reports whether the step was applied: an absent
// temperature, humidity, wind, or precipitation leaves the state unchanged and
// returns false, because the system needs all four inputs.
//
// The observation's wind is used as measured and is converted to kilometres per
// hour. It is not corrected for the anemometer height; correct it with
// Obs.WindAt toward the 10 m reference first if the site is non-standard.
func (s *State) StepObs(o firewx.Obs, month time.Month, lat float64) (Output, bool) {
	t, okT := o.Temperature.Get()
	rh, okH := o.RelativeHumidity.Get()
	wind, okW := o.WindSpeed.Get()
	rain, okR := o.Precipitation.Get()
	if !okT || !okH || !okW || !okR {
		return Output{}, false
	}
	return s.Step(Weather{
		Temperature: t,
		Humidity:    rh,
		Wind:        wind.KilometersPerHour(),
		Rain:        rain,
		Month:       month,
		Latitude:    lat,
	}), true
}
