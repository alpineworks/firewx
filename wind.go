package firewx

import "math"

// Reference anemometer heights assumed by the fire danger models.
const (
	// HeightFWI is the 10 m open exposure assumed by the Canadian FWI system.
	HeightFWI Meters = 10.0

	// HeightNFDRS is the 20 ft exposure assumed by NFDRS and by the RAWS
	// siting standard.
	HeightNFDRS Meters = 6.096
)

// Roughness is an aerodynamic roughness length in metres, characterising how
// much the surrounding terrain and vegetation slow the wind near the ground.
type Roughness float64

// Representative roughness lengths. These are order-of-magnitude guides, not
// measurements; if the corrected wind matters, calibrate against a nearby RAWS
// over a period of similar synoptic conditions rather than trusting a constant.
const (
	RoughnessWater      Roughness = 0.0002
	RoughnessMownGrass  Roughness = 0.01
	RoughnessOpenGrass  Roughness = 0.03 // the reference exposure both systems assume
	RoughnessFarmland   Roughness = 0.10
	RoughnessScattered  Roughness = 0.25
	RoughnessSuburban   Roughness = 0.50
	RoughnessForest     Roughness = 1.00
	RoughnessCityCentre Roughness = 2.00
)

// AdjustWindHeight converts a wind speed measured at height from to the
// equivalent speed at height to, using the logarithmic wind profile.
//
// This corrects for measurement height only. It does not correct for shelter:
// a sensor on a house in a treed neighbourhood is not measuring a slowed
// version of the open-country wind, it is measuring a different flow. Treat
// the result as a defensible approximation for trend purposes rather than as
// equivalent to a properly sited RAWS observation.
func AdjustWindHeight(v MetersPerSecond, from, to Meters, z0 Roughness) MetersPerSecond {
	if from <= 0 || to <= 0 || z0 <= 0 {
		return v
	}
	if from == to {
		return v
	}
	num := math.Log(float64(to) / float64(z0))
	den := math.Log(float64(from) / float64(z0))
	if den == 0 {
		return v
	}
	return MetersPerSecond(float64(v) * num / den)
}

// WindAt returns the observation's wind speed corrected from the station's
// anemometer height to the given reference height, or an empty Opt if the
// observation has no wind measurement.
func (o Obs) WindAt(s Station, h Meters) Opt[MetersPerSecond] {
	v, ok := o.WindSpeed.Get()
	if !ok {
		return None[MetersPerSecond]()
	}
	if s.AnemometerHeight <= 0 {
		return Some(v)
	}
	return Some(AdjustWindHeight(v, s.AnemometerHeight, h, s.Roughness))
}
