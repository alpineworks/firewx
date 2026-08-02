package simple

import (
	"encoding/json"
	"testing"

	firewx "alpineworks.io/firewx"
)

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

func TestNesterovResetBoundary(t *testing.T) {
	// Precipitation of exactly 3 mm is at the limit, not above it, so it must
	// NOT reset. This pins the > versus >= choice.
	s := NewNesterovState()
	s.Step(25, 10, 0) // 375
	s.Step(20, 10, 3) // +20*(20-10)=200, no reset
	closeTo(t, float64(s.Index), 575, 1e-9, "Nesterov precip=3 does not reset")
}

func TestNesterovDailyDewAboveTemp(t *testing.T) {
	// When the dew point is at or above the temperature (near saturation) the
	// daily term is zero or negative. The term is not floored.
	if got := NesterovDaily(20, 20); got != 0 {
		t.Errorf("NesterovDaily(20,20)=%v, want 0", got)
	}
	if got := NesterovDaily(20, 25); got != -100 {
		t.Errorf("NesterovDaily(20,25)=%v, want -100", got)
	}
}

func TestNesterovClassBoundaries(t *testing.T) {
	cases := []struct {
		v    Nesterov
		want DangerClass
	}{
		{299, ClassLow}, {300, ClassModerate}, {1000, ClassHigh},
		{4000, ClassVeryHigh}, {10000, ClassExtreme},
	}
	for _, c := range cases {
		if got := c.v.Class(); got != c.want {
			t.Errorf("Nesterov(%v).Class()=%v, want %v", c.v, got, c.want)
		}
	}
}

func TestNesterovStepObs(t *testing.T) {
	// Success path: dew point is derived from temperature and humidity.
	o := firewx.Obs{
		Temperature:      firewx.Some(firewx.Celsius(25)),
		RelativeHumidity: firewx.Some(firewx.Percent(40)),
		Precipitation:    firewx.Some(firewx.Millimeters(0)),
	}
	dew := o.DewPoint().Must()
	s := NewNesterovState()
	if !s.StepObs(o) {
		t.Fatal("StepObs should apply with T, RH and precip present")
	}
	closeTo(t, float64(s.Index), float64(NesterovDaily(25, dew)), 1e-9, "StepObs derived term")

	// Missing temperature must not advance the state.
	s2 := NewNesterovState()
	if s2.StepObs(firewx.Obs{
		RelativeHumidity: firewx.Some(firewx.Percent(40)),
		Precipitation:    firewx.Some(firewx.Millimeters(0)),
	}) {
		t.Error("StepObs should report false without temperature")
	}
}

func TestNesterovStepObsAbsentPrecip(t *testing.T) {
	s := NewNesterovState()
	// Missing precipitation must not advance the state, since assuming zero rain
	// could carry the index through a wet spell it should have reset on.
	o := firewx.Obs{
		Temperature:      firewx.Some(firewx.Celsius(25)),
		RelativeHumidity: firewx.Some(firewx.Percent(40)),
	}
	if s.StepObs(o) {
		t.Error("StepObs should report false without precipitation")
	}
	if s.Index != 0 {
		t.Error("state must be untouched when an input is absent")
	}
}

func TestNesterovStateJSONRoundTrip(t *testing.T) {
	// Property: any state value survives a JSON round trip byte for byte, which
	// the driver relies on for persistence between daily runs.
	states := []NesterovState{
		NewNesterovState(),
		{SchemaVersion: schemaVersion, Index: 375.5},
		{SchemaVersion: schemaVersion, Index: -12.25},
		{SchemaVersion: schemaVersion, Index: 12345.678},
	}
	for _, n := range states {
		var back NesterovState
		b, err := json.Marshal(n)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatal(err)
		}
		if back != n {
			t.Errorf("NesterovState round trip: got %+v, want %+v", back, n)
		}
	}
}

func TestNesterovStateSchemaVersionIsReadable(t *testing.T) {
	// A state written by an older binary carries a different schema version. The
	// field must decode faithfully so a caller can detect the mismatch.
	var n NesterovState
	if err := json.Unmarshal([]byte(`{"schema_version":0,"index":5}`), &n); err != nil {
		t.Fatal(err)
	}
	if n.SchemaVersion != 0 {
		t.Errorf("schema version should decode to 0, got %d", n.SchemaVersion)
	}
	if NewNesterovState().SchemaVersion != schemaVersion {
		t.Errorf("constructor should stamp schema version %d", schemaVersion)
	}
}
