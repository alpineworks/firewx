package simple

import (
	"encoding/json"
	"math"
	"testing"

	firewx "github.com/alpineworks/firewx"
)

func closeTo(t *testing.T, got, want, tol float64, what string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s: got %v, want %v (tol %v)", what, got, want, tol)
	}
}

// --- Fosberg ------------------------------------------------------------

func TestFosbergGolden(t *testing.T) {
	// T=85F, RH=20%, wind=15mph. Worked through the piecewise EMC (RH in the
	// 10-50 band), the moisture damping cubic, and the wind term by hand.
	got := FosbergIndex(85, 20, 15)
	closeTo(t, float64(got), 37.53, 0.05, "FFWI(85F,20%,15mph)")
}

func TestFosbergCalibration(t *testing.T) {
	// The index is defined so that dry fuel (EMC clamped to 0 via very low RH
	// and high temperature) in a 30 mph wind approaches the extreme value 100.
	got := FosbergIndex(110, 0, 30)
	if got < 95 || got > 100.5 {
		t.Errorf("calibration point: got %v, want near 100", got)
	}
}

func TestFosbergMonotonic(t *testing.T) {
	// Drier air raises the index; stronger wind raises the index.
	if FosbergIndex(85, 15, 10) <= FosbergIndex(85, 40, 10) {
		t.Error("FFWI should rise as RH falls")
	}
	if FosbergIndex(85, 20, 20) <= FosbergIndex(85, 20, 10) {
		t.Error("FFWI should rise as wind rises")
	}
}

func TestFosbergFromObsPropagatesAbsence(t *testing.T) {
	o := firewx.Obs{
		Temperature:      firewx.Some(firewx.Celsius(29.4)), // ~85F
		RelativeHumidity: firewx.Some(firewx.Percent(20)),
		// no wind
	}
	if FosbergFromObs(o).Valid() {
		t.Error("Fosberg should be absent without wind")
	}
	o.WindSpeed = firewx.Some(firewx.MetersPerSecond(6.7)) // ~15mph
	if !FosbergFromObs(o).Valid() {
		t.Error("Fosberg should be present with all inputs")
	}
}

// --- Chandler -----------------------------------------------------------

func TestChandlerGolden(t *testing.T) {
	// T=25C, RH=40%. Hand-computed against the modified CBI equation.
	got := ChandlerIndex(25, 40)
	closeTo(t, float64(got), 35.25, 0.05, "CBI(25C,40%)")
}

func TestChandlerMonotonic(t *testing.T) {
	if ChandlerIndex(25, 15) <= ChandlerIndex(25, 60) {
		t.Error("CBI should rise as RH falls")
	}
}

// --- Angstrom -----------------------------------------------------------

func TestAngstromGolden(t *testing.T) {
	got := AngstromIndex(25, 40)
	closeTo(t, float64(got), 2.2, 1e-9, "Angstrom(25C,40%)")
	if got.Class() != ClassHigh {
		t.Errorf("Angstrom 2.2 should classify High, got %v", got.Class())
	}
}

func TestAngstromInverted(t *testing.T) {
	// Lower index means greater danger, so hotter/drier lowers it.
	if AngstromIndex(35, 20) >= AngstromIndex(15, 80) {
		t.Error("Angstrom should fall as conditions worsen")
	}
}

// --- Hot-Dry-Windy ------------------------------------------------------

func TestHDWGolden(t *testing.T) {
	// T=25C, RH=40%, wind=5 m/s. VPD ~19.01 hPa, product ~95.04.
	got := HDWIndex(firewx.Hectopascals(19.00893), 5)
	closeTo(t, float64(got), 95.04, 0.05, "HDW pure")

	o := firewx.Obs{
		Temperature:      firewx.Some(firewx.Celsius(25)),
		RelativeHumidity: firewx.Some(firewx.Percent(40)),
		WindSpeed:        firewx.Some(firewx.MetersPerSecond(5)),
	}
	fromObs, ok := HDWFromObs(o).Get()
	if !ok {
		t.Fatal("HDW should be present")
	}
	closeTo(t, float64(fromObs), 95.04, 0.05, "HDW from obs")
}

func TestHDWAbsentWithoutHumidity(t *testing.T) {
	o := firewx.Obs{
		Temperature: firewx.Some(firewx.Celsius(25)),
		WindSpeed:   firewx.Some(firewx.MetersPerSecond(5)),
	}
	if HDWFromObs(o).Valid() {
		t.Error("HDW should be absent without the humidity needed for VPD")
	}
}

// --- Nesterov -----------------------------------------------------------

