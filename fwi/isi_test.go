package fwi

import "testing"

func TestInitialSpreadIndexGolden(t *testing.T) {
	// Day 1 of the standard dataset: FFMC 87.65, wind 25 km/h.
	got := InitialSpreadIndex(87.65, 25)
	closeTo(t, float64(got), 10.78, 0.05, "ISI(87.65,25)")
}

func TestInitialSpreadIndexMonotonic(t *testing.T) {
	cases := []struct {
		name   string
		hi, lo ISI
	}{
		{"rises with wind", InitialSpreadIndex(87, 40), InitialSpreadIndex(87, 10)},
		{"rises with a drier FFMC", InitialSpreadIndex(95, 20), InitialSpreadIndex(80, 20)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hi <= tc.lo {
				t.Errorf("%s: got hi=%v, lo=%v", tc.name, tc.hi, tc.lo)
			}
		})
	}
}
