package gsi

import (
	"math"
	"time"

	firewx "alpineworks.io/firewx"
)

// Constants of the day length function, from MT-CLIM as coded in
// firelab/NFDRS4.
const (
	radPerDay = 0.017214   // radians the Earth moves in one day
	radPerDeg = 0.01745329 // radians in one degree
	minDecl   = -0.4092797 // minimum solar declination (radians)
	secPerRad = 13750.9871 // seconds in one radian of the Earth's rotation
	daysOff   = 10.25      // days from the winter solstice to the new year
)

// Daylength returns the length of the day for a latitude and a day of the year.
// The latitude is in decimal degrees. The day of the year is 1 to 366. The
// function is the day length function of MT-CLIM.
func Daylength(latitude float64, dayOfYear int) time.Duration {
	lat := latitude * radPerDeg
	// Constrain the latitude to just less than the pole, so the calculation
	// does not fail.
	if lat > 1.5707 {
		lat = 1.5707
	}
	if lat < -1.5707 {
		lat = -1.5707
	}
	coslat := math.Cos(lat)
	sinlat := math.Sin(lat)

	decl := minDecl * math.Cos((float64(dayOfYear)+daysOff)*radPerDay)
	cosdecl := math.Cos(decl)
	sindecl := math.Sin(decl)

	cosegeom := coslat * cosdecl
	sinegeom := sinlat * sindecl
	coshss := -sinegeom / cosegeom
	if coshss < -1.0 {
		coshss = -1.0 // 24-hour day
	}
	if coshss > 1.0 {
		coshss = 1.0 // 0-hour day
	}
	hss := math.Acos(coshss) // hour angle at sunset (radians)
	seconds := 2.0 * hss * secPerRad
	return time.Duration(seconds * float64(time.Second))
}

// SaturationVaporPressure returns the saturation vapor pressure in pascals for
// an air temperature. It uses the equation of firelab/NFDRS4.
func SaturationVaporPressure(t firewx.Celsius) float64 {
	c := float64(t)
	return 610.7 * math.Exp((17.38*c)/(239.0+c))
}

// VaporPressureDeficit returns the vapor pressure deficit in pascals for an air
// temperature and a relative humidity. It is never negative.
func VaporPressureDeficit(t firewx.Celsius, rh firewx.Percent) float64 {
	vp := SaturationVaporPressure(t)
	vpd := vp - (float64(rh)/100.0)*vp
	if vpd < 0.0 {
		vpd = 0.0
	}
	return vpd
}

// tminIndicator returns the minimum temperature indicator (0 to 1). The
// temperature and the limits are in Celsius. Below the low limit the indicator
// is 0. Above the high limit it is 1.
func tminIndicator(tminC, min, max float64) float64 {
	if max == min {
		return 0
	}
	if tminC < min {
		return 0
	}
	if tminC > max {
		return 1
	}
	return (tminC - min) / (max - min)
}

// vpdIndicator returns the vapor pressure deficit indicator (0 to 1). The units
// are pascals. The indicator falls as the deficit rises: below the low limit it
// is 1, above the high limit it is 0.
func vpdIndicator(vpdPa, min, max float64) float64 {
	if max == min {
		return 0
	}
	if vpdPa < min {
		return 1
	}
	if vpdPa > max {
		return 0
	}
	return 1.0 - (vpdPa-min)/(max-min)
}

// daylIndicator returns the day length indicator (0 to 1). The units are
// seconds. Below the low limit the indicator is 0. Above the high limit it is 1.
func daylIndicator(daylSec, min, max float64) float64 {
	if max == min {
		return 0
	}
	if daylSec < min {
		return 0
	}
	if daylSec > max {
		return 1
	}
	return (daylSec - min) / (max - min)
}
