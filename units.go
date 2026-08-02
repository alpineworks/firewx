// Package firewx provides the shared observation and unit types used by the
// fire danger models in this repository.
//
// Every physical quantity has a named type. The public API never accepts a
// bare float64 for a measurement, because the models in this repository
// disagree about units in ways that are easy to get wrong and hard to notice:
// the Canadian FWI system is specified in degrees Celsius, km/h and
// millimetres, while NFDRS and the Rothermel spread model are specified in
// degrees Fahrenheit, mph, inches and imperial fuel loadings.
//
// The convention is that observations are carried in SI, and each model
// converts at its own boundary to whatever its published equations require.
package firewx

import "math"

// ---------------------------------------------------------------------------
// Temperature
// ---------------------------------------------------------------------------

// Celsius is a temperature in degrees Celsius. This is the canonical
// temperature unit for observations.
type Celsius float64

// Fahrenheit is a temperature in degrees Fahrenheit, used by NFDRS and by
// several of the older single-equation indices.
type Fahrenheit float64

// Fahrenheit converts c to degrees Fahrenheit.
func (c Celsius) Fahrenheit() Fahrenheit { return Fahrenheit(float64(c)*9.0/5.0 + 32.0) }

// Celsius converts f to degrees Celsius.
func (f Fahrenheit) Celsius() Celsius { return Celsius((float64(f) - 32.0) * 5.0 / 9.0) }

// Kelvin returns c as an absolute temperature. Used by the radiative terms of
// the Nelson dead fuel moisture model.
func (c Celsius) Kelvin() float64 { return float64(c) + 273.15 }

// ---------------------------------------------------------------------------
// Wind speed
// ---------------------------------------------------------------------------

// MetersPerSecond is a wind speed. This is the canonical wind unit for
// observations.
type MetersPerSecond float64

// KilometersPerHour is a wind speed, as required by the Canadian FWI system's
// Initial Spread Index.
type KilometersPerHour float64

// MilesPerHour is a wind speed, as required by NFDRS and the Fosberg index.
type MilesPerHour float64

// KilometersPerHour converts v to kilometres per hour.
func (v MetersPerSecond) KilometersPerHour() KilometersPerHour {
	return KilometersPerHour(float64(v) * 3.6)
}

// MilesPerHour converts v to miles per hour.
func (v MetersPerSecond) MilesPerHour() MilesPerHour {
	return MilesPerHour(float64(v) * 2.2369362920544)
}

// MetersPerSecond converts v to metres per second.
func (v KilometersPerHour) MetersPerSecond() MetersPerSecond {
	return MetersPerSecond(float64(v) / 3.6)
}

// MetersPerSecond converts v to metres per second.
func (v MilesPerHour) MetersPerSecond() MetersPerSecond {
	return MetersPerSecond(float64(v) / 2.2369362920544)
}

// ---------------------------------------------------------------------------
// Precipitation and length
// ---------------------------------------------------------------------------

// Millimeters is a depth of precipitation, or a length. This is the canonical
// precipitation unit for observations.
type Millimeters float64

// Inches is a depth of precipitation, as required by NFDRS and the
// Keetch-Byram Drought Index.
type Inches float64

// Inches converts m to inches.
func (m Millimeters) Inches() Inches { return Inches(float64(m) / 25.4) }

// Millimeters converts i to millimetres.
func (i Inches) Millimeters() Millimeters { return Millimeters(float64(i) * 25.4) }

// Meters is a length, used for sensor heights and station elevation.
type Meters float64

// Feet is a length. NFDRS specifies its reference anemometer height in feet.
type Feet float64

// Feet converts m to feet.
func (m Meters) Feet() Feet { return Feet(float64(m) / 0.3048) }

// Meters converts f to metres.
func (f Feet) Meters() Meters { return Meters(float64(f) * 0.3048) }

// ---------------------------------------------------------------------------
// Dimensionless and derived
// ---------------------------------------------------------------------------

// Percent is a relative humidity or a fuel moisture content, expressed as a
// percentage. Fuel moisture content routinely exceeds 100% and may reach
// several hundred percent for live fuels, so this type is deliberately not
// range-limited.
type Percent float64

// Degrees is a compass bearing, 0-360, measured clockwise from true north.
type Degrees float64

// WattsPerSquareMeter is an irradiance. NFDRS2016's Nelson dead fuel moisture
// model requires measured solar radiation; without it the dominant term in the
// fuel particle energy balance has to be estimated from cloud cover.
type WattsPerSquareMeter float64

// Kilopascals is a pressure, used for vapour pressure and vapour pressure
// deficit.
type Kilopascals float64

// Hectopascals is a pressure, the conventional unit for both station pressure
// and for the Hot-Dry-Windy index's vapour pressure deficit term.
type Hectopascals float64

// Hectopascals converts k to hectopascals.
func (k Kilopascals) Hectopascals() Hectopascals { return Hectopascals(float64(k) * 10.0) }

// Kilopascals converts h to kilopascals.
func (h Hectopascals) Kilopascals() Kilopascals { return Kilopascals(float64(h) / 10.0) }

// ---------------------------------------------------------------------------
// Psychrometrics
// ---------------------------------------------------------------------------

// SaturationVaporPressure returns the saturation vapour pressure at
// temperature t, using the Magnus-Tetens approximation. The coefficients
// (0.6108, 17.27, 237.3) are the form given by Allen et al. 1998, FAO Irrigation
// and Drainage Paper 56, Equation 11.
func SaturationVaporPressure(t Celsius) Kilopascals {
	return Kilopascals(0.6108 * math.Exp(17.27*float64(t)/(float64(t)+237.3)))
}

// VaporPressureDeficit returns the difference between the saturation vapour
// pressure at t and the actual vapour pressure implied by rh.
//
// VPD is a stronger predictor of fire activity in the Pacific Northwest than
// relative humidity alone, and is the moisture term of the Hot-Dry-Windy index.
func VaporPressureDeficit(t Celsius, rh Percent) Kilopascals {
	es := SaturationVaporPressure(t)
	return Kilopascals(float64(es) * (1.0 - float64(rh)/100.0))
}

// DewPoint returns the dew point temperature for t and rh, inverting the
// Magnus-Tetens approximation. Results below roughly -45C or above the ambient
// temperature should be treated as out of range.
func DewPoint(t Celsius, rh Percent) Celsius {
	if rh <= 0 {
		return Celsius(math.NaN())
	}
	const a, b = 17.27, 237.3
	gamma := a*float64(t)/(b+float64(t)) + math.Log(float64(rh)/100.0)
	return Celsius(b * gamma / (a - gamma))
}

// RelativeHumidity returns the relative humidity implied by a temperature and
// dew point. Some data sources report dew point rather than RH.
func RelativeHumidity(t, dew Celsius) Percent {
	es := SaturationVaporPressure(t)
	e := SaturationVaporPressure(dew)
	return Percent(100.0 * float64(e) / float64(es))
}
