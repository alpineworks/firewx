package simple

import firewx "github.com/alpineworks/firewx"

// HDW is the Hot-Dry-Windy index. It is the product of the vapour pressure
// deficit and the wind speed. It has units of hPa·m/s. It is not a fitted score.
// Use it to rank the days of a forecast against each other. Do not compare it to
// a fixed danger scale.
//
// The published index takes the maximum of the product through the depth of the
// daytime boundary layer. This surface form uses the surface vapour pressure
// deficit and the surface wind. This is the practical approximation when you
// have only station observations.
//
// This package uses the vapour pressure deficit in hectopascals. This agrees
// with Srock et al. 2018 and the USDA HDW product. The firebehavioR R package
// uses kilopascals, and its value is 10 times smaller.
type HDW float64

// HDWIndex calculates the surface Hot-Dry-Windy index. The inputs are a vapour
// pressure deficit in hectopascals and a wind speed in metres per second.
func HDWIndex(vpd firewx.Hectopascals, wind firewx.MetersPerSecond) HDW {
	return HDW(float64(vpd) * float64(wind))
}

// HDWFromObs calculates the index from an observation. The function derives the
// vapour pressure deficit from the temperature and the humidity. The result is
// absent if the temperature, the humidity, or the wind is absent.
func HDWFromObs(o firewx.Obs) firewx.Opt[HDW] {
	vpd, okV := o.VaporPressureDeficit().Get()
	w, okW := o.WindSpeed.Get()
	if !okV || !okW {
		return firewx.None[HDW]()
	}
	return firewx.Some(HDWIndex(vpd.Hectopascals(), w))
}
