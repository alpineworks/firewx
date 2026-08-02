package simple

import (
	"testing"

	firewx "alpineworks.io/firewx"
)

func TestHDWValues(t *testing.T) {
	cases := []struct {
		name string
		vpd  firewx.Hectopascals
		wind firewx.MetersPerSecond
		want float64
		tol  float64
	}{
		// T=25C, RH=40%, wind=5 m/s gives VPD ~19.01 hPa, product ~95.04.
		{"golden 19.01hPa x 5m/s", 19.00893, 5, 95.04, 0.05},
		{"zero VPD gives zero", 0, 5, 0, 1e-12},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closeTo(t, float64(HDWIndex(tc.vpd, tc.wind)), tc.want, tc.tol, tc.name)
		})
	}
}

func TestHDWMonotonic(t *testing.T) {
	cases := []struct {
		name   string
		hi, lo HDW
	}{
		{"rises with wind", HDWIndex(19, 10), HDWIndex(19, 5)},
		{"rises with VPD", HDWIndex(30, 5), HDWIndex(19, 5)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hi <= tc.lo {
				t.Errorf("%s: got hi=%v, lo=%v", tc.name, tc.hi, tc.lo)
			}
		})
	}
}

func TestHDWFromObs(t *testing.T) {
	cases := []struct {
		name      string
		obs       firewx.Obs
		wantValid bool
		want, tol float64
	}{
		{
			name: "present derives VPD from T and RH",
			obs: firewx.Obs{
				Temperature:      firewx.Some(firewx.Celsius(25)),
				RelativeHumidity: firewx.Some(firewx.Percent(40)),
				WindSpeed:        firewx.Some(firewx.MetersPerSecond(5)),
			},
			wantValid: true,
			want:      95.04,
			tol:       0.05,
		},
		{
			name: "absent without the humidity needed for VPD",
			obs: firewx.Obs{
				Temperature: firewx.Some(firewx.Celsius(25)),
				WindSpeed:   firewx.Some(firewx.MetersPerSecond(5)),
			},
			wantValid: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := HDWFromObs(tc.obs)
			if got.Valid() != tc.wantValid {
				t.Fatalf("%s: Valid()=%v, want %v", tc.name, got.Valid(), tc.wantValid)
			}
			if tc.wantValid {
				closeTo(t, float64(got.Must()), tc.want, tc.tol, tc.name)
			}
		})
	}
}
