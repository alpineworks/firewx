package nfdrs

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

// TestDriverAgainstFEMS is the end-to-end validation of the assembly driver.
//
// Source: USDA Forest Service FEMS, station GREENBASE (20284), hourly weather
// and hourly NFDR output for 2026-04-15 through 2026-08-02. GREENBASE uses the
// standard NFDRS2016 fuel model Y (timber), with slope class 1 and no drought
// fuel load.
//
// The driver takes only the raw hourly weather. It runs the four Nelson sticks,
// then it computes the indices. The test compares the dead fuel moisture and the
// three weather-driven indices with the FEMS output.
//
// The test skips a spin-up of 1100 hours, about 46 days. The 1000-hour stick has
// a long time lag, so it needs about six weeks to settle from the cold start.
// After the spin-up, the four sticks match the FEMS output closely, and the
// indices follow.
func TestDriverAgainstFEMS(t *testing.T) {
	rows := readHourly(t, "testdata/greenbase_20284_hourly.csv")

	d := NewDriver(Config{
		FuelModel:        FuelModelY,
		Latitude:         47.7,
		SlopeClass:       1,
		KBDIThreshold:    800, // the drought load is off for this station
		MeanAnnualPrecip: 40,
		AnnualHerb:       true,
		RegObsHour:       13,
		LSTOffset:        -8 * time.Hour, // Pacific standard time
	})

	const spinup = 1100
	var n int
	var scSq, ercSq, biSq float64
	var scMax, ercMax float64
	var mc10Sq, mc100Sq, mc1000Sq float64

	for i, r := range rows {
		if !d.Update(r.obs) {
			t.Fatalf("row %d rejected", i)
		}
		if i < spinup {
			continue
		}
		out := d.Indices()
		dsc := out.SpreadComponent - r.femsSC
		derc := out.EnergyReleaseComponent - r.femsERC
		dbi := out.BurningIndex - r.femsBI
		scSq += dsc * dsc
		ercSq += derc * derc
		biSq += dbi * dbi
		scMax = math.Max(scMax, math.Abs(dsc))
		ercMax = math.Max(ercMax, math.Abs(derc))

		_, mc10, mc100, mc1000 := d.DeadMoistures()
		mc10Sq += sq(float64(mc10) - r.mc10)
		mc100Sq += sq(float64(mc100) - r.mc100)
		mc1000Sq += sq(float64(mc1000) - r.mc1000)
		n++
	}

	rmse := func(sum float64) float64 { return math.Sqrt(sum / float64(n)) }
	t.Logf("driver n=%d | SC RMSE=%.3f max=%.2f | ERC RMSE=%.3f max=%.2f | BI RMSE=%.3f", n, rmse(scSq), scMax, rmse(ercSq), ercMax, rmse(biSq))
	t.Logf("dead moisture RMSE | 10h=%.3f 100h=%.3f 1000h=%.3f", rmse(mc10Sq), rmse(mc100Sq), rmse(mc1000Sq))

	// The four sticks match the FEMS output after the spin-up.
	if rmse(mc10Sq) > 0.15 {
		t.Errorf("10-hour moisture RMSE %.3f exceeds 0.15", rmse(mc10Sq))
	}
	if rmse(mc100Sq) > 0.15 {
		t.Errorf("100-hour moisture RMSE %.3f exceeds 0.15", rmse(mc100Sq))
	}
	if rmse(mc1000Sq) > 0.15 {
		t.Errorf("1000-hour moisture RMSE %.3f exceeds 0.15", rmse(mc1000Sq))
	}
	// The indices follow the FEMS output. The bounds allow the small residual
	// of the dead fuel moisture and one hour whose exported wind is zero.
	if rmse(scSq) > 0.2 {
		t.Errorf("SC RMSE %.3f exceeds 0.2", rmse(scSq))
	}
	if rmse(ercSq) > 0.4 {
		t.Errorf("ERC RMSE %.3f exceeds 0.4", rmse(ercSq))
	}
	if rmse(biSq) > 1.0 {
		t.Errorf("BI RMSE %.3f exceeds 1.0", rmse(biSq))
	}
}

// TestDriverRejectsIncompleteObs checks that the driver leaves its state
// unchanged and returns false for an observation that lacks a required input.
func TestDriverRejectsIncompleteObs(t *testing.T) {
	d := NewDriver(Config{FuelModel: FuelModelY, Latitude: 47.7, SlopeClass: 1, KBDIThreshold: 800, RegObsHour: 13})
	o := firewx.Obs{
		Time:        time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		Temperature: firewx.Some(firewx.Celsius(20)),
		// The humidity, the solar radiation, and the precipitation are absent.
	}
	if d.Update(o) {
		t.Errorf("the driver accepted an incomplete observation")
	}
}

