package nelson

import "math"

// TimeLag is a standard dead fuel moisture time-lag class. The time lag is the
// time a stick needs to lose about two thirds of the difference between its
// moisture and the equilibrium moisture.
type TimeLag int

const (
	OneHour TimeLag = iota
	TenHour
	HundredHour
	ThousandHour
)

// standardStick holds the fixed parameters of a standard time-lag stick, ported
// from the firelab/NFDRS4 stick initializers.
type standardStick struct {
	radius           float64 // stick radius (cm)
	adsorptionRate   float64 // surface mass transfer rate for adsorption
	maxLocalMoisture float64 // maximum local moisture content (g/g)
}

// standardSticks gives the fixed parameters of the four standard sticks.
var standardSticks = map[TimeLag]standardStick{
	OneHour:      {radius: 0.20, adsorptionRate: 0.462252733, maxLocalMoisture: 0.35},
	TenHour:      {radius: 0.64, adsorptionRate: 0.079548303, maxLocalMoisture: 0.35},
	HundredHour:  {radius: 2.00, adsorptionRate: 0.060000000, maxLocalMoisture: 0.35},
	ThousandHour: {radius: 3.81, adsorptionRate: 0.060000000, maxLocalMoisture: 0.35},
}

// The following functions derive stick parameters from the radius, after
// Bevins (2005), as coded in firelab/NFDRS4. The radius is in centimetres.

// deriveAdsorptionRate returns the surface mass transfer rate for adsorption.
func deriveAdsorptionRate(radius float64) float64 {
	return 0.06 + 0.006126/math.Pow(radius, 2.6)
}

// derivePlanarHeatTransferRate returns the planar heat transfer rate.
func derivePlanarHeatTransferRate(radius float64) float64 {
	return 0.2195 + 0.05260/math.Pow(radius, 2.5)
}

// deriveRainfallRunoffFactor returns the initial rainfall runoff factor.
func deriveRainfallRunoffFactor(radius float64) float64 {
	return 0.02822 + 0.1056/math.Pow(radius, 2.2)
}

// deriveDiffusivitySteps returns the number of diffusivity computation steps per
// observation.
func deriveDiffusivitySteps(radius float64) int {
	return int(4.777 + 2.496/math.Pow(radius, 1.3))
}

// deriveMoistureSteps returns the number of moisture content computation steps
// per observation.
func deriveMoistureSteps(radius float64) int {
	return int(9.8202 + 26.865/math.Pow(radius, 1.4))
}

// deriveStickNodes returns the number of radial computation nodes. The count is
// always odd, so the stick has a centre node.
func deriveStickNodes(radius float64) int {
	nodes := int(10.727 + 0.1746/radius)
	if nodes%2 == 0 {
		nodes++
	}
	return nodes
}