func TestNesterovAccumulatesAndResets(t *testing.T) {
	s := NewNesterovState()
	s.Step(25, 10, 0) // 25*(25-10) = 375
	closeTo(t, float64(s.Index), 375, 1e-9, "Nesterov day 1")
	s.Step(30, 12, 1) // +30*(30-12) = 540, precip below reset
	closeTo(t, float64(s.Index), 915, 1e-9, "Nesterov day 2")
	if s.Index.Class() != ClassModerate {
		t.Errorf("Nesterov 915 should classify Moderate, got %v", s.Index.Class())
	}
	s.Step(28, 11, 5) // precip > 3mm resets to zero
	closeTo(t, float64(s.Index), 0, 1e-9, "Nesterov reset on rain")
}

func TestNesterovStepObsAbsence(t *testing.T) {
	s := NewNesterovState()
	// Missing precipitation must not advance the state, since assuming zero
	// rain could carry the index through a wet spell it should have reset on.
	o := firewx.Obs{
		Temperature:      firewx.Some(firewx.Celsius(25)),
		RelativeHumidity: firewx.Some(firewx.Percent(40)),
	}
	if s.StepObs(o) {
		t.Error("StepObs should report false without precipitation")
	}
	if s.Index != 0 {
		t.Error("state must be untouched when an input is missing")
	}
}

// --- KBDI ---------------------------------------------------------------

func TestKBDIIncrementGolden(t *testing.T) {
	// From a saturated soil (Q=0), one day at 90F with 50 in mean annual
	// rainfall raises the index by about 24.92 hundredths of an inch.
	s := NewKBDIState(50)
	s.Step(90, 0)
	closeTo(t, float64(s.Index), 24.92, 0.05, "KBDI increment from 0")
}

func TestKBDIRainInterceptionOncePerSpell(t *testing.T) {
	// 40F clamps the evapotranspiration term to zero, isolating the rain
	// bookkeeping: the 0.20 in interception is charged once per wet spell.
	s := NewKBDIState(50)
	s.Index = 400

	s.Step(40, 0.5) // net 0.30 -> -30
	closeTo(t, float64(s.Index), 370, 1e-9, "KBDI after first wet day")
	s.Step(40, 0.5) // consecutive: interception already spent, net 0.50 -> -50
	closeTo(t, float64(s.Index), 320, 1e-9, "KBDI after second wet day")
	s.Step(40, 0) // dry day ends the spell, no drought added
	closeTo(t, float64(s.Index), 320, 1e-9, "KBDI unchanged on dry cool day")
	s.Step(40, 0.1) // sub-threshold rain wets nothing
	closeTo(t, float64(s.Index), 320, 1e-9, "KBDI unchanged on sub-threshold rain")
}

func TestKBDIStaysInRange(t *testing.T) {
	// Heavy rain floors at zero.
	s := NewKBDIState(50)
	s.Index = 100
	s.Step(40, 2.0) // net 1.8 -> -180, floored at 0
	closeTo(t, float64(s.Index), 0, 1e-9, "KBDI floored at zero")

	// Sustained heat drives the index up toward, but never past, 800. The
	// (800-Q) factor shrinks the daily rise as Q climbs, so the approach is
	// asymptotic; the invariant is that it stays within the bound.
	s.Index = 790
	for range 60 {
		s.Step(110, 0)
	}
	if s.Index > KBDIMax {
		t.Errorf("KBDI exceeded max: %v", s.Index)
	}
	if s.Index <= 790 {
		t.Errorf("KBDI should climb under sustained heat, got %v", s.Index)
	}
}

// --- State persistence --------------------------------------------------

func TestStateJSONRoundTrip(t *testing.T) {
	n := NewNesterovState()
	n.Step(25, 10, 0)
	n.Step(30, 12, 1)
	var nBack NesterovState
	if b, err := json.Marshal(n); err != nil {
		t.Fatal(err)
	} else if err := json.Unmarshal(b, &nBack); err != nil {
		t.Fatal(err)
	}
	if nBack != n {
		t.Errorf("Nesterov state did not round-trip: got %+v, want %+v", nBack, n)
	}

	k := NewKBDIState(50)
	k.Step(90, 0.5)
	var kBack KBDIState
	if b, err := json.Marshal(k); err != nil {
		t.Fatal(err)
	} else if err := json.Unmarshal(b, &kBack); err != nil {
		t.Fatal(err)
	}
	if kBack != k {
		t.Errorf("KBDI state did not round-trip: got %+v, want %+v", kBack, k)
	}
	if kBack.SchemaVersion != schemaVersion {
		t.Errorf("schema version lost: got %d, want %d", kBack.SchemaVersion, schemaVersion)
	}
}
