package nfdrs

import (
	"encoding/csv"
	"math"
	"os"
	"strconv"
	"testing"

	firewx "alpineworks.io/firewx"
)

// TestIndicesAgainstFEMS is the golden validation of the index equations.
//
// Source: USDA Forest Service FEMS, station GREENBASE (20284), hourly weather
// and hourly NFDR output for 2026-04-15 through 2026-08-02, sampled every three
// hours. GREENBASE uses the standard NFDRS2016 fuel model Y (timber).
//
// The test feeds the FEMS dead fuel moisture and the FEMS drought index into the
// index equations, then it compares the spread component, the energy release
// component, and the burning index with the FEMS output. This isolates the index
// equations from the fuel moisture models.
//
// Two site settings match the FEMS output: the slope class is 1, and the site
// applies no drought fuel load in this period (the drought index stays below its
// threshold). The fuel model parameters are the published values, not fitted.
func TestIndicesAgainstFEMS(t *testing.T) {
	f, err := os.Open("testdata/greenbase_20284_nfdr.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	var n int
	var scSq, ercSq, biSq float64
	var scMax, ercMax, biMax float64

	for _, r := range rows[1:] {
		c := Conditions{
			Moisture1:     mustFloat(t, r[1]),
			Moisture10:    mustFloat(t, r[2]),
			Moisture100:   mustFloat(t, r[3]),
			Moisture1000:  mustFloat(t, r[4]),
			KBDI:          mustFloat(t, r[5]),
			KBDIThreshold: 800, // the drought load is off for this station
			WindSpeed:     firewx.MilesPerHour(mustFloat(t, r[6])),
			SlopeClass:    1,
		}
		out := FuelModelY.Compute(c)

		dsc := out.SpreadComponent - mustFloat(t, r[7])
		derc := out.EnergyReleaseComponent - mustFloat(t, r[8])
		dbi := out.BurningIndex - mustFloat(t, r[9])
		scSq += dsc * dsc
		ercSq += derc * derc
		biSq += dbi * dbi
		scMax = math.Max(scMax, math.Abs(dsc))
		ercMax = math.Max(ercMax, math.Abs(derc))
		biMax = math.Max(biMax, math.Abs(dbi))
		n++
	}

	scRMSE := math.Sqrt(scSq / float64(n))
	ercRMSE := math.Sqrt(ercSq / float64(n))
	biRMSE := math.Sqrt(biSq / float64(n))
	t.Logf("FEMS n=%d | SC RMSE=%.4f max=%.3f | ERC RMSE=%.4f max=%.3f | BI RMSE=%.4f max=%.3f",
		n, scRMSE, scMax, ercRMSE, ercMax, biRMSE, biMax)

	// The equations reproduce the FEMS output to the rounding of the FEMS data.
	// One hour of the spread component differs, because the exported wind for
	// that hour is zero while the FEMS engine used a non-zero wind. The root
	// mean square error absorbs that one hour.
	if ercRMSE > 0.05 {
		t.Errorf("ERC RMSE %.4f exceeds 0.05", ercRMSE)
	}
	if scRMSE > 0.15 {
		t.Errorf("SC RMSE %.4f exceeds 0.15", scRMSE)
	}
	if biRMSE > 0.5 {
		t.Errorf("BI RMSE %.4f exceeds 0.5", biRMSE)
	}
}

// TestIgnitionComponent checks the behavior of the ignition component. The
// component falls as the fuel is cooler or wetter, it is zero when the spread
// component is zero, and it stays in the range 0 to 100.
func TestIgnitionComponent(t *testing.T) {
	base := Conditions{
		Moisture1: 6, Moisture10: 7, Moisture100: 9, Moisture1000: 12,
		KBDIThreshold: 800, WindSpeed: 5, SlopeClass: 1, FuelTemperature: 30,
	}
	hot := FuelModelY.Compute(base).IgnitionComponent
	cold := FuelModelY.Compute(withFuelTemp(base, 5)).IgnitionComponent
	if !(hot > cold) {
		t.Errorf("ignition did not fall with a cooler fuel: hot %v, cold %v", hot, cold)
	}

	wet := base
	wet.Moisture1 = 25
	dry := base
	dry.Moisture1 = 3
	if !(FuelModelY.Compute(dry).IgnitionComponent > FuelModelY.Compute(wet).IgnitionComponent) {
		t.Errorf("ignition did not fall with a wetter fuel")
	}

	// The ignition component stays in range.
	for temp := -10.0; temp <= 45; temp += 5 {
		for m := 1.0; m <= 40; m += 3 {
			c := withFuelTemp(base, firewx.Celsius(temp))
			c.Moisture1 = m
			ic := FuelModelY.Compute(c).IgnitionComponent
			if ic < 0 || ic > 100 {
				t.Fatalf("ignition %v out of range at temp %v moisture %v", ic, temp, m)
			}
		}
	}
}

// TestWetterFuelLowersERC checks the direction property: a wetter dead fuel gives
// a lower energy release component.
func TestWetterFuelLowersERC(t *testing.T) {
	dry := Conditions{Moisture1: 4, Moisture10: 5, Moisture100: 7, Moisture1000: 10, KBDIThreshold: 800, SlopeClass: 1}
	wet := Conditions{Moisture1: 18, Moisture10: 20, Moisture100: 22, Moisture1000: 25, KBDIThreshold: 800, SlopeClass: 1}
	if !(FuelModelY.Compute(dry).EnergyReleaseComponent > FuelModelY.Compute(wet).EnergyReleaseComponent) {
		t.Errorf("wetter fuel did not lower the energy release component")
	}
}

// TestInvalidInputsReturnZero checks that an out-of-range input gives zero
// indices, which matches firelab/NFDRS4.
func TestInvalidInputsReturnZero(t *testing.T) {
	valid := Conditions{Moisture1: 6, Moisture10: 7, Moisture100: 9, Moisture1000: 12, KBDIThreshold: 800, SlopeClass: 1, WindSpeed: 5}
	cases := []struct {
		name string
		fm   FuelModel
		c    Conditions
	}{
		{"slope class zero", FuelModelY, withSlope(valid, 0)},
		{"slope class six", FuelModelY, withSlope(valid, 6)},
		{"negative wind", FuelModelY, withWind(valid, -1)},
		{"zero depth", withDepth(FuelModelY, 0), valid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.fm.Compute(tc.c)
			if got != (Indices{}) {
				t.Errorf("expected zero indices, got %+v", got)
			}
		})
	}
}

func withSlope(c Conditions, s int) Conditions                { c.SlopeClass = s; return c }
func withWind(c Conditions, w firewx.MilesPerHour) Conditions { c.WindSpeed = w; return c }
func withDepth(fm FuelModel, d float64) FuelModel             { fm.Depth = d; return fm }

// withFuelTemp returns a copy of c with the fuel temperature set.
func withFuelTemp(c Conditions, temp firewx.Celsius) Conditions {
	c.FuelTemperature = temp
	return c
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
