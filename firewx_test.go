package firewx

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func closeTo(t *testing.T, got, want, tol float64, what string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s: got %v, want %v (tol %v)", what, got, want, tol)
	}
}

func TestTemperatureConversion(t *testing.T) {
	cases := []struct{ c, f float64 }{
		{0, 32}, {100, 212}, {-40, -40}, {37, 98.6},
	}
	for _, tc := range cases {
		closeTo(t, float64(Celsius(tc.c).Fahrenheit()), tc.f, 1e-9, "C->F")
		closeTo(t, float64(Fahrenheit(tc.f).Celsius()), tc.c, 1e-9, "F->C")
	}
}

func TestWindConversionRoundTrip(t *testing.T) {
	v := MetersPerSecond(7.3)
	closeTo(t, float64(v.KilometersPerHour().MetersPerSecond()), 7.3, 1e-9, "m/s->km/h->m/s")
	closeTo(t, float64(v.MilesPerHour().MetersPerSecond()), 7.3, 1e-9, "m/s->mph->m/s")
	closeTo(t, float64(v.KilometersPerHour()), 26.28, 1e-9, "m/s->km/h")
}

func TestPrecipConversion(t *testing.T) {
	closeTo(t, float64(Millimeters(25.4).Inches()), 1.0, 1e-12, "mm->in")
	closeTo(t, float64(Inches(1).Millimeters()), 25.4, 1e-12, "in->mm")
}

func TestDewPointInvertsRelativeHumidity(t *testing.T) {
	for _, temp := range []Celsius{-10, 0, 12.5, 30} {
		for _, rh := range []Percent{15, 45, 90} {
			dew := DewPoint(temp, rh)
			back := RelativeHumidity(temp, dew)
			closeTo(t, float64(back), float64(rh), 1e-6, "RH round trip")
		}
	}
}

func TestVaporPressureDeficitIsZeroAtSaturation(t *testing.T) {
	closeTo(t, float64(VaporPressureDeficit(20, 100)), 0, 1e-12, "VPD at 100% RH")
	if VaporPressureDeficit(30, 20) <= VaporPressureDeficit(20, 20) {
		t.Error("VPD should increase with temperature at fixed RH")
	}
}

func TestOptJSONRoundTrip(t *testing.T) {
	type wrapper struct {
		A Opt[Celsius] `json:"a"`
		B Opt[Celsius] `json:"b"`
	}
	in := wrapper{A: Some(Celsius(21.5)), B: None[Celsius]()}

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"a":21.5,"b":null}` {
		t.Fatalf("unexpected encoding: %s", b)
	}

	var out wrapper
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if v, ok := out.A.Get(); !ok || v != 21.5 {
		t.Errorf("A: got %v %v", v, ok)
	}
	if out.B.Valid() {
		t.Error("B should be absent after round trip")
	}
}

func TestAdjustWindHeight(t *testing.T) {
	// A 3 m backyard sensor over suburban roughness, corrected up to the
	// 10 m exposure the FWI system assumes, must report a higher speed.
	got := AdjustWindHeight(4.0, 3.0, HeightFWI, RoughnessSuburban)
	if got <= 4.0 {
		t.Errorf("correcting upward should increase speed, got %v", got)
	}
	// Identity when heights match.
	if got := AdjustWindHeight(4.0, 10, 10, RoughnessOpenGrass); got != 4.0 {
		t.Errorf("identity case: got %v", got)
	}
}

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
