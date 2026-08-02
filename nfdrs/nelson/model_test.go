package nelson

import (
	"encoding/csv"
	"encoding/json"
	"math"
	"os"
	"strconv"
	"testing"
	"time"

	firewx "alpineworks.io/firewx"
)

// TestNewStandardStick checks the construction invariants of the four standard
// sticks. The expected values follow the initDeadFuelMoisture factory methods
// in firelab/NFDRS4: the maximum local moisture is 0.35, the initial moisture
// is half of that, and the fiber saturation point starts 0.1 above the initial
// moisture.
func TestNewStandardStick(t *testing.T) {
	cases := []struct {
		name string
		tl   TimeLag
		stca float64
	}{
		{"1-hour", OneHour, 0.462252733},
		{"10-hour", TenHour, 0.079548303},
		{"100-hour", HundredHour, 0.06},
		{"1000-hour", ThousandHour, 0.06},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewStandardStick(tc.tl)
			if s.SchemaVersion != schemaVersion {
				t.Errorf("schema version: got %d, want %d", s.SchemaVersion, schemaVersion)
			}
			if s.Nodes != 11 {
				t.Errorf("nodes: got %d, want 11", s.Nodes)
			}
			closeTo(t, s.Wmx, 0.35, 1e-9, "max local moisture")
			closeTo(t, s.Stca, tc.stca, 1e-9, "adsorption rate")
			closeTo(t, s.Stcd, 0.06, 1e-9, "desorption rate")
			// Initial moisture is 0.5 * Wmx = 0.175 g/g at every node.
			for _, w := range s.W {
				closeTo(t, w, 0.175, 1e-12, "initial nodal moisture")
			}
			closeTo(t, float64(s.MoistureContent()), 17.5, 1e-6, "initial mean moisture")
			// Fiber saturation starts at wi + 0.1 = 0.275 g/g.
			closeTo(t, s.Wsa, 0.275, 1e-9, "initial fiber saturation")
			// Maximum possible moisture is 1/density - 1/1.53.
			closeTo(t, s.Wmax, (1.0/0.400)-(1.0/1.53), 1e-9, "max possible moisture")
			// The radial nodes run from the surface (radius) to the centre (0).
			closeTo(t, s.X[0], s.Radius, 1e-12, "surface node radius")
			closeTo(t, s.X[s.Nodes-1], 0.0, 1e-12, "centre node radius")
		})
	}
}

// TestJSONRoundTrip checks that a stick marshals to JSON and back with an exact
// round trip. The repository requires that the radial nodes survive exactly,
// because the state is persisted between runs. The test drives a stick, saves
// it, restores it into a second stick, then drives both sticks with the same
// weather. The two sticks must agree exactly at every node.
func TestJSONRoundTrip(t *testing.T) {
	for _, tl := range []TimeLag{OneHour, TenHour, HundredHour, ThousandHour} {
		t.Run(NewStandardStick(tl).Name, func(t *testing.T) {
			original := NewStandardStick(tl)
			// Drive the stick through a range of conditions, including rain.
			warmup := []Weather{
				{Elapsed: time.Hour, Temperature: 25, RelativeHumidity: 30, SolarRadiation: 400},
				{Elapsed: time.Hour, Temperature: 12, RelativeHumidity: 95, SolarRadiation: 0, Rainfall: 8},
				{Elapsed: time.Hour, Temperature: 18, RelativeHumidity: 60, SolarRadiation: 150},
			}
			for _, w := range warmup {
				original.Update(w)
			}

			// Marshal, then restore into a second stick.
			blob, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var restored Stick
			if err := json.Unmarshal(blob, &restored); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			// A second marshal must be byte-identical to the first.
			blob2, err := json.Marshal(&restored)
			if err != nil {
				t.Fatalf("re-marshal: %v", err)
			}
			if string(blob) != string(blob2) {
				t.Fatalf("re-marshalled JSON differs from the original")
			}

			// Drive both sticks with the same further weather. They must stay
			// identical at every node.
			further := []Weather{
				{Elapsed: time.Hour, Temperature: 22, RelativeHumidity: 40, SolarRadiation: 600},
				{Elapsed: time.Hour, Temperature: 20, RelativeHumidity: 55, SolarRadiation: 200},
				{Elapsed: time.Hour, Temperature: 8, RelativeHumidity: 98, SolarRadiation: 0, Rainfall: 15},
				{Elapsed: time.Hour, Temperature: 15, RelativeHumidity: 70, SolarRadiation: 100},
			}
			for step, w := range further {
				original.Update(w)
				restored.Update(w)
				for i := 0; i < original.Nodes; i++ {
					if original.W[i] != restored.W[i] {
						t.Fatalf("step %d node %d moisture: original %v, restored %v", step, i, original.W[i], restored.W[i])
					}
					if original.T[i] != restored.T[i] {
						t.Fatalf("step %d node %d temperature: original %v, restored %v", step, i, original.T[i], restored.T[i])
					}
					if original.S[i] != restored.S[i] {
						t.Fatalf("step %d node %d saturation: original %v, restored %v", step, i, original.S[i], restored.S[i])
					}
					if original.D[i] != restored.D[i] {
						t.Fatalf("step %d node %d diffusivity: original %v, restored %v", step, i, original.D[i], restored.D[i])
					}
				}
				if original.MoistureContent() != restored.MoistureContent() {
					t.Fatalf("step %d mean moisture: original %v, restored %v", step, original.MoistureContent(), restored.MoistureContent())
				}
			}
		})
	}
}

