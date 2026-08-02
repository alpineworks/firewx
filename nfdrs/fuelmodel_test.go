package nfdrs

import (
	"math"
	"testing"
)

// TestFuelModelYParameters pins the published parameters of fuel model Y
// (timber). The values are from the National Wildfire Coordinating Group fuel
// model parameter table.
func TestFuelModelYParameters(t *testing.T) {
	fm := FuelModelY
	closeTo(t, fm.Load1, 2.5, 1e-9, "1-hour load")
	closeTo(t, fm.Load10, 2.2, 1e-9, "10-hour load")
	closeTo(t, fm.Load100, 3.6, 1e-9, "100-hour load")
	closeTo(t, fm.Load1000, 10.16, 1e-9, "1000-hour load")
	closeTo(t, fm.LoadHerb, 0, 1e-9, "herbaceous load")
	closeTo(t, fm.LoadWoody, 0, 1e-9, "woody load")
	closeTo(t, fm.LoadDrought, 5, 1e-9, "drought load")
	closeTo(t, fm.SAV1, 2000, 1e-9, "1-hour SAV")
	closeTo(t, fm.SAV10, 109, 1e-9, "10-hour SAV")
	closeTo(t, fm.SAV100, 30, 1e-9, "100-hour SAV")
	closeTo(t, fm.SAV1000, 8, 1e-9, "1000-hour SAV")
	closeTo(t, fm.HeatContent, 8000, 1e-9, "heat content")
	closeTo(t, fm.ExtinctionMoisture, 25, 1e-9, "extinction moisture")
	closeTo(t, fm.Depth, 0.6, 1e-9, "depth")
	closeTo(t, fm.WindReductionFactor, 0.2, 1e-9, "wind reduction factor")
	closeTo(t, fm.MaxSpreadComponent, 5, 1e-9, "maximum spread component")
}

// TestAllModelsProduceFiniteIndices checks that every standard fuel model
// produces finite, non-negative indices for a normal set of conditions.
func TestAllModelsProduceFiniteIndices(t *testing.T) {
	models := []FuelModel{FuelModelV, FuelModelW, FuelModelX, FuelModelY, FuelModelZ}
	c := Conditions{
		Moisture1: 6, Moisture10: 7, Moisture100: 9, Moisture1000: 12,
		MoistureHerb: 90, MoistureWoody: 110,
		KBDIThreshold: 800,
		WindSpeed:     6, SlopeClass: 2, FuelTemperature: 25,
		GSI: 0.4, GSIMax: 1.0, GreenupThreshold: 0.5,
	}
	for _, fm := range models {
		t.Run(string(fm.Code), func(t *testing.T) {
			out := fm.Compute(c)
			for _, v := range []struct {
				name string
				val  float64
			}{
				{"SC", out.SpreadComponent},
				{"ERC", out.EnergyReleaseComponent},
				{"BI", out.BurningIndex},
				{"IC", out.IgnitionComponent},
			} {
				if math.IsNaN(v.val) || math.IsInf(v.val, 0) {
					t.Errorf("%s is not finite: %v", v.name, v.val)
				}
				if v.val < 0 {
					t.Errorf("%s is negative: %v", v.name, v.val)
				}
			}
		})
	}
}
