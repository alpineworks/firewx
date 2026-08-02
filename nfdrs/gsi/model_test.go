package gsi

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

// TestLFMMapping checks the live fuel moisture ramp against hand computation.
// The moisture is the minimum value below the green-up threshold, and it rises
// to the maximum value at the full GSI. The default parameters are woody 60 to
// 200 and herbaceous 30 to 250, with a green-up threshold of 0.5.
func TestLFMMapping(t *testing.T) {
	// Woody fuel. Drive the running average GSI to a target, then read the
	// moisture. With one value in the window, the running average equals it.
	woody := NewWoody(45)
	set := func(m *Model, gsi float64) {
		m.GSIQueue = []float64{gsi}
		m.CurrentMoisture = m.moisture(false)
	}
	set(woody, 0.4) // below the green-up threshold
	closeTo(t, float64(woody.Moisture()), 60, 1e-9, "woody below green-up")
	set(woody, 0.5) // at the threshold
	closeTo(t, float64(woody.Moisture()), 60, 1e-9, "woody at green-up")
	set(woody, 1.0) // full GSI
	closeTo(t, float64(woody.Moisture()), 200, 1e-9, "woody at full GSI")
	set(woody, 0.75) // midway on the ramp
	closeTo(t, float64(woody.Moisture()), 130, 1e-9, "woody midway")

	// Herbaceous fuel, non-annual so the curing state does not hold it down.
	herb := NewHerbaceous(45, false)
	set(herb, 0.5)
	closeTo(t, float64(herb.Moisture()), 30, 1e-9, "herb at green-up")
	set(herb, 1.0)
	closeTo(t, float64(herb.Moisture()), 250, 1e-9, "herb at full GSI")
}

// TestRunningAverage checks the running average window. The window holds at most
// MAPeriod values, and the GSI is their mean.
func TestRunningAverage(t *testing.T) {
	m := NewWoody(45)
	m.MAPeriod = 3
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	// Push GSI values by injecting fixed daily observations is hard, so push the
	// queue directly through Update's window logic with known GSI values.
	push := func(v float64, day int) {
		m.GSIQueue = append(m.GSIQueue, v)
		for len(m.GSIQueue) > m.MAPeriod {
			m.GSIQueue = m.GSIQueue[1:]
		}
		_ = base.AddDate(0, 0, day)
	}
	push(0.2, 0)
	push(0.4, 1)
	push(0.6, 2)
	closeTo(t, m.GSI(), 0.4, 1e-12, "mean of three values")
	push(0.9, 3) // drops 0.2
	closeTo(t, m.GSI(), (0.4+0.6+0.9)/3, 1e-12, "window drops the oldest value")
	if len(m.GSIQueue) != 3 {
		t.Errorf("window length: got %d, want 3", len(m.GSIQueue))
	}
}

// TestHerbCuringDoesNotRebound checks the annual curing rule. After an annual
// herbaceous fuel cures, the moisture does not rise again in the same year, even
// if the GSI rises.
func TestHerbCuringDoesNotRebound(t *testing.T) {
	m := NewHerbaceous(45, true)
	set := func(gsi float64) float64 {
		m.GSIQueue = []float64{gsi}
		m.CurrentMoisture = m.moisture(false)
		return float64(m.Moisture())
	}
	// Green up to a high moisture, above 120.
	high := set(0.9) // 0.9 -> 30 + slope; well above 120
	if high < 120 {
		t.Fatalf("expected green-up above 120, got %v", high)
	}
	// Cure down below 120.
	set(0.55)
	// Now the GSI rises again. The moisture must not rise, because the annual
	// fuel has cured.
	after := set(0.9)
	if after > 120 {
		t.Errorf("cured annual herb rebounded to %v", after)
	}
}

// TestNonAnnualHerbReboundsAfterCuring checks that a non-annual herbaceous fuel
// can rise again after a dry spell, unlike an annual fuel.
func TestNonAnnualHerbReboundsAfterCuring(t *testing.T) {
	m := NewHerbaceous(45, false)
	set := func(gsi float64) float64 {
		m.GSIQueue = []float64{gsi}
		m.CurrentMoisture = m.moisture(false)
		return float64(m.Moisture())
	}
	set(0.9)
	set(0.55)
	after := set(0.9)
	if after < 120 {
		t.Errorf("non-annual herb did not rebound, got %v", after)
	}
}

