package simple

import (
	"fmt"
	"testing"

	firewx "alpineworks.io/firewx"
)

func TestChandlerValues(t *testing.T) {
	cases := []struct {
		name string
		t    firewx.Celsius
		rh   firewx.Percent
		want float64
		tol  float64
	}{
		// Identical to firebehavioR::fireIndex(temp=25, rh=40)$chandler
		// = (((110-1.373*40)-0.54*(10.20-25))*(124*10^(-0.0142*40)))/60.
		{"golden 25C/40%", 25, 40, 35.25, 0.05},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closeTo(t, float64(ChandlerIndex(tc.t, tc.rh)), tc.want, tc.tol, tc.name)
		})
	}
}

func TestChandlerMonotonic(t *testing.T) {
	cases := []struct {
		name   string
		hi, lo Chandler
	}{
		{"rises as RH falls", ChandlerIndex(25, 15), ChandlerIndex(25, 60)},
		// The temperature term -0.54*(10.20-T) must raise CBI as temperature rises.
		{"rises as temperature rises", ChandlerIndex(35, 40), ChandlerIndex(15, 40)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hi <= tc.lo {
				t.Errorf("%s: got hi=%v, lo=%v", tc.name, tc.hi, tc.lo)
			}
		})
	}
}

func TestChandlerClass(t *testing.T) {
	cases := []struct {
		v    Chandler
		want DangerClass
	}{
		{49, ClassLow}, {50, ClassModerate}, {74, ClassModerate},
		{75, ClassHigh}, {89, ClassHigh}, {90, ClassVeryHigh},
		{97.4, ClassVeryHigh}, {97.5, ClassExtreme},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("CBI=%v", tc.v), func(t *testing.T) {
			if got := tc.v.Class(); got != tc.want {
				t.Errorf("Chandler(%v).Class()=%v, want %v", tc.v, got, tc.want)
			}
		})
	}
}

func TestChandlerFromObs(t *testing.T) {
	cases := []struct {
		name      string
		obs       firewx.Obs
		wantValid bool
		want, tol float64
	}{
		{
			name:      "absent without RH",
			obs:       firewx.Obs{Temperature: firewx.Some(firewx.Celsius(25))},
			wantValid: false,
		},
		{
			name:      "present with T and RH",
			obs:       firewx.Obs{Temperature: firewx.Some(firewx.Celsius(25)), RelativeHumidity: firewx.Some(firewx.Percent(40))},
			wantValid: true,
			want:      float64(ChandlerIndex(25, 40)),
			tol:       1e-9,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ChandlerFromObs(tc.obs)
			if got.Valid() != tc.wantValid {
				t.Fatalf("%s: Valid()=%v, want %v", tc.name, got.Valid(), tc.wantValid)
			}
			if tc.wantValid {
				closeTo(t, float64(got.Must()), tc.want, tc.tol, tc.name)
			}
		})
	}
}
