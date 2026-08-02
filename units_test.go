package firewx

import "testing"

func TestTemperatureConversion(t *testing.T) {
	cases := []struct{ c, f float64 }{
		{0, 32}, {100, 212}, {-40, -40}, {37, 98.6},
	}
	for _, tc := range cases {
		closeTo(t, float64(Celsius(tc.c).Fahrenheit()), tc.f, 1e-9, "C->F")
		closeTo(t, float64(Fahrenheit(tc.f).Celsius()), tc.c, 1e-9, "F->C")
	}
}

func TestWindConversionRoundTrip(t *testing.T) {
	v := MetersPerSecond(7.3)
	closeTo(t, float64(v.KilometersPerHour().MetersPerSecond()), 7.3, 1e-9, "m/s->km/h->m/s")
	closeTo(t, float64(v.MilesPerHour().MetersPerSecond()), 7.3, 1e-9, "m/s->mph->m/s")
	closeTo(t, float64(v.KilometersPerHour()), 26.28, 1e-9, "m/s->km/h")
}

func TestPrecipConversion(t *testing.T) {
	closeTo(t, float64(Millimeters(25.4).Inches()), 1.0, 1e-12, "mm->in")
	closeTo(t, float64(Inches(1).Millimeters()), 25.4, 1e-12, "in->mm")
}

func TestDewPointInvertsRelativeHumidity(t *testing.T) {
	for _, temp := range []Celsius{-10, 0, 12.5, 30} {
		for _, rh := range []Percent{15, 45, 90} {
			dew := DewPoint(temp, rh)
			back := RelativeHumidity(temp, dew)
			closeTo(t, float64(back), float64(rh), 1e-6, "RH round trip")
		}
	}
}

func TestVaporPressureDeficitIsZeroAtSaturation(t *testing.T) {
	closeTo(t, float64(VaporPressureDeficit(20, 100)), 0, 1e-12, "VPD at 100% RH")
	if VaporPressureDeficit(30, 20) <= VaporPressureDeficit(20, 20) {
		t.Error("VPD should increase with temperature at fixed RH")
	}
}
