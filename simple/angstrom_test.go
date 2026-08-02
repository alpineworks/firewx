package simple

import (
	"testing"

	firewx "alpineworks.io/firewx"
)

func TestAngstromGolden(t *testing.T) {
	// Identical to firebehavioR::fireIndex(temp=25, rh=40)$angstrom
	// = 40/20 + (27-25)/10 = 2.2.
	got := AngstromIndex(25, 40)
	closeTo(t, float64(got), 2.2, 1e-9, "Angstrom(25C,40%)")
}

func TestAngstromInverted(t *testing.T) {
	// A lower index means greater danger, so hotter and drier lowers it.
	if AngstromIndex(35, 20) >= AngstromIndex(15, 80) {
		t.Error("Angstrom should fall as conditions worsen")
	}
}

func TestAngstromClassBoundaries(t *testing.T) {
	cases := []struct {
		v    Angstrom
		want DangerClass
	}{
		{4.1, ClassLow}, {4.0, ClassModerate}, {2.6, ClassModerate},
		{2.5, ClassHigh}, {2.1, ClassHigh}, {2.0, ClassVeryHigh}, {0.7, ClassVeryHigh},
	}
	for _, c := range cases {
		if got := c.v.Class(); got != c.want {
			t.Errorf("Angstrom(%v).Class()=%v, want %v", c.v, got, c.want)
		}
	}
}

func TestAngstromFromObs(t *testing.T) {
	o := firewx.Obs{RelativeHumidity: firewx.Some(firewx.Percent(40))}
	if AngstromFromObs(o).Valid() {
		t.Error("Angstrom should be absent without temperature")
	}
	o.Temperature = firewx.Some(firewx.Celsius(25))
	got, ok := AngstromFromObs(o).Get()
	if !ok {
		t.Fatal("Angstrom should be present with T and RH")
	}
	closeTo(t, float64(got), 2.2, 1e-9, "AngstromFromObs value")
}
