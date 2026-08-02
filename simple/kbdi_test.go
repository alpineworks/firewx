package simple

import (
	"encoding/json"
	"testing"

	firewx "alpineworks.io/firewx"
)

func TestKBDIIncrementGolden(t *testing.T) {
	// From a saturated soil (Q=0), one day at 90F with 50 in mean annual
	// rainfall raises the index by about 24.92 hundredths of an inch. Recomputed
	// from the corrected Keetch-Byram (1968) equation.
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

	// At the ceiling the (800-Q) factor is zero, so a hot day adds nothing and
	// the index holds at 800. This exercises the upper bound directly.
	s.Index = KBDIMax
	s.Step(110, 0)
	closeTo(t, float64(s.Index), float64(KBDIMax), 1e-9, "KBDI holds at 800")

	// Sustained heat drives the index up toward, but never past, 800.
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

func TestKBDIClassBoundaries(t *testing.T) {
	cases := []struct {
		v    KBDI
		want DangerClass
	}{
		{199, ClassLow}, {200, ClassModerate}, {399, ClassModerate},
		{400, ClassHigh}, {599, ClassHigh}, {600, ClassExtreme}, {800, ClassExtreme},
	}
	for _, c := range cases {
		if got := c.v.Class(); got != c.want {
			t.Errorf("KBDI(%v).Class()=%v, want %v", c.v, got, c.want)
		}
	}
}

func TestKBDIStepObs(t *testing.T) {
	// Success path: 90F, no rain, from a saturated soil.
	s := NewKBDIState(50)
	o := firewx.Obs{
		Temperature:   firewx.Some(firewx.Celsius(32.2222)), // 90F
		Precipitation: firewx.Some(firewx.Millimeters(0)),
	}
	if !s.StepObs(o) {
		t.Fatal("StepObs should apply with temperature and precip present")
	}
	closeTo(t, float64(s.Index), 24.92, 0.05, "KBDI StepObs increment")

	// Absent rain must not advance the state.
	s2 := NewKBDIState(50)
	s2.Index = 300
	if s2.StepObs(firewx.Obs{Temperature: firewx.Some(firewx.Celsius(30))}) {
		t.Error("StepObs should report false without precipitation")
	}
	if s2.Index != 300 {
		t.Error("state must be untouched when precipitation is absent")
	}
}

func TestKBDIStateJSONRoundTrip(t *testing.T) {
	// Property: any state value survives a JSON round trip byte for byte, which
	// the driver relies on for persistence between daily runs.
	states := []KBDIState{
		NewKBDIState(50),
		{SchemaVersion: schemaVersion, Index: 412.5, MeanAnnualRain: 33.75, WetSpellRain: 0.15},
		{SchemaVersion: schemaVersion, Index: 800, MeanAnnualRain: 60.125, WetSpellRain: 1.2},
	}
	for _, k := range states {
		var back KBDIState
		b, err := json.Marshal(k)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatal(err)
		}
		if back != k {
			t.Errorf("KBDIState round trip: got %+v, want %+v", back, k)
		}
	}
}
