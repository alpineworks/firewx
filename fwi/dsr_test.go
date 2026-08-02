package fwi

import "testing"

func TestDailySeverityRatingGolden(t *testing.T) {
	// Day 1 of the standard dataset: FWI 10.04.
	got := DailySeverityRating(10.04)
	closeTo(t, float64(got), 1.61, 0.05, "DSR(10.04)")
}

func TestDailySeverityRatingRisesWithFWI(t *testing.T) {
	cases := []struct {
		name   string
		hi, lo DSR
	}{
		{"rises with the fire weather index", DailySeverityRating(20), DailySeverityRating(10)},
		{"zero at zero", DailySeverityRating(0), 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "zero at zero" {
				closeTo(t, float64(tc.hi), 0, 1e-12, tc.name)
				return
			}
			if tc.hi <= tc.lo {
				t.Errorf("%s: got hi=%v, lo=%v", tc.name, tc.hi, tc.lo)
			}
		})
	}
}
