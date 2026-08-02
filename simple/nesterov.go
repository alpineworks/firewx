package simple

import firewx "alpineworks.io/firewx"

// Nesterov is the Nesterov ignition index. It is a cumulative measure of
// dryness. It adds a daily term of temperature and dryness across a run of days
// without significant rain. One wet day empties the total.
//
// Reference: Nesterov, V.G. 1949. The modern description, including the 3 mm
// reset, is in Groisman et al. 2007, Global and Planetary Change 56(3-4).
type Nesterov float64

// Class gives the danger category from the Nesterov regimes. Below 300 is regime
// I, and this maps to low. The limits 300, 1000, 4000, and 10000 give the higher
// regimes up to regime V, which is extreme.
func (n Nesterov) Class() DangerClass {
	switch {
	case n < 300:
		return ClassLow
	case n < 1000:
		return ClassModerate
	case n < 4000:
		return ClassHigh
	case n < 10000:
		return ClassVeryHigh
	default:
		return ClassExtreme
	}
}

// NesterovDaily is the term for one day. It is the temperature multiplied by the
// dew point depression. Both are in degrees Celsius. The driver adds this term
// each day. The function is public so that a caller can sum the term across a
// different calendar.
func NesterovDaily(t, dew firewx.Celsius) Nesterov {
	return Nesterov(float64(t) * (float64(t) - float64(dew)))
}

// nesterovRainReset is the daily precipitation limit in millimetres. Above this
// limit the code empties the total.
const nesterovRainReset firewx.Millimeters = 3

// NesterovState holds the Nesterov index across days. It is a plain struct. Its
// fields are exported and have JSON tags. It goes to and from storage without a
// change, so the daily runs continue the total.
type NesterovState struct {
	SchemaVersion int      `json:"schema_version"`
	Index         Nesterov `json:"index"`
}

// NewNesterovState returns an empty accumulator with the current schema version.
func NewNesterovState() NesterovState {
	return NesterovState{SchemaVersion: schemaVersion}
}

// Step moves the index forward by one day. Precipitation above the reset limit
// empties the total and adds nothing. Precipitation at or below the limit lets
// the code add the day's term of temperature and dryness.
func (s *NesterovState) Step(t, dew firewx.Celsius, precip firewx.Millimeters) {
	if precip > nesterovRainReset {
		s.Index = 0
		return
	}
	s.Index += NesterovDaily(t, dew)
}

// StepObs moves the index forward from an observation. It derives the dew point
// from the temperature and the humidity. It returns true if it applied the step.
// An absent temperature, humidity, or precipitation leaves the state unchanged
// and returns false. An absent precipitation must not become a zero, because a
// zero could carry the index through a wet spell that must reset it.
func (s *NesterovState) StepObs(o firewx.Obs) bool {
	t, okT := o.Temperature.Get()
	dew, okDew := o.DewPoint().Get()
	precip, okP := o.Precipitation.Get()
	if !okT || !okDew || !okP {
		return false
	}
	s.Step(t, dew, precip)
	return true
}
