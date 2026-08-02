package gsi

import (
	"testing"
	"time"

	firewx "alpineworks.io/firewx"
)

// TestDaylengthPhysics checks the day length function against exact physical
// checkpoints that do not depend on the station settings.
//
// Reference: the day length is a function of the latitude and the solar
// declination. The equator has a 12-hour day all year. Every latitude has a
// 12-hour day at an equinox. The Arctic Circle has a 24-hour day at the summer
// solstice and no day at the winter solstice.
func TestDaylengthPhysics(t *testing.T) {
	hours := func(d time.Duration) float64 { return d.Hours() }

	// The equator has a 12-hour day all year.
	closeTo(t, hours(Daylength(0, 15)), 12.0, 0.05, "equator in January")
	closeTo(t, hours(Daylength(0, 200)), 12.0, 0.05, "equator in July")

	// Every latitude has about a 12-hour day at the spring equinox (about day
	// 80) and the autumn equinox (about day 266).
	closeTo(t, hours(Daylength(47.7, 80)), 12.0, 0.25, "47.7 N at the spring equinox")
	closeTo(t, hours(Daylength(-33.9, 266)), 12.0, 0.25, "33.9 S at the autumn equinox")

	// The Arctic Circle has a full day at the summer solstice and no day at the
	// winter solstice.
	closeTo(t, hours(Daylength(66.6, 172)), 24.0, 0.05, "Arctic Circle at the summer solstice")
	closeTo(t, hours(Daylength(66.6, 355)), 0.0, 0.05, "Arctic Circle at the winter solstice")

	// The day is longer at a higher latitude in the northern summer.
	if !(hours(Daylength(60, 172)) > hours(Daylength(30, 172))) {
		t.Errorf("expected a longer summer day at 60 N than at 30 N")
	}
}

// TestSaturationVaporPressure checks the saturation vapor pressure against
// published values.
//
// Reference: the saturation vapor pressure over water is about 611 Pa at 0 C,
// 2338 Pa at 20 C, and 4243 Pa at 30 C (Tetens 1930; Murray 1967). The tolerance
// is 1 percent.
func TestSaturationVaporPressure(t *testing.T) {
	cases := []struct {
		c    float64
		want float64
	}{
		{0, 611.0},
		{10, 1228.0},
		{20, 2338.0},
		{30, 4243.0},
	}
	for _, tc := range cases {
		got := SaturationVaporPressure(firewx.Celsius(tc.c))
		closeTo(t, got, tc.want, tc.want*0.01, "saturation vapor pressure")
	}
}

// TestVaporPressureDeficit checks the vapor pressure deficit. At 100 percent
// humidity the deficit is zero. At a lower humidity the deficit is the
// saturation vapor pressure times one minus the humidity fraction.
func TestVaporPressureDeficit(t *testing.T) {
	closeTo(t, VaporPressureDeficit(20, 100), 0.0, 1e-9, "deficit at saturation")

	svp := SaturationVaporPressure(30)
	closeTo(t, VaporPressureDeficit(30, 40), svp*0.6, 1e-6, "deficit at 40 percent humidity")
	// The deficit is never negative.
	closeTo(t, VaporPressureDeficit(20, 100), 0.0, 1e-9, "deficit is not negative")
}

// TestIndicatorFunctions checks the three GSI indicator ramps against hand
// computation. Each indicator has a range of 0 to 1.
func TestIndicatorFunctions(t *testing.T) {
	// Minimum temperature indicator rises with the temperature.
	closeTo(t, tminIndicator(-5, -2, 5), 0.0, 1e-12, "tmin below the low limit")
	closeTo(t, tminIndicator(10, -2, 5), 1.0, 1e-12, "tmin above the high limit")
	closeTo(t, tminIndicator(1.5, -2, 5), 0.5, 1e-12, "tmin at the midpoint")

	// Vapor pressure deficit indicator falls as the deficit rises.
	closeTo(t, vpdIndicator(500, 900, 4100), 1.0, 1e-12, "vpd below the low limit")
	closeTo(t, vpdIndicator(5000, 900, 4100), 0.0, 1e-12, "vpd above the high limit")
	closeTo(t, vpdIndicator(2500, 900, 4100), 0.5, 1e-12, "vpd at the midpoint")

	// Day length indicator rises with the day length.
	closeTo(t, daylIndicator(30000, 36000, 39600), 0.0, 1e-12, "day length below the low limit")
	closeTo(t, daylIndicator(42000, 36000, 39600), 1.0, 1e-12, "day length above the high limit")
	closeTo(t, daylIndicator(37800, 36000, 39600), 0.5, 1e-12, "day length at the midpoint")
}
