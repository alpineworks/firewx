package simple

import (
	"math"

	firewx "alpineworks.io/firewx"
)

// KBDI is the Keetch-Byram Drought Index. It is a soil moisture deficit. The
// unit is hundredths of an inch of water. The scale is 0 to 800. Zero is a
// saturated soil. 800 is the maximum deficit. The index drives the heavy dead
// fuels. In NFDRS it drives the drought fuel loading.
//
// Reference: Keetch and Byram 1968, Res. Pap. SE-38.
type KBDI float64

// KBDIMax is the upper limit of the index. The unit is hundredths of an inch of
// soil moisture deficit.
const KBDIMax KBDI = 800

// Class gives the danger category. The limits are: below 200 is low, 200 to 400
// is moderate, 400 to 600 is high, and 600 or more is extreme.
func (q KBDI) Class() DangerClass {
	switch {
	case q < 200:
		return ClassLow
	case q < 400:
		return ClassModerate
	case q < 600:
		return ClassHigh
	default:
		return ClassExtreme
	}
}

// KBDIIncrement gives the drought factor. This is the rise of the index in one
// day. The inputs are the current value q, the daily maximum temperature in
// degrees Fahrenheit, and the mean annual rainfall of the station in inches.
//
// The equation uses the corrected constant 8.30. The original Keetch and Byram
// (1968) paper printed 0.083, and this makes the factor too large. Alexander
// 1990 (Fire Management Notes 51(4):23-25) gives the correction. The code limits
// the evapotranspiration term to zero or more. Therefore a cool day does not add
// drought and does not remove drought. Only rain lowers the index.
func KBDIIncrement(q KBDI, maxT firewx.Fahrenheit, meanAnnualRain firewx.Inches) KBDI {
	et := 0.968*math.Exp(0.0486*float64(maxT)) - 8.30
	if et < 0 {
		et = 0
	}
	num := (float64(KBDIMax) - float64(q)) * et
	den := 1 + 10.88*math.Exp(-0.0441*float64(meanAnnualRain))
	return KBDI(num / den * 1e-3)
}

// kbdiInterception is the rain in inches that the canopy holds before any rain
// reaches the soil. The code applies it one time for each run of wet days, not
// one time each day.
const kbdiInterception firewx.Inches = 0.20

// KBDIState holds the drought index across days. Its fields are exported and
// have JSON tags for exact storage between daily runs.
type KBDIState struct {
	SchemaVersion int `json:"schema_version"`

	// Index is the current drought index, 0 to 800.
	Index KBDI `json:"index"`

	// MeanAnnualRain is the mean annual rainfall of the station in inches. It is
	// a fixed parameter of the evapotranspiration term. The struct holds it so
	// that a restored state has all that Step needs.
	MeanAnnualRain firewx.Inches `json:"mean_annual_rain"`

	// WetSpellRain is the rain that has fallen in the current run of wet days.
	// The code subtracts the 0.20 inch interception from this total. Therefore
	// the code applies the interception one time at the start of a wet spell, not
	// each day. A dry day sets this value to zero.
	WetSpellRain firewx.Inches `json:"wet_spell_rain"`
}

// NewKBDIState returns a state for a station with the given mean annual rainfall.
// It sets the current schema version. The index starts at zero. To start from a
// known value, set Index directly.
func NewKBDIState(meanAnnualRain firewx.Inches) KBDIState {
	return KBDIState{
		SchemaVersion:  schemaVersion,
		MeanAnnualRain: meanAnnualRain,
	}
}

// Step moves the index forward by one day. The inputs are the daily maximum
// temperature and the total rain of the day. The code applies the rain first.
// The rain lowers the index by the net amount after interception. Then the code
// adds the drought factor for the lower index. The code keeps the result between
// 0 and 800.
func (s *KBDIState) Step(maxT firewx.Fahrenheit, rain firewx.Inches) {
	if rain > 0 {
		// Subtract the interception from the rain that the wet spell already has.
		// Therefore the code removes the 0.20 inch one time for each wet spell.
		remainingInterception := kbdiInterception - s.WetSpellRain
		if remainingInterception < 0 {
			remainingInterception = 0
		}
		net := rain - remainingInterception
		if net < 0 {
			net = 0
		}
		s.WetSpellRain += rain
		s.Index -= KBDI(net * 100)
		if s.Index < 0 {
			s.Index = 0
		}
	} else {
		// A dry day ends the wet spell.
		s.WetSpellRain = 0
	}

	s.Index += KBDIIncrement(s.Index, maxT, s.MeanAnnualRain)
	if s.Index > KBDIMax {
		s.Index = KBDIMax
	}
}

// StepObs moves the index forward from an observation. It returns true if it
// applied the step. An absent maximum temperature or precipitation leaves the
// state unchanged and returns false. An absent rain must not become a zero,
// because a drought could deepen through rain that the station did not see.
//
// KBDI uses the daily maximum temperature. One observation's temperature is the
// daily maximum only if that observation is the daytime peak. Give Step the true
// daily maximum where you have one.
func (s *KBDIState) StepObs(o firewx.Obs) bool {
	t, okT := o.Temperature.Get()
	rain, okR := o.Precipitation.Get()
	if !okT || !okR {
		return false
	}
	s.Step(t.Fahrenheit(), rain.Inches())
	return true
}
