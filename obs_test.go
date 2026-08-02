package firewx

import (
	"testing"
	"time"
)

func TestLocalStandardTimeIgnoresDST(t *testing.T) {
	s := Station{LSTOffset: -8 * time.Hour} // Pacific standard, year round

	summer := time.Date(2026, 7, 15, 20, 0, 0, 0, time.UTC)
	winter := time.Date(2026, 1, 15, 20, 0, 0, 0, time.UTC)

	if h := s.LocalStandardTime(summer).Hour(); h != 12 {
		t.Errorf("summer: got hour %d, want 12", h)
	}
	if h := s.LocalStandardTime(winter).Hour(); h != 12 {
		t.Errorf("winter: got hour %d, want 12", h)
	}
}

func TestObsDerivedValuesPropagateAbsence(t *testing.T) {
	o := Obs{Temperature: Some(Celsius(20))} // no RH
	if o.DewPoint().Valid() {
		t.Error("dew point should be absent without RH")
	}
	if o.VaporPressureDeficit().Valid() {
		t.Error("VPD should be absent without RH")
	}
}

func TestObsDewPointAbsentWhenUndefined(t *testing.T) {
	// DewPoint is undefined at zero humidity and the bare function returns NaN.
	// Obs.DewPoint must report absence, never a present Opt holding NaN.
	o := Obs{Temperature: Some(Celsius(20)), RelativeHumidity: Some(Percent(0))}
	if got := o.DewPoint(); got.Valid() {
		v, _ := got.Get()
		t.Errorf("dew point should be absent at 0%% RH, got present %v", v)
	}
}
