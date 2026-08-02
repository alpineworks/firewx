package simple

import (
	"math"

	firewx "github.com/alpineworks/firewx"
)

// Fosberg is the Fosberg Fire Weather Index (FFWI). The index is a number
// without units. It combines the temperature, the relative humidity, and the
// wind into one measure. The measure shows how easily the fine dead fuels carry
// a fire now.
//
// The index has a calibration point. A wind of 30 mph over fuel with no moisture
// gives an index of 100. This is an extreme condition. The index has no upper
// limit. A very strong wind can make the index more than 100.
type Fosberg float64

// FosbergIndex calculates the Fosberg Fire Weather Index. The inputs are the
// temperature in degrees Fahrenheit, the relative humidity in percent, and the
// wind speed in miles per hour. These are the units of the published equations.
//
// Reference: Fosberg, M.A. 1978. The code limits the equilibrium moisture
// content to 30 percent. At 30 percent the moisture term is zero and a fire does
// not spread. Above 30 percent the published equation gives a negative value.
//
// This code uses the original Fosberg equilibrium moisture content. The
// firebehavioR R package uses the Simard fuel moisture model in its place, so
// its Fosberg values differ from these.
func FosbergIndex(t firewx.Fahrenheit, rh firewx.Percent, wind firewx.MilesPerHour) Fosberg {
	m := fosbergEMC(float64(t), float64(rh))
	if m > 30 {
		m = 30
	}
	r := m / 30
	eta := 1 - 2*r + 1.5*r*r - 0.5*r*r*r
	u := float64(wind)
	return Fosberg(eta * math.Sqrt(1+u*u) / 0.3002)
}

// fosbergEMC gives the equilibrium moisture content in percent. It is a
// piecewise function of the relative humidity h in percent and the temperature t
// in degrees Fahrenheit.
func fosbergEMC(t, h float64) float64 {
	switch {
	case h < 10:
		return 0.03229 + 0.281073*h - 0.000578*h*t
	case h <= 50:
		return 2.22749 + 0.160107*h - 0.014784*t
	default:
		return 21.0606 + 0.005565*h*h - 0.00035*h*t - 0.483199*h
	}
}

// FosbergFromObs calculates the index from an observation. It changes the SI
// fields to the units of the equation. It returns an absent Opt if the
// temperature, the humidity, or the wind is absent.
//
// The function uses the wind as measured. It does not correct the wind for the
// height of the anemometer. For a sheltered site, first correct the wind to the
// 20 ft reference height with Obs.WindAt, then make the index.
func FosbergFromObs(o firewx.Obs) firewx.Opt[Fosberg] {
	t, okT := o.Temperature.Get()
	rh, okRH := o.RelativeHumidity.Get()
	w, okW := o.WindSpeed.Get()
	if !okT || !okRH || !okW {
		return firewx.None[Fosberg]()
	}
	return firewx.Some(FosbergIndex(t.Fahrenheit(), rh, w.MilesPerHour()))
}