// TestMoistureStaysInRange checks that the mean moisture content stays between
// zero and the maximum local moisture plus the water film, for a long run over
// a wide range of conditions. This is one of the property tests that the
// repository asks for: the moisture codes stay in range.
func TestMoistureStaysInRange(t *testing.T) {
	for _, tl := range []TimeLag{OneHour, TenHour, HundredHour, ThousandHour} {
		t.Run(NewStandardStick(tl).Name, func(t *testing.T) {
			s := NewStandardStick(tl)
			// A deterministic sweep of conditions, without a random source.
			for i := range 500 {
				temp := 5.0 + 30.0*math.Abs(math.Sin(float64(i)/7.0))
				rh := 15.0 + 80.0*math.Abs(math.Cos(float64(i)/5.0))
				if rh > 99 {
					rh = 99
				}
				solar := 500.0 * math.Abs(math.Sin(float64(i)/3.0))
				rain := 0.0
				if i%13 == 0 {
					rain = 10.0
				}
				ok := s.Update(Weather{
					Elapsed:          time.Hour,
					Temperature:      firewx.Celsius(temp),
					RelativeHumidity: firewx.Percent(rh),
					SolarRadiation:   firewx.WattsPerSquareMeter(solar),
					Rainfall:         firewx.Millimeters(rain),
				})
				if !ok {
					t.Fatalf("step %d rejected", i)
				}
				mc := float64(s.MoistureContent())
				upper := (s.Wmx + s.Wfilm) * 100.0
				if mc < 0 || mc > upper+1e-9 {
					t.Fatalf("step %d moisture %v out of range [0, %v]", i, mc, upper)
				}
			}
		})
	}
}

// TestDrierAirLowersMoisture checks that a stick that is held in drier air
// settles to a lower moisture content than a stick that is held in wetter air.
// This is the direction property for the dead fuel moisture model.
func TestDrierAirLowersMoisture(t *testing.T) {
	run := func(rh float64) float64 {
		s := NewStandardStick(TenHour)
		for range 240 {
			s.Update(Weather{
				Elapsed:          time.Hour,
				Temperature:      20,
				RelativeHumidity: firewx.Percent(rh),
				SolarRadiation:   0,
			})
		}
		return float64(s.MoistureContent())
	}
	dry := run(20)
	wet := run(80)
	if !(dry < wet) {
		t.Errorf("drier air did not lower moisture: dry(20%%RH)=%.3f wet(80%%RH)=%.3f", dry, wet)
	}
}

