package nfdrs

import (
	"encoding/csv"
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
