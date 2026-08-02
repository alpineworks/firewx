package rothermel

import (
	"encoding/csv"
	"os"
	"strconv"
	"testing"

	firewx "alpineworks.io/firewx"
)

// TestSpreadAgainstAndrews2018Table17 is the golden validation of the spread
// engine.
//
// Source: Andrews, P.L. 2018. RMRS-GTR-371, Table 17. The table divides a total
// oven-dry load of 2 tons/acre between the 1-hour and the 10-hour size classes,
// four ways. The fixed conditions are: 1-hour SAV 2500 ft2/ft3, 10-hour SAV 109
// ft2/ft3, fuel bed depth 1 ft, dead fuel moisture of extinction 20 percent,
// dead fuel moisture 5 percent, midflame wind speed 5 mi/h, and no slope. The
// table gives the bulk density, packing ratio, characteristic SAV, optimum
// packing ratio, relative packing ratio, wind factor, and rate of spread.
//
// The wind limit is off, which matches the table and the current BehavePlus
// default.
func TestSpreadAgainstAndrews2018Table17(t *testing.T) {
	f, err := os.Open("testdata/andrews2018_table17.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	for i, r := range rows[1:] {
		oneH := mustFloat(t, r[0])
		tenH := mustFloat(t, r[1])
		depth := mustFloat(t, r[2])
		mx := mustFloat(t, r[3])
		deadMoisture := mustFloat(t, r[4])
		wind := mustFloat(t, r[5])
		wantBulk := mustFloat(t, r[6])
		wantPacking := mustFloat(t, r[7])
		wantSAV := mustFloat(t, r[8])
		wantOptPacking := mustFloat(t, r[9])
		wantRelPacking := mustFloat(t, r[10])
		wantWindFactor := mustFloat(t, r[11])
		wantROS := mustFloat(t, r[12])

		fm := FuelModel{
			Name: "Andrews 2018 Table 17",
			Dead: []Particle{
				{Load: FuelLoadFromTonsPerAcre(oneH), SAV: 2500},
				{Load: FuelLoadFromTonsPerAcre(tenH), SAV: 109},
			},
			Depth:                    firewx.Feet(depth),
			DeadMoistureOfExtinction: firewx.Percent(mx),
		}
		cond := Conditions{
			Moisture:     Moisture{Dead: []firewx.Percent{firewx.Percent(deadMoisture), firewx.Percent(deadMoisture)}},
			MidflameWind: firewx.MilesPerHour(wind),
		}
		res := fm.Spread(cond)

		t.Run(rowName(oneH, tenH), func(t *testing.T) {
			closeTo(t, res.BulkDensity, wantBulk, 1e-3, "bulk density")
			closeTo(t, res.PackingRatio, wantPacking, 1e-4, "packing ratio")
			closeTo(t, res.CharacteristicSAV, wantSAV, 0.5, "characteristic SAV")
			closeTo(t, res.OptimumPackingRatio, wantOptPacking, 1e-4, "optimum packing ratio")
			closeTo(t, res.RelativePackingRatio, wantRelPacking, 1e-3, "relative packing ratio")
			closeTo(t, res.WindFactor, wantWindFactor, 0.1, "wind factor")
			closeTo(t, float64(res.RateOfSpread), wantROS, 0.1, "rate of spread")
		})
		_ = i
	}
}

// TestWindLimitReducesSpread checks that the wind limit reduces or holds the
// rate of spread, never increases it, for a coarse fuel with a strong wind.
func TestWindLimitReducesSpread(t *testing.T) {
	fm := FuelModel{
		Dead:                     []Particle{{Load: FuelLoadFromTonsPerAcre(2.0), SAV: 109}},
		Depth:                    1.0,
		DeadMoistureOfExtinction: 20,
	}
	base := Conditions{Moisture: Moisture{Dead: []firewx.Percent{5}}, MidflameWind: 20}
	limited := base
	limited.ApplyWindLimit = true

	unl := fm.Spread(base)
	lim := fm.Spread(limited)
	if lim.RateOfSpread > unl.RateOfSpread {
		t.Errorf("wind limit increased spread: limited %v, unlimited %v", lim.RateOfSpread, unl.RateOfSpread)
	}
	if !lim.WindLimited {
		t.Errorf("expected the wind limit to bind for a coarse fuel in a strong wind")
	}
}

// TestDrierFuelSpreadsFaster checks the direction property: a drier dead fuel
// spreads faster than a wetter one, other things equal.
func TestDrierFuelSpreadsFaster(t *testing.T) {
	fm := FuelModel{
		Dead:                     []Particle{{Load: FuelLoadFromTonsPerAcre(1.0), SAV: 2000}},
		Depth:                    1.0,
		DeadMoistureOfExtinction: 25,
	}
	ros := func(m firewx.Percent) float64 {
		res := fm.Spread(Conditions{Moisture: Moisture{Dead: []firewx.Percent{m}}, MidflameWind: 5})
		return float64(res.RateOfSpread)
	}
	dry := ros(4)
	wet := ros(20)
	if !(dry > wet) {
		t.Errorf("drier fuel did not spread faster: dry %v, wet %v", dry, wet)
	}
}

// TestSpreadStopsAtMoistureOfExtinction checks that the rate of spread falls to
// zero when the dead fuel moisture reaches the moisture of extinction.
func TestSpreadStopsAtMoistureOfExtinction(t *testing.T) {
	fm := FuelModel{
		Dead:                     []Particle{{Load: FuelLoadFromTonsPerAcre(1.0), SAV: 2000}},
		Depth:                    1.0,
		DeadMoistureOfExtinction: 20,
	}
	res := fm.Spread(Conditions{Moisture: Moisture{Dead: []firewx.Percent{20}}, MidflameWind: 5})
	if res.RateOfSpread != 0 {
		t.Errorf("expected zero spread at the moisture of extinction, got %v", res.RateOfSpread)
	}
}

// TestSlopeIncreasesSpread checks that a slope increases the rate of spread.
func TestSlopeIncreasesSpread(t *testing.T) {
	fm := FuelModel{
		Dead:                     []Particle{{Load: FuelLoadFromTonsPerAcre(1.0), SAV: 2000}},
		Depth:                    1.0,
		DeadMoistureOfExtinction: 25,
	}
	flat := fm.Spread(Conditions{Moisture: Moisture{Dead: []firewx.Percent{6}}})
	steep := fm.Spread(Conditions{Moisture: Moisture{Dead: []firewx.Percent{6}}, Slope: 30})
	if !(float64(steep.RateOfSpread) > float64(flat.RateOfSpread)) {
		t.Errorf("slope did not increase spread: flat %v, steep %v", flat.RateOfSpread, steep.RateOfSpread)
	}
	if steep.SlopeFactor <= 0 {
		t.Errorf("expected a positive slope factor, got %v", steep.SlopeFactor)
	}
}

// rowName makes a subtest name from the load split.
func rowName(oneH, tenH float64) string {
	return "1h=" + strconv.FormatFloat(oneH, 'g', -1, 64) + " 10h=" + strconv.FormatFloat(tenH, 'g', -1, 64)
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
