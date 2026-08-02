package simple

import firewx "github.com/alpineworks/firewx"

// Angstrom is the Angström index. It is a Swedish index. It is simple, and a
// person can calculate it without a computer. A low value means more danger.
// This is the opposite of the other indices in this package.
type Angstrom float64

// AngstromIndex calculates the Angström index. The inputs are the temperature in
// degrees Celsius and the relative humidity in percent. Usually these are the
// values at 13:00.
//
//	I = RH/20 + (27 - T)/10
func AngstromIndex(t firewx.Celsius, rh firewx.Percent) Angstrom {
	return Angstrom(float64(rh)/20 + (27-float64(t))/10)
}

// Class gives the danger category. The index is inverse to the danger.
// Therefore the limits decrease. Above 4.0 is low. 2.5 to 4.0 is moderate. 2.0
// to 2.5 is high. 2.0 or less is very high.
func (a Angstrom) Class() DangerClass {
	switch {
	case a > 4.0:
		return ClassLow
	case a > 2.5:
		return ClassModerate
	case a > 2.0:
		return ClassHigh
	default:
		return ClassVeryHigh
	}
}

// AngstromFromObs calculates the index from an observation. It returns an absent
// Opt if the temperature or the humidity is absent.
func AngstromFromObs(o firewx.Obs) firewx.Opt[Angstrom] {
	t, okT := o.Temperature.Get()
	rh, okRH := o.RelativeHumidity.Get()
	if !okT || !okRH {
		return firewx.None[Angstrom]()
	}
	return firewx.Some(AngstromIndex(t, rh))
}