// TestDriverStateRoundTrip checks that a driver marshals to JSON and back with
// an exact round trip. A restored driver produces identical output for the same
// further weather, so a caller can persist the driver and resume without the
// spin-up.
func TestDriverStateRoundTrip(t *testing.T) {
	cfg := Config{
		FuelModel: FuelModelY, Latitude: 47.7, SlopeClass: 1, KBDIThreshold: 800,
		MeanAnnualPrecip: 40, AnnualHerb: true, RegObsHour: 13, LSTOffset: -8 * time.Hour,
	}
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	mkObs := func(i int) firewx.Obs {
		fi := float64(i)
		rain := 0.0
		if i%17 == 0 {
			rain = 3.0
		}
		return firewx.Obs{
			Time:             base.Add(time.Duration(i) * time.Hour),
			Temperature:      firewx.Some(firewx.Celsius(15 + 10*math.Sin(fi/6))),
			RelativeHumidity: firewx.Some(firewx.Percent(40 + 30*math.Cos(fi/5))),
			SolarRadiation:   firewx.Some(firewx.WattsPerSquareMeter(400 * math.Abs(math.Sin(fi/12)))),
			Precipitation:    firewx.Some(firewx.Millimeters(rain)),
			WindSpeed:        firewx.Some(firewx.MetersPerSecond(3)),
		}
	}

	// Run the driver across two daily boundaries so the daily state is set.
	orig := NewDriver(cfg)
	for i := range 60 {
		orig.Update(mkObs(i))
	}

	blob, err := json.Marshal(orig.State())
	if err != nil {
		t.Fatal(err)
	}
	var st DriverState
	if err := json.Unmarshal(blob, &st); err != nil {
		t.Fatal(err)
	}
	restored := NewDriver(cfg)
	if err := restored.SetState(st); err != nil {
		t.Fatal(err)
	}

	// A second marshal after the restore is byte-identical.
	blob2, err := json.Marshal(restored.State())
	if err != nil {
		t.Fatal(err)
	}
	if string(blob) != string(blob2) {
		t.Fatalf("re-marshalled driver state differs from the original")
	}

	// Drive both with the same further weather. The output must stay identical.
	for i := 60; i < 96; i++ {
		o := mkObs(i)
		orig.Update(o)
		restored.Update(o)
		if orig.Indices() != restored.Indices() {
			t.Fatalf("step %d indices: original %+v, restored %+v", i, orig.Indices(), restored.Indices())
		}
		a1, a10, a100, a1000 := orig.DeadMoistures()
		b1, b10, b100, b1000 := restored.DeadMoistures()
		if a1 != b1 || a10 != b10 || a100 != b100 || a1000 != b1000 {
			t.Fatalf("step %d dead moistures differ", i)
		}
	}
}

// TestDriverSetStateErrors checks that SetState rejects a wrong schema version
// and an absent sub-model.
func TestDriverSetStateErrors(t *testing.T) {
	cfg := Config{FuelModel: FuelModelY, SlopeClass: 1, KBDIThreshold: 800, RegObsHour: 13}
	d := NewDriver(cfg)
	good := NewDriver(cfg).State()

	wrongVersion := good
	wrongVersion.SchemaVersion = good.SchemaVersion + 1
	if d.SetState(wrongVersion) == nil {
		t.Errorf("expected an error for a wrong schema version")
	}

	absentModel := good
	absentModel.Stick10 = nil
	if d.SetState(absentModel) == nil {
		t.Errorf("expected an error for an absent sub-model")
	}

	if err := d.SetState(good); err != nil {
		t.Errorf("unexpected error for a valid state: %v", err)
	}
}

type hourly struct {
	obs                     firewx.Obs
	mc10, mc100, mc1000     float64
	femsSC, femsERC, femsBI float64
}

// readHourly reads the driver fixture into observations and the FEMS reference.
func readHourly(t *testing.T, path string) []hourly {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	out := make([]hourly, 0, len(records)-1)
	for _, r := range records[1:] {
		ts, err := time.Parse(time.RFC3339, r[0])
		if err != nil {
			t.Fatalf("time %q: %v", r[0], err)
		}
		// Skip a weather gap. FEMS fills the gap in its own output, but the
		// driver takes only the raw weather, so it processes a longer elapsed
		// time across the gap.
		tf, ok1 := optFloat(r[1])
		rh, ok2 := optFloat(r[2])
		pin, ok3 := optFloat(r[3])
		wind, ok4 := optFloat(r[4])
		solar, ok5 := optFloat(r[5])
		if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
			continue
		}
		snow, _ := optFloat(r[6])
		o := firewx.Obs{
			Time:             ts,
			Temperature:      firewx.Some(firewx.Fahrenheit(tf).Celsius()),
			RelativeHumidity: firewx.Some(firewx.Percent(rh)),
			Precipitation:    firewx.Some(firewx.Inches(pin).Millimeters()),
			WindSpeed:        firewx.Some(firewx.MilesPerHour(wind).MetersPerSecond()),
			SolarRadiation:   firewx.Some(firewx.WattsPerSquareMeter(solar)),
			SnowCovered:      firewx.Some(snow != 0),
		}
		out = append(out, hourly{
			obs:     o,
			mc10:    mustFloat(t, r[7]),
			mc100:   mustFloat(t, r[8]),
			mc1000:  mustFloat(t, r[9]),
			femsSC:  mustFloat(t, r[10]),
			femsERC: mustFloat(t, r[11]),
			femsBI:  mustFloat(t, r[12]),
		})
	}
	return out
}

// optFloat parses a float, reporting whether the cell held a value.
func optFloat(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

func sq(x float64) float64 { return x * x }