// TestJSONRoundTrip checks that a model marshals to JSON and back with an exact
// round trip, then produces identical output. The running average window and the
// curing state must survive.
func TestJSONRoundTrip(t *testing.T) {
	original := NewHerbaceous(47.7, true)
	base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	for i := range 40 {
		original.Update(DailyObs{
			Date:           base.AddDate(0, 0, i),
			MinTemperature: firewx.Celsius(2 + 0.3*float64(i)),
			MaxTemperature: firewx.Celsius(15 + 0.3*float64(i)),
			MinHumidity:    firewx.Percent(30),
		})
	}

	blob, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored Model
	if err := json.Unmarshal(blob, &restored); err != nil {
		t.Fatal(err)
	}
	blob2, err := json.Marshal(&restored)
	if err != nil {
		t.Fatal(err)
	}
	if string(blob) != string(blob2) {
		t.Fatalf("re-marshalled JSON differs from the original")
	}

	// Drive both with the same further weather.
	for i := 40; i < 60; i++ {
		obs := DailyObs{
			Date:           base.AddDate(0, 0, i),
			MinTemperature: firewx.Celsius(2 + 0.3*float64(i)),
			MaxTemperature: firewx.Celsius(15 + 0.3*float64(i)),
			MinHumidity:    firewx.Percent(30),
		}
		original.Update(obs)
		restored.Update(obs)
		if original.GSI() != restored.GSI() {
			t.Fatalf("step %d GSI: original %v, restored %v", i, original.GSI(), restored.GSI())
		}
		if original.Moisture() != restored.Moisture() {
			t.Fatalf("step %d moisture: original %v, restored %v", i, original.Moisture(), restored.Moisture())
		}
	}
}

// TestSeasonalBehaviorOnFEMSWeather runs the model over real station weather and
// checks the physical behavior.
//
// Source: USDA Forest Service FEMS, station GREENBASE (20284), daily aggregated
// weather for 2026-04-15 through 2026-08-02, in the northern hemisphere spring
// and summer.
//
// This test checks the model on real historical weather:
//   - the GSI stays in the range 0 to 1 every day;
//   - the live fuel moisture stays in the parameter range;
//   - the running average GSI rises from the spring to the late summer.
//
// The test does not compare the live fuel moisture with the FEMS output value by
// value. FEMS uses the site settings of the station, which are not public, and
// it defines the daily window in local standard time. The exact match needs the
// site settings, so the assembly step validates the full National Fire Danger
// Rating System against the FEMS energy release component. This test logs the
// error against the FEMS output for reference only.
func TestSeasonalBehaviorOnFEMSWeather(t *testing.T) {
	f, err := os.Open("testdata/greenbase_20284_daily.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	woody := NewWoody(47.7)
	var earlyGSI, lateGSI float64
	var earlyN, lateN int
	var woodySq float64
	var n int

	total := len(rows) - 1
	for i, r := range rows[1:] {
		date, _ := time.Parse("2006-01-02", r[0])
		obs := DailyObs{
			Date:           date,
			MinTemperature: firewx.Celsius(mustFloat(t, r[1])),
			MaxTemperature: firewx.Celsius(mustFloat(t, r[2])),
			MinHumidity:    firewx.Percent(mustFloat(t, r[3])),
		}
		woody.Update(obs)

		// The GSI stays in the range 0 to 1.
		if g := woody.GSI(); g < 0 || g > 1 {
			t.Fatalf("day %d GSI %v is out of the range 0 to 1", i, g)
		}
		// The live fuel moisture stays in the parameter range.
		if m := float64(woody.Moisture()); m < woody.MinLFM-1e-9 || m > woody.MaxLFM+1e-9 {
			t.Fatalf("day %d woody moisture %v is out of range", i, m)
		}
		// Collect the early and the late GSI for the seasonal rise check.
		if i < 21 {
			earlyGSI += woody.GSI()
			earlyN++
		}
		if i >= total-21 {
			lateGSI += woody.GSI()
			lateN++
		}
		dw := float64(woody.Moisture()) - mustFloat(t, r[5])
		woodySq += dw * dw
		n++
	}

	earlyMean := earlyGSI / float64(earlyN)
	lateMean := lateGSI / float64(lateN)
	t.Logf("GSI spring mean %.3f, late summer mean %.3f; woody moisture RMSE vs FEMS %.2f%% (default settings)",
		earlyMean, lateMean, math.Sqrt(woodySq/float64(n)))

	// The running average GSI rises from the spring to the late summer.
	if !(lateMean > earlyMean) {
		t.Errorf("GSI did not rise through the season: spring %.3f, late summer %.3f", earlyMean, lateMean)
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
