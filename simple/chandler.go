package simple

import (
	"math"

	firewx "alpineworks.io/firewx"
)

// Chandler is the Chandler Burning Index (CBI). The index uses the temperature
// and the humidity only. It has no fuel term and no wind term. Many amateur
// weather stations show this index. Use it mainly to compare with those
// stations.
//
// Reference: Chandler et al. 1983, Fire in Forestry, Volume 1. The formula here
// is identical to the firebehavioR R package (Sharples et al. 2009).
type Chandler float64

// ChandlerIndex calculates the Chandler Burning Index. The inputs are the
// temperature in degrees Celsius and the relative humidity in percent.
func ChandlerIndex(t firewx.Celsius, rh firewx.Percent) Chandler {
	h := float64(rh)
	c := float64(t)
	cbi := ((110 - 1.373*h) - 0.54*(10.20-c)) * (124 * math.Pow(10, -0.0142*h)) / 60
	return Chandler(cbi)
}

// Class gives the danger category for the index. The limits are: below 50 is
// low, 50 to 75 is moderate, 75 to 90 is high, 90 to 97.5 is very high, and 97.5
// or more is extreme.
func (c Chandler) Class() DangerClass {
	switch {
	case c < 50:
		return ClassLow
	case c < 75:
		return ClassModerate
	case c < 90:
		return ClassHigh
	case c < 97.5:
		return ClassVeryHigh
	default:
		return ClassExtreme
	}
}

// ChandlerFromObs calculates the index from an observation. It returns an absent
// Opt if the temperature or the humidity is absent.
func ChandlerFromObs(o firewx.Obs) firewx.Opt[Chandler] {
	t, okT := o.Temperature.Get()
	rh, okRH := o.RelativeHumidity.Get()
	if !okT || !okRH {
		return firewx.None[Chandler]()
	}
	return firewx.Some(ChandlerIndex(t, rh))
}
