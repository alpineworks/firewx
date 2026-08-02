package simple

import (
	"testing"

	firewx "alpineworks.io/firewx"
)

func TestHDWGolden(t *testing.T) {
	// T=25C, RH=40%, wind=5 m/s. VPD ~19.01 hPa, product ~95.04.
	got := HDWIndex(firewx.Hectopascals(19.00893), 5)
	closeTo(t, float64(got), 95.04, 0.05, "HDW pure")

	o := firewx.Obs{
		Temperature:      firewx.Some(firewx.Celsius(25)),
		RelativeHumidity: firewx.Some(firewx.Percent(40)),
		WindSpeed:        firewx.Some(firewx.MetersPerSecond(5)),
	}
	fromObs, ok := HDWFromObs(o).Get()
	if !ok {
		t.Fatal("HDW should be present")
	}
	closeTo(t, float64(fromObs), 95.04, 0.05, "HDW from obs")
}

func TestHDWMonotonicAndZero(t *testing.T) {
	if HDWIndex(0, 5) != 0 {
		t.Error("HDW must be zero when VPD is zero")
	}
	if HDWIndex(19, 10) <= HDWIndex(19, 5) {
		t.Error("HDW should rise with wind")
	}
	if HDWIndex(30, 5) <= HDWIndex(19, 5) {
		t.Error("HDW should rise with VPD")
	}
}

func TestHDWAbsentWithoutHumidity(t *testing.T) {
	o := firewx.Obs{
		Temperature: firewx.Some(firewx.Celsius(25)),
		WindSpeed:   firewx.Some(firewx.MetersPerSecond(5)),
	}
	if HDWFromObs(o).Valid() {
		t.Error("HDW should be absent without the humidity needed for VPD")
	}
}
