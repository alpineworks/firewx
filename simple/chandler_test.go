package simple

import (
	"testing"

	firewx "alpineworks.io/firewx"
)

func TestChandlerGolden(t *testing.T) {
	// T=25C, RH=40%. Identical to firebehavioR::fireIndex(temp=25, rh=40)$chandler
	// = (((110-1.373*40)-0.54*(10.20-25))*(124*10^(-0.0142*40)))/60 = 35.25.
	got := ChandlerIndex(25, 40)
	closeTo(t, float64(got), 35.25, 0.05, "CBI(25C,40%)")
}

func TestChandlerMonotonic(t *testing.T) {
	if ChandlerIndex(25, 15) <= ChandlerIndex(25, 60) {
		t.Error("CBI should rise as RH falls")
	}
	// The temperature term -0.54*(10.20-T) must raise CBI as temperature rises.
	if ChandlerIndex(35, 40) <= ChandlerIndex(15, 40) {
		t.Error("CBI should rise as temperature rises")
	}
}

func TestChandlerClassBoundaries(t *testing.T) {
	cases := []struct {
		v    Chandler
		want DangerClass
	}{
		{49, ClassLow}, {50, ClassModerate}, {74, ClassModerate},
		{75, ClassHigh}, {89, ClassHigh}, {90, ClassVeryHigh},
		{97.4, ClassVeryHigh}, {97.5, ClassExtreme},
	}
	for _, c := range cases {
		if got := c.v.Class(); got != c.want {
			t.Errorf("Chandler(%v).Class()=%v, want %v", c.v, got, c.want)
		}
	}
}

func TestChandlerFromObs(t *testing.T) {
	o := firewx.Obs{Temperature: firewx.Some(firewx.Celsius(25))}
	if ChandlerFromObs(o).Valid() {
		t.Error("Chandler should be absent without RH")
	}
	o.RelativeHumidity = firewx.Some(firewx.Percent(40))
	got, ok := ChandlerFromObs(o).Get()
	if !ok {
		t.Fatal("Chandler should be present with T and RH")
	}
	closeTo(t, float64(got), float64(ChandlerIndex(25, 40)), 1e-9, "ChandlerFromObs value")
}