// TestUpdateRejectsBadData checks that an out-of-range observation leaves the
// stick unchanged and returns false. The bounds match the checks in
// firelab/NFDRS4.
func TestUpdateRejectsBadData(t *testing.T) {
	cases := []struct {
		name string
		w    Weather
	}{
		{"zero elapsed", Weather{Elapsed: 0, Temperature: 20, RelativeHumidity: 50, SolarRadiation: 100}},
		{"humidity too high", Weather{Elapsed: time.Hour, Temperature: 20, RelativeHumidity: 150, SolarRadiation: 100}},
		{"humidity too low", Weather{Elapsed: time.Hour, Temperature: 20, RelativeHumidity: 0, SolarRadiation: 100}},
		{"temperature too high", Weather{Elapsed: time.Hour, Temperature: 70, RelativeHumidity: 50, SolarRadiation: 100}},
		{"temperature too low", Weather{Elapsed: time.Hour, Temperature: -70, RelativeHumidity: 50, SolarRadiation: 100}},
		{"solar too high", Weather{Elapsed: time.Hour, Temperature: 20, RelativeHumidity: 50, SolarRadiation: 2500}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewStandardStick(TenHour)
			before := float64(s.MoistureContent())
			if s.Update(tc.w) {
				t.Errorf("update accepted bad data")
			}
			after := float64(s.MoistureContent())
			if before != after {
				t.Errorf("bad data changed the moisture: before %v, after %v", before, after)
			}
		})
	}
}

// TestTenHourAgainstFEMS is the golden validation of the 10-hour stick against
// the official NFDRS4 output.
//
// Source: USDA Forest Service Fire Environment Mapping System (FEMS),
// climatology download for station GREENBASE (stationId 20284), hourly
// observations and hourly computed NFDR output for 2026-07-19 through
// 2026-08-02. FEMS computes the 10-hour time lag fuel moisture with the same
// Nelson (2000) dead fuel moisture model that this package implements.
//
// The test runs the 10-hour stick over the FEMS hourly weather, then compares
// the mean moisture content with the FEMS tenHR_TL_FuelMoisture column. It
// skips the first 48 hours, because this stick starts from a cold state and
// FEMS runs the stick continuously with a long history. After the spin-up, the
// two must agree closely.
func TestTenHourAgainstFEMS(t *testing.T) {
	f, err := os.Open("testdata/greenbase_20284.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	const spinupHours = 48
	s := NewStandardStick(TenHour)
	var prev time.Time
	var n int
	var sumSq, maxAbs float64

	for i, r := range rows[1:] {
		ts, err := time.Parse(time.RFC3339, r[0])
		if err != nil {
			t.Fatalf("row %d time: %v", i, err)
		}
		tempF := mustFloat(t, r[1])
		rh := mustFloat(t, r[2])
		precipIn := mustFloat(t, r[3])
		solar := mustFloat(t, r[4])
		fems := mustFloat(t, r[5])

		elapsed := time.Hour
		if !prev.IsZero() {
			elapsed = ts.Sub(prev)
		}
		prev = ts

		ok := s.Update(Weather{
			Elapsed:          elapsed,
			Temperature:      firewx.Fahrenheit(tempF).Celsius(),
			RelativeHumidity: firewx.Percent(rh),
			SolarRadiation:   firewx.WattsPerSquareMeter(solar),
			Rainfall:         firewx.Inches(precipIn).Millimeters(),
		})
		if !ok {
			t.Fatalf("row %d rejected", i)
		}
		if i < spinupHours {
			continue
		}
		d := float64(s.MoistureContent()) - fems
		sumSq += d * d
		if a := math.Abs(d); a > maxAbs {
			maxAbs = a
		}
		n++
	}

	if n == 0 {
		t.Fatal("no comparison points")
	}
	rmse := math.Sqrt(sumSq / float64(n))
	t.Logf("FEMS 10-hour comparison: n=%d RMSE=%.3f%% maxAbs=%.3f%%", n, rmse, maxAbs)

	// The port must track the reference implementation closely. The residual is
	// from the cold start, a possible barometric pressure difference, and
	// sub-hourly interpolation.
	if rmse > 1.0 {
		t.Errorf("RMSE %.3f%% exceeds 1.0%%", rmse)
	}
	if maxAbs > 4.0 {
		t.Errorf("max deviation %.3f%% exceeds 4.0%%", maxAbs)
	}
}

// mustFloat parses a float or fails the test.
func mustFloat(t *testing.T, s string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		t.Fatalf("parse float %q: %v", s, err)
	}
	return v
}
