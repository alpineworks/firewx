package simple

import (
	"encoding/json"
	"fmt"
	"testing"

	firewx "alpineworks.io/firewx"
)

func TestKBDIIncrement(t *testing.T) {
	cases := []struct {
		name           string
		q              KBDI
		maxT           firewx.Fahrenheit
		meanAnnualRain firewx.Inches
		want           float64
		tol            float64
	}{
		// From a saturated soil (Q=0), 90F with 50 in mean annual rainfall raises
		// the index by about 24.92 hundredths of an inch. Recomputed from the
		// corrected Keetch-Byram (1968) equation.
		{"saturated soil, 90F, 50in", 0, 90, 50, 24.92, 0.05},
		// A cool day floors the evapotranspiration term at zero: no drought added.
		{"cool day adds nothing", 400, 40, 50, 0, 1e-9},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closeTo(t, float64(KBDIIncrement(tc.q, tc.maxT, tc.meanAnnualRain)), tc.want, tc.tol, tc.name)
		})
	}
}

func TestKBDIRainInterceptionOncePerSpell(t *testing.T) {
	// 40F clamps the evapotranspiration term to zero, isolating the rain
	// bookkeeping: the 0.20 in interception is charged once per wet spell.
	steps := []struct {
		name string
		rain firewx.Inches
		want float64
	}{
		{"first wet day, net 0.30", 0.5, 370},
		{"second wet day, interception spent, net 0.50", 0.5, 320},
		{"dry cool day ends the spell", 0, 320},
		{"sub-threshold rain wets nothing", 0.1, 320},
	}
	s := NewKBDIState(50)
	s.Index = 400
	for _, st := range steps {
		s.Step(40, st.rain)
		t.Run(st.name, func(t *testing.T) {
			closeTo(t, float64(s.Index), st.want, 1e-9, st.name)
		})
	}
}

func TestKBDIBounds(t *testing.T) {
	cases := []struct {
		name  string
		start KBDI
		maxT  firewx.Fahrenheit
		rain  firewx.Inches
		want  float64
	}{
		{"heavy rain floors at zero", 100, 40, 2.0, 0},     // net 1.8 -> -180
		{"holds at the 800 ceiling", KBDIMax, 110, 0, 800}, // (800-Q) factor is zero
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewKBDIState(50)
			s.Index = tc.start
			s.Step(tc.maxT, tc.rain)
			closeTo(t, float64(s.Index), tc.want, 1e-9, tc.name)
		})
	}
}

func TestKBDIApproachesCeiling(t *testing.T) {
	// Sustained heat drives the index up toward, but never past, 800. The
	// (800-Q) factor shrinks the daily rise, so the approach is asymptotic and
	// cannot be expressed as a fixed table of expected values.
	s := NewKBDIState(50)
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

func TestKBDIClass(t *testing.T) {
	cases := []struct {
		v    KBDI
		want DangerClass
	}{
		{199, ClassLow}, {200, ClassModerate}, {399, ClassModerate},
		{400, ClassHigh}, {599, ClassHigh}, {600, ClassExtreme}, {800, ClassExtreme},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("Q=%v", tc.v), func(t *testing.T) {
			if got := tc.v.Class(); got != tc.want {
				t.Errorf("KBDI(%v).Class()=%v, want %v", tc.v, got, tc.want)
			}
		})
	}
}

func TestKBDIStepObs(t *testing.T) {
	// KBDI uses the daily maximum temperature. An absent temperature or
	// precipitation must leave the state untouched.
	cases := []struct {
		name        string
		obs         firewx.Obs
		start       KBDI
		wantApplied bool
		want, tol   float64
	}{
		{
			name:        "applies with temperature and precip",
			obs:         firewx.Obs{Temperature: firewx.Some(firewx.Celsius(32.2222)), Precipitation: firewx.Some(firewx.Millimeters(0))}, // 90F
			start:       0,
			wantApplied: true,
			want:        24.92,
			tol:         0.05,
		},
		{
			name:        "skips without precipitation",
			obs:         firewx.Obs{Temperature: firewx.Some(firewx.Celsius(30))},
			start:       300,
			wantApplied: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewKBDIState(50)
			s.Index = tc.start
			applied := s.StepObs(tc.obs)
			if applied != tc.wantApplied {
				t.Fatalf("%s: applied=%v, want %v", tc.name, applied, tc.wantApplied)
			}
			if tc.wantApplied {
				closeTo(t, float64(s.Index), tc.want, tc.tol, tc.name)
			} else if s.Index != tc.start {
				t.Errorf("%s: state must be untouched, got %v", tc.name, s.Index)
			}
		})
	}
}

func TestKBDIStateJSONRoundTrip(t *testing.T) {
	// Property: any state value survives a JSON round trip byte for byte, which
	// the driver relies on for persistence between daily runs.
	cases := []struct {
		name  string
		state KBDIState
	}{
		{"zero value", NewKBDIState(50)},
		{"mid range with wet spell", KBDIState{SchemaVersion: schemaVersion, Index: 412.5, MeanAnnualRain: 33.75, WetSpellRain: 0.15}},
		{"at the ceiling", KBDIState{SchemaVersion: schemaVersion, Index: 800, MeanAnnualRain: 60.125, WetSpellRain: 1.2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.state)
			if err != nil {
				t.Fatal(err)
			}
			var back KBDIState
			if err := json.Unmarshal(b, &back); err != nil {
				t.Fatal(err)
			}
			if back != tc.state {
				t.Errorf("round trip: got %+v, want %+v", back, tc.state)
			}
		})
	}
}
