package fwi

import (
	"testing"
	"time"
)

func TestDuffMoistureCodeGolden(t *testing.T) {
	// Day 1 of the standard dataset, computed from the cffdrs equations.
	got := DuffMoistureCode(6, 17, 42, 0, time.April, 40)
	closeTo(t, float64(got), 8.55, 0.02, "DMC(6,17C,42%,0,Apr,40N)")
}

func TestDuffMoistureCodeProperties(t *testing.T) {
	cases := []struct {
		name   string
		hi, lo DMC
	}{
		{"warmer air dries the duff faster", DuffMoistureCode(6, 25, 42, 0, time.April, 40), DuffMoistureCode(6, 5, 42, 0, time.April, 40)},
		{"rain lowers the code", 40, DuffMoistureCode(40, 15, 60, 10, time.April, 40)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hi <= tc.lo {
				t.Errorf("%s: got hi=%v, lo=%v", tc.name, tc.hi, tc.lo)
			}
		})
	}
}

func TestDuffMoistureCodeNonNegative(t *testing.T) {
	// A cold day below the -1.1C floor adds no drying and stays non-negative.
	if got := DuffMoistureCode(0, -10, 90, 0, time.January, 40); got < 0 {
		t.Errorf("DMC should not go negative, got %v", got)
	}
}
