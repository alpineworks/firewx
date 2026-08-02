package simple

import (
	"encoding/json"
	"fmt"
	"testing"

	firewx "alpineworks.io/firewx"
)

func TestNesterovSteps(t *testing.T) {
	// A single run of days accumulates, then a wet day empties the total. Each
	// row applies one day and states the expected running index.
	steps := []struct {
		name   string
		t, dew firewx.Celsius
		precip firewx.Millimeters
		want   float64
	}{
		{"day 1", 25, 10, 0, 375},                     // 25*(25-10)
		{"day 2, precip below reset", 30, 12, 1, 915}, // +30*(30-12)
		{"rain over 3mm resets", 28, 11, 5, 0},
	}
	s := NewNesterovState()
	for _, st := range steps {
		s.Step(st.t, st.dew, st.precip)
		t.Run(st.name, func(t *testing.T) {
			closeTo(t, float64(s.Index), st.want, 1e-9, st.name)
		})
	}
}

func TestNesterovResetBoundary(t *testing.T) {
	// Precipitation of exactly 3 mm is at the limit, not above it, so it must NOT
	// reset. This pins the > versus >= choice.
	steps := []struct {
		name   string
		t, dew firewx.Celsius
		precip firewx.Millimeters
		want   float64
	}{
		{"day 1", 25, 10, 0, 375},
		{"precip=3 does not reset", 20, 10, 3, 575}, // +20*(20-10)
	}
	s := NewNesterovState()
	for _, st := range steps {
		s.Step(st.t, st.dew, st.precip)
		t.Run(st.name, func(t *testing.T) {
			closeTo(t, float64(s.Index), st.want, 1e-9, st.name)
		})
	}
}

func TestNesterovDaily(t *testing.T) {
	// The daily term is not floored, so a dew point at or above the temperature
	// gives a zero or negative term.
	cases := []struct {
		name   string
		t, dew firewx.Celsius
		want   float64
	}{
		{"dew equals temperature", 20, 20, 0},
		{"dew above temperature", 20, 25, -100},
		{"normal drying", 25, 10, 375},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closeTo(t, float64(NesterovDaily(tc.t, tc.dew)), tc.want, 1e-9, tc.name)
		})
	}
}

func TestNesterovClass(t *testing.T) {
	cases := []struct {
		v    Nesterov
		want DangerClass
	}{
		{299, ClassLow}, {300, ClassModerate}, {1000, ClassHigh},
		{4000, ClassVeryHigh}, {10000, ClassExtreme},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("NI=%v", tc.v), func(t *testing.T) {
			if got := tc.v.Class(); got != tc.want {
				t.Errorf("Nesterov(%v).Class()=%v, want %v", tc.v, got, tc.want)
			}
		})
	}
}

func TestNesterovStepObs(t *testing.T) {
	// The success case derives the dew point from temperature and humidity. An
	// absent temperature or precipitation must leave the state untouched, because
	// assuming a value could carry the index through a wet spell it should reset.
	full := firewx.Obs{
		Temperature:      firewx.Some(firewx.Celsius(25)),
		RelativeHumidity: firewx.Some(firewx.Percent(40)),
		Precipitation:    firewx.Some(firewx.Millimeters(0)),
	}
	wantTerm := float64(NesterovDaily(25, full.DewPoint().Must()))
	cases := []struct {
		name        string
		obs         firewx.Obs
		wantApplied bool
		want        float64
	}{
		{"applies with T, RH and precip", full, true, wantTerm},
		{
			name: "skips without temperature",
			obs: firewx.Obs{
				RelativeHumidity: firewx.Some(firewx.Percent(40)),
				Precipitation:    firewx.Some(firewx.Millimeters(0)),
			},
		},
		{
			name: "skips without precipitation",
			obs: firewx.Obs{
				Temperature:      firewx.Some(firewx.Celsius(25)),
				RelativeHumidity: firewx.Some(firewx.Percent(40)),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewNesterovState()
			applied := s.StepObs(tc.obs)
			if applied != tc.wantApplied {
				t.Fatalf("%s: applied=%v, want %v", tc.name, applied, tc.wantApplied)
			}
			if tc.wantApplied {
				closeTo(t, float64(s.Index), tc.want, 1e-9, tc.name)
			} else if s.Index != 0 {
				t.Errorf("%s: state must be untouched, got %v", tc.name, s.Index)
			}
		})
	}
}

func TestNesterovStateJSONRoundTrip(t *testing.T) {
	// Property: any state value survives a JSON round trip byte for byte, which
	// the driver relies on for persistence between daily runs.
	cases := []struct {
		name  string
		state NesterovState
	}{
		{"zero value", NewNesterovState()},
		{"positive index", NesterovState{SchemaVersion: schemaVersion, Index: 375.5}},
		{"negative index", NesterovState{SchemaVersion: schemaVersion, Index: -12.25}},
		{"large index", NesterovState{SchemaVersion: schemaVersion, Index: 12345.678}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.state)
			if err != nil {
				t.Fatal(err)
			}
			var back NesterovState
			if err := json.Unmarshal(b, &back); err != nil {
				t.Fatal(err)
			}
			if back != tc.state {
				t.Errorf("round trip: got %+v, want %+v", back, tc.state)
			}
		})
	}
}

func TestNesterovStateSchemaVersion(t *testing.T) {
	// A state written by an older binary must decode its own version so a caller
	// can detect the mismatch, and the constructor must stamp the current one.
	var old NesterovState
	if err := json.Unmarshal([]byte(`{"schema_version":0,"index":5}`), &old); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name      string
		got, want int
	}{
		{"old state decodes its version", old.SchemaVersion, 0},
		{"constructor stamps current version", NewNesterovState().SchemaVersion, schemaVersion},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("%s: got %d, want %d", tc.name, tc.got, tc.want)
			}
		})
	}
}
