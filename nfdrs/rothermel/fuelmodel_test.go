package rothermel

import "testing"

// TestParticleDefaults checks that a zero field returns the standard value, and
// that a set field returns the set value.
func TestParticleDefaults(t *testing.T) {
	// A particle with only the load and the SAV uses the standard values.
	def := Particle{Load: 0.1, SAV: 2000}
	closeTo(t, def.heatContent(), 8000, 1e-9, "default heat content")
	closeTo(t, def.density(), 32, 1e-9, "default density")
	closeTo(t, def.totalMineral(), 0.0555, 1e-9, "default total mineral")
	closeTo(t, def.effectiveMineral(), 0.010, 1e-9, "default effective mineral")

	// A particle with set values returns them.
	set := Particle{
		Load:             0.1,
		SAV:              2000,
		HeatContent:      9000,
		Density:          30,
		TotalMineral:     0.06,
		EffectiveMineral: 0.012,
	}
	closeTo(t, set.heatContent(), 9000, 1e-9, "set heat content")
	closeTo(t, set.density(), 30, 1e-9, "set density")
	closeTo(t, set.totalMineral(), 0.06, 1e-9, "set total mineral")
	closeTo(t, set.effectiveMineral(), 0.012, 1e-9, "set effective mineral")
}

// TestSpreadPanicsOnMoistureMismatch checks that Spread panics when the moisture
// count does not match the particle count.
func TestSpreadPanicsOnMoistureMismatch(t *testing.T) {
	fm := FuelModel{
		Dead:                     []Particle{{Load: 0.1, SAV: 2000}},
		Depth:                    1.0,
		DeadMoistureOfExtinction: 20,
	}
	defer func() {
		if recover() == nil {
			t.Errorf("expected a panic on a moisture count mismatch")
		}
	}()
	// No dead moisture given for one dead particle.
	fm.Spread(Conditions{Moisture: Moisture{}, MidflameWind: 5})
}
