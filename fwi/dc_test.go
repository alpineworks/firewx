package fwi

import (
	"testing"
	"time"
)

func TestDroughtCodeGolden(t *testing.T) {
	// Day 1 of the standard dataset, computed from the cffdrs equations.
	got := DroughtCode(15, 17, 0, time.April, 40)
	closeTo(t, float64(got), 19.01, 0.02, "DC(15,17C,0,Apr,40N)")
}

func TestDroughtCodeProperties(t *testing.T) {
	cases := []struct {
		name   string
		hi, lo DC
	}{
		{"warmer air deepens the drought", DroughtCode(15, 25, 0, time.April, 40), DroughtCode(15, 5, 0, time.April, 40)},
		{"heavy rain lowers the code", 200, DroughtCode(200, 15, 20, time.April, 40)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hi <= tc.lo {
				t.Errorf("%s: got hi=%v, lo=%v", tc.name, tc.hi, tc.lo)
			}
		})
	}
}

func TestDroughtCodeLightRainDoesNotWet(t *testing.T) {
	// Rain of 2.8 mm or less is at or below the threshold and adds no wetting,
	// so the code only rises by the day's evapotranspiration.
	dry := DroughtCode(100, 17, 0, time.April, 40)
	light := DroughtCode(100, 17, 2.8, time.April, 40)
	if dry != light {
		t.Errorf("rain of 2.8mm must not wet: dry=%v, light=%v", dry, light)
	}
}
