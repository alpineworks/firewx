package fwi

import "testing"

func TestFireWeatherIndexGolden(t *testing.T) {
	// Day 1 of the standard dataset: ISI 10.775, BUI 8.490.
	got := FireWeatherIndex(10.775, 8.490)
	closeTo(t, float64(got), 10.04, 0.05, "FWI(10.775,8.490)")
}

func TestFireWeatherIndexBranches(t *testing.T) {
	cases := []struct {
		name string
		i    ISI
		b    BUI
	}{
		// A high BUI (over 80) uses the second duff function.
		{"high BUI branch", 15, 120},
		// A low product keeps the index equal to the intermediate value.
		{"low intensity branch", 0.5, 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FireWeatherIndex(tc.i, tc.b); got < 0 || isNaNOrInf(float64(got)) {
				t.Errorf("%s: got %v", tc.name, got)
			}
		})
	}
}

func TestFireWeatherIndexMonotonic(t *testing.T) {
	if FireWeatherIndex(20, 40) <= FireWeatherIndex(5, 40) {
		t.Error("FWI should rise with the spread index")
	}
}
