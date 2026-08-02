package fwi

import (
	"math"

	firewx "alpineworks.io/firewx"
)

// ffmcCoefficient scales the Fine Fuel Moisture Code to and from a moisture
// content. It is the value 250*59.5/101 from Van Wagner and Pickett (1985).
const ffmcCoefficient = 250.0 * 59.5 / 101.0

// FFMC is the Fine Fuel Moisture Code. It tracks the moisture of the fine
// surface litter and reacts within about two thirds of a day. The scale runs
// from 0 to 101; a higher value means drier fuel and easier ignition.
type FFMC float64

// ffmcMoisture converts a Fine Fuel Moisture Code to its moisture content in
// percent. The Initial Spread Index uses the same conversion.
func ffmcMoisture(f FFMC) float64 {
	return ffmcCoefficient * (101 - float64(f)) / (59.5 + float64(f))
}

// FineFuelMoistureCode computes today's Fine Fuel Moisture Code from yesterday's
// code and today's noon weather: temperature (Celsius), relative humidity
// (percent), wind speed (kilometres per hour), and 24-hour rainfall
// (millimetres).
//
// Reference: Van Wagner and Pickett 1985, Forestry Technical Report 33. The
// equations match the cffdrs R package.
func FineFuelMoistureCode(prev FFMC, t firewx.Celsius, rh firewx.Percent, wind firewx.KilometersPerHour, rain firewx.Millimeters) FFMC {
	temp := float64(t)
	h := float64(rh)
	w := float64(wind)
	ro := float64(rain)

	wmo := ffmcMoisture(prev)

	if ro > 0.5 {
		ra := ro - 0.5
		// The rain term and the >150 correction both use the pre-rain moisture.
		rain := 42.5 * ra * math.Exp(-100/(251-wmo)) * (1 - math.Exp(-6.93/ra))
		if wmo > 150 {
			wmo = wmo + rain + 0.0015*(wmo-150)*(wmo-150)*math.Sqrt(ra)
		} else {
			wmo = wmo + rain
		}
	}
	if wmo > 250 {
		wmo = 250
	}

	ed := 0.942*math.Pow(h, 0.679) + 11*math.Exp((h-100)/10) + 0.18*(21.1-temp)*(1-math.Exp(-0.115*h))
	ew := 0.618*math.Pow(h, 0.753) + 10*math.Exp((h-100)/10) + 0.18*(21.1-temp)*(1-math.Exp(-0.115*h))

	var wm float64
	switch {
	case wmo > ed:
		// Drying: the fuel is wetter than the equilibrium, so it loses water.
		ko := 0.424*(1-math.Pow(h/100, 1.7)) + 0.0694*math.Sqrt(w)*(1-math.Pow(h/100, 8))
		kd := ko * 0.581 * math.Exp(0.0365*temp)
		wm = ed + (wmo-ed)/math.Pow(10, kd)
	case wmo < ew:
		// Wetting: the fuel is drier than the equilibrium, so it gains water.
		kl := 0.424*(1-math.Pow((100-h)/100, 1.7)) + 0.0694*math.Sqrt(w)*(1-math.Pow((100-h)/100, 8))
		kw := kl * 0.581 * math.Exp(0.0365*temp)
		wm = ew - (ew-wmo)/math.Pow(10, kw)
	default:
		wm = wmo
	}

	f := 59.5 * (250 - wm) / (ffmcCoefficient + wm)
	if f > 101 {
		f = 101
	}
	if f < 0 {
		f = 0
	}
	return FFMC(f)
}
