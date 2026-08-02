package fwi

import "testing"

func TestBuildupIndexGolden(t *testing.T) {
	// Day 1 of the standard dataset: DMC 8.545, DC 19.014.
	got := BuildupIndex(8.545, 19.014)
	closeTo(t, float64(got), 8.49, 0.02, "BUI(8.545,19.014)")
}

func TestBuildupIndexEdges(t *testing.T) {
	cases := []struct {
		name      string
		d         DMC
		c         DC
		want, tol float64
	}{
		{"both zero gives zero", 0, 0, 0, 1e-12},
		{"zero duff gives zero", 0, 100, 0, 1e-12},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closeTo(t, float64(BuildupIndex(tc.d, tc.c)), tc.want, tc.tol, tc.name)
		})
	}
}

func TestBuildupIndexRisesWithFuel(t *testing.T) {
	if BuildupIndex(50, 200) <= BuildupIndex(10, 40) {
		t.Error("BUI should rise with the moisture codes")
	}
}
