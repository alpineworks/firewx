package simple

import (
	"fmt"
	"testing"

	firewx "alpineworks.io/firewx"
)

func TestAngstromValues(t *testing.T) {
	cases := []struct {
		name string
		t    firewx.Celsius
		rh   firewx.Percent
		want float64
		tol  float64
	}{
		// Identical to firebehavioR::fireIndex(temp=25, rh=40)$angstrom
		// = 40/20 + (27-25)/10.
		{"golden 25C/40%", 25, 40, 2.2, 1e-9},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closeTo(t, float64(AngstromIndex(tc.t, tc.rh)), tc.want, tc.tol, tc.name)
		})
	}
}

func TestAngstromInverted(t *testing.T) {
	// A lower index means greater danger, so hotter and drier air must produce a
	// smaller index than cool and humid air.
	cases := []struct {
		name   string
		hi, lo Angstrom
	}{
		{"milder air scores higher than worse air", AngstromIndex(15, 80), AngstromIndex(35, 20)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hi <= tc.lo {
				t.Errorf("%s: got hi=%v, lo=%v", tc.name, tc.hi, tc.lo)
			}
		})
	}
}

func TestAngstromClass(t *testing.T) {
	cases := []struct {
		v    Angstrom
		want DangerClass
	}{
		{4.1, ClassLow}, {4.0, ClassModerate}, {2.6, ClassModerate},
		{2.5, ClassHigh}, {2.1, ClassHigh}, {2.0, ClassVeryHigh}, {0.7, ClassVeryHigh},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("I=%v", tc.v), func(t *testing.T) {
			if got := tc.v.Class(); got != tc.want {
				t.Errorf("Angstrom(%v).Class()=%v, want %v", tc.v, got, tc.want)
			}
		})
	}
}

func TestAngstromFromObs(t *testing.T) {
	cases := []struct {
		name      string
		obs       firewx.Obs
		wantValid bool
		want, tol float64
	}{
		{
			name:      "absent without temperature",
			obs:       firewx.Obs{RelativeHumidity: firewx.Some(firewx.Percent(40))},
			wantValid: false,
		},
		{
			name:      "present with T and RH",
			obs:       firewx.Obs{Temperature: firewx.Some(firewx.Celsius(25)), RelativeHumidity: firewx.Some(firewx.Percent(40))},
			wantValid: true,
			want:      2.2,
			tol:       1e-9,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AngstromFromObs(tc.obs)
			if got.Valid() != tc.wantValid {
				t.Fatalf("%s: Valid()=%v, want %v", tc.name, got.Valid(), tc.wantValid)
			}
			if tc.wantValid {
				closeTo(t, float64(got.Must()), tc.want, tc.tol, tc.name)
			}
		})
	}
}
