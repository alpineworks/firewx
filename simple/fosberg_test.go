package simple

import (
	"testing"

	firewx "alpineworks.io/firewx"
)

func TestFosbergValues(t *testing.T) {
	cases := []struct {
		name string
		t    firewx.Fahrenheit
		rh   firewx.Percent
		wind firewx.MilesPerHour
		want float64
		tol  float64
	}{
		// Recomputed independently against the Fosberg (1978) equations.
		{"golden 85F/20%/15mph (10-50 EMC band)", 85, 20, 15, 37.53, 0.05},
		{"high-RH branch 60F/70%/10mph", 60, 70, 10, 12.49, 0.05},
		// A saturated, very cold input drives EMC above 30; the clamp holds it at
		// 30 where the moisture term is zero, so the index is zero. An unclamped
		// equation would return a small negative value here instead.
		{"saturation clamps to zero", -60, 100, 10, 0, 1e-9},
		// Near-zero fuel moisture in a 30 mph wind approaches the extreme 100.
		{"calibration near 100", 110, 0, 30, 99.77, 0.1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closeTo(t, float64(FosbergIndex(tc.t, tc.rh, tc.wind)), tc.want, tc.tol, tc.name)
		})
	}
}

func TestFosbergMonotonic(t *testing.T) {
	cases := []struct {
		name   string
		hi, lo Fosberg
	}{
		{"rises as RH falls", FosbergIndex(85, 15, 10), FosbergIndex(85, 40, 10)},
		{"rises as wind rises", FosbergIndex(85, 20, 20), FosbergIndex(85, 20, 10)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hi <= tc.lo {
				t.Errorf("%s: got hi=%v, lo=%v", tc.name, tc.hi, tc.lo)
			}
		})
	}
}

func TestFosbergFromObs(t *testing.T) {
	// 29.4C=84.92F and 6.7056 m/s=15 mph; the present case must track the pure
	// function on the converted inputs.
	wantConv := float64(FosbergIndex(firewx.Celsius(29.4).Fahrenheit(), 20, firewx.MetersPerSecond(6.7056).MilesPerHour()))
	cases := []struct {
		name      string
		obs       firewx.Obs
		wantValid bool
		want, tol float64
	}{
		{
			name:      "absent without wind",
			obs:       firewx.Obs{Temperature: firewx.Some(firewx.Celsius(29.4)), RelativeHumidity: firewx.Some(firewx.Percent(20))},
			wantValid: false,
		},
		{
			name: "present converts SI inputs",
			obs: firewx.Obs{
				Temperature:      firewx.Some(firewx.Celsius(29.4)),
				RelativeHumidity: firewx.Some(firewx.Percent(20)),
				WindSpeed:        firewx.Some(firewx.MetersPerSecond(6.7056)),
			},
			wantValid: true,
			want:      wantConv,
			tol:       1e-9,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FosbergFromObs(tc.obs)
			if got.Valid() != tc.wantValid {
				t.Fatalf("%s: Valid()=%v, want %v", tc.name, got.Valid(), tc.wantValid)
			}
			if tc.wantValid {
				closeTo(t, float64(got.Must()), tc.want, tc.tol, tc.name)
			}
		})
	}
}
