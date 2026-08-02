package fwi

import "testing"

func TestFineFuelMoistureCodeGolden(t *testing.T) {
	// Day 1 of the standard dataset, computed from the cffdrs equations.
	got := FineFuelMoistureCode(85, 17, 42, 25, 0)
	closeTo(t, float64(got), 87.65, 0.02, "FFMC(85,17C,42%,25,0)")
}

func TestFineFuelMoistureCodeProperties(t *testing.T) {
	cases := []struct {
		name   string
		hi, lo FFMC
	}{
		{"drier air raises the code", FineFuelMoistureCode(85, 17, 20, 25, 0), FineFuelMoistureCode(85, 17, 80, 25, 0)},
		{"more wind raises the code when drying", FineFuelMoistureCode(85, 17, 20, 40, 0), FineFuelMoistureCode(85, 17, 20, 5, 0)},
		{"rain lowers the code", 90, FineFuelMoistureCode(90, 15, 60, 10, 8)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hi <= tc.lo {
				t.Errorf("%s: got hi=%v, lo=%v", tc.name, tc.hi, tc.lo)
			}
		})
	}
}

func TestFineFuelMoistureCodeStaysInRange(t *testing.T) {
	cases := []struct {
		name string
		f    FFMC
	}{
		{"hot dry windy", FineFuelMoistureCode(90, 40, 5, 50, 0)},
		{"cold saturated calm", FineFuelMoistureCode(10, -20, 100, 0, 0)},
		{"heavy rain", FineFuelMoistureCode(95, 20, 50, 20, 60)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.f < 0 || tc.f > 101 {
				t.Errorf("%s: FFMC out of range: %v", tc.name, tc.f)
			}
		})
	}
}
