package nelson

import (
	"math"
	"testing"
)

func closeTo(t *testing.T, got, want, tol float64, what string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s: got %v, want %v (tol %v)", what, got, want, tol)
	}
}

func TestDeriveStickNodes(t *testing.T) {
	// Every standard stick derives to 11 nodes, which matches the firelab/NFDRS4
	// default. The count must be odd, so the stick has a centre node.
	cases := []struct {
		name   string
		radius float64
		want   int
	}{
		{"1-hour", 0.20, 11},
		{"10-hour", 0.64, 11},
		{"100-hour", 2.00, 11},
		{"1000-hour", 3.81, 11},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveStickNodes(tc.radius)
			if got != tc.want {
				t.Errorf("deriveStickNodes(%v)=%d, want %d", tc.radius, got, tc.want)
			}
			if got%2 == 0 {
				t.Errorf("node count must be odd, got %d", got)
			}
		})
	}
}

func TestDerivedParameters10Hour(t *testing.T) {
	// The 10-hour stick has a radius of 0.64 cm. The expected values are
	// computed from the Bevins (2005) equations in firelab/NFDRS4.
	const r = 0.64
	closeTo(t, deriveAdsorptionRate(r), 0.079545, 1e-5, "adsorption rate")
	closeTo(t, derivePlanarHeatTransferRate(r), 0.380022, 1e-5, "planar heat transfer rate")
	closeTo(t, deriveRainfallRunoffFactor(r), 0.310094, 1e-5, "rainfall runoff factor")
	if got := deriveDiffusivitySteps(r); got != 9 {
		t.Errorf("diffusivity steps: got %d, want 9", got)
	}
	if got := deriveMoistureSteps(r); got != 60 {
		t.Errorf("moisture steps: got %d, want 60", got)
	}
}

func TestStandardSticks(t *testing.T) {
	// The fixed parameters of the four standard sticks, from the firelab/NFDRS4
	// stick initializers.
	cases := []struct {
		name                            string
		tl                              TimeLag
		radius, adsorption, maxMoisture float64
	}{
		{"1-hour", OneHour, 0.20, 0.462252733, 0.35},
		{"10-hour", TenHour, 0.64, 0.079548303, 0.35},
		{"100-hour", HundredHour, 2.00, 0.06, 0.35},
		{"1000-hour", ThousandHour, 3.81, 0.06, 0.35},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := standardSticks[tc.tl]
			closeTo(t, s.radius, tc.radius, 1e-9, "radius")
			closeTo(t, s.adsorptionRate, tc.adsorption, 1e-9, "adsorption rate")
			closeTo(t, s.maxLocalMoisture, tc.maxMoisture, 1e-9, "max local moisture")
		})
	}
}

func TestTenHourAdsorptionMatchesBevins(t *testing.T) {
	// The 10-hour stick is the one whose hardcoded adsorption rate agrees with
	// the Bevins equation. The 1-hour rate is calibrated, and the 100-hour and
	// 1000-hour rates are held at the 0.06 floor.
	s := standardSticks[TenHour]
	closeTo(t, s.adsorptionRate, deriveAdsorptionRate(s.radius), 1e-4, "10-hour adsorption")
}
