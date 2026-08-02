package simple

import (
	"testing"

	firewx "alpineworks.io/firewx"
)

func TestFosbergGolden(t *testing.T) {
	// T=85F, RH=20%, wind=15mph. RH is in the 10-50 EMC band. Recomputed
	// independently against the Fosberg (1978) equations: 37.53.
	got := FosbergIndex(85, 20, 15)
	closeTo(t, float64(got), 37.53, 0.05, "FFWI(85F,20%,15mph)")
}

func TestFosbergHighHumidityBranch(t *testing.T) {
	// RH=70% exercises the EMC branch for RH>50. T=60F, wind=10mph -> 12.49.
	got := FosbergIndex(60, 70, 10)
	closeTo(t, float64(got), 12.49, 0.05, "FFWI(60F,70%,10mph) high-RH branch")
}

func TestFosbergSaturationClampsToZero(t *testing.T) {
	// A saturated, very cold input drives the equilibrium moisture content above
	// 30 percent. The clamp holds it at 30, where the moisture term is zero, so
	// the index is zero. This is the only path that exercises the m>30 clamp.
	if emc := fosbergEMC(-60, 100); emc <= 30 {
		t.Fatalf("test setup: expected EMC>30 to exercise the clamp, got %v", emc)
	}
	got := FosbergIndex(-60, 100, 10)
	closeTo(t, float64(got), 0, 1e-9, "FFWI at saturation clamps to zero")
}

func TestFosbergCalibration(t *testing.T) {
	// The index is calibrated so that near-zero fuel moisture in a 30 mph wind
	// approaches the extreme value 100. At T=110F/RH=0 the EMC is 0.032 (not
	// clamped), giving 99.77.
	got := FosbergIndex(110, 0, 30)
	closeTo(t, float64(got), 99.77, 0.1, "FFWI calibration near 100")
}

func TestFosbergMonotonic(t *testing.T) {
	if FosbergIndex(85, 15, 10) <= FosbergIndex(85, 40, 10) {
		t.Error("FFWI should rise as RH falls")
	}
	if FosbergIndex(85, 20, 20) <= FosbergIndex(85, 20, 10) {
		t.Error("FFWI should rise as wind rises")
	}
}

func TestFosbergFromObs(t *testing.T) {
	o := firewx.Obs{
		Temperature:      firewx.Some(firewx.Celsius(29.4)), // ~85F
		RelativeHumidity: firewx.Some(firewx.Percent(20)),
	}
	if FosbergFromObs(o).Valid() {
		t.Error("Fosberg should be absent without wind")
	}
	o.WindSpeed = firewx.Some(firewx.MetersPerSecond(6.7056)) // exactly 15 mph
	got, ok := FosbergFromObs(o).Get()
	if !ok {
		t.Fatal("Fosberg should be present with all inputs")
	}
	// The FromObs path must convert SI to the equation's units. 29.4C=84.92F,
	// 6.7056 m/s=15 mph; the value must track the pure function on those inputs.
	want := FosbergIndex(firewx.Celsius(29.4).Fahrenheit(), 20, firewx.MetersPerSecond(6.7056).MilesPerHour())
	closeTo(t, float64(got), float64(want), 1e-9, "FosbergFromObs conversion")
}
