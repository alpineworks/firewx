package firewx

import (
	"fmt"
	"testing"
)

func TestTemperatureConversion(t *testing.T) {
	cases := []struct {
		name string
		c, f float64
	}{
		{"freezing", 0, 32},
		{"boiling", 100, 212},
		{"equal point", -40, -40},
		{"body temperature", 37, 98.6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closeTo(t, float64(Celsius(tc.c).Fahrenheit()), tc.f, 1e-9, "C->F")
			closeTo(t, float64(Fahrenheit(tc.f).Celsius()), tc.c, 1e-9, "F->C")
		})
	}
}

func TestWindConversion(t *testing.T) {
	v := MetersPerSecond(7.3)
	cases := []struct {
		name           string
		got, want, tol float64
	}{
		{"m/s->km/h->m/s", float64(v.KilometersPerHour().MetersPerSecond()), 7.3, 1e-9},
		{"m/s->mph->m/s", float64(v.MilesPerHour().MetersPerSecond()), 7.3, 1e-9},
		{"m/s->km/h", float64(v.KilometersPerHour()), 26.28, 1e-9},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closeTo(t, tc.got, tc.want, tc.tol, tc.name)
		})
	}
}

func TestPrecipConversion(t *testing.T) {
	cases := []struct {
		name           string
		got, want, tol float64
	}{
		{"mm->in", float64(Millimeters(25.4).Inches()), 1.0, 1e-12},
		{"in->mm", float64(Inches(1).Millimeters()), 25.4, 1e-12},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closeTo(t, tc.got, tc.want, tc.tol, tc.name)
		})
	}
}

func TestDewPointInvertsRelativeHumidity(t *testing.T) {
	cases := []struct {
		temp Celsius
		rh   Percent
	}{
		{-10, 15}, {-10, 45}, {-10, 90},
		{0, 15}, {0, 45}, {0, 90},
		{12.5, 15}, {12.5, 45}, {12.5, 90},
		{30, 15}, {30, 45}, {30, 90},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("T=%v/RH=%v", tc.temp, tc.rh), func(t *testing.T) {
			dew := DewPoint(tc.temp, tc.rh)
			back := RelativeHumidity(tc.temp, dew)
			closeTo(t, float64(back), float64(tc.rh), 1e-6, "RH round trip")
		})
	}
}

func TestVaporPressureDeficitValues(t *testing.T) {
	cases := []struct {
		name string
		t    Celsius
		rh   Percent
		want float64
		tol  float64
	}{
		{"zero at saturation", 20, 100, 0, 1e-12},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closeTo(t, float64(VaporPressureDeficit(tc.t, tc.rh)), tc.want, tc.tol, tc.name)
		})
	}
}

func TestVaporPressureDeficitMonotonic(t *testing.T) {
	cases := []struct {
		name   string
		hi, lo Kilopascals
	}{
		{"rises with temperature at fixed RH", VaporPressureDeficit(30, 20), VaporPressureDeficit(20, 20)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.hi <= tc.lo {
				t.Errorf("%s: got hi=%v, lo=%v", tc.name, tc.hi, tc.lo)
			}
		})
	}
}
