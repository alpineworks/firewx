package rothermel

import (
	"math"

	firewx "alpineworks.io/firewx"
)

// Result holds the output of a spread computation. It carries the rate of spread
// and the reaction intensity, plus the main intermediate values, because the
// National Fire Danger Rating System reuses them and because they make the model
// testable against the worked example in Andrews (2018).
type Result struct {
	// RateOfSpread is the forward rate of spread of the fire head.
	RateOfSpread FeetPerMinute
	// ReactionIntensity is the energy release rate of the flaming front
	// (BTU/ft2-min).
	ReactionIntensity float64

	// CharacteristicSAV is the surface-area weighted surface-area-to-volume
	// ratio of the fuel bed (1/ft).
	CharacteristicSAV float64
	// BulkDensity is the oven-dry fuel load per unit fuel bed volume (lb/ft3).
	BulkDensity float64
	// PackingRatio is the fraction of the fuel bed volume that is fuel.
	PackingRatio float64
	// OptimumPackingRatio is the packing ratio that gives the maximum reaction
	// velocity.
	OptimumPackingRatio float64
	// RelativePackingRatio is the packing ratio divided by the optimum packing
	// ratio.
	RelativePackingRatio float64

	// PropagatingFluxRatio is the fraction of the reaction intensity that heats
	// the adjacent fuel.
	PropagatingFluxRatio float64
	// WindFactor is the fractional increase in the rate of spread from the wind.
	WindFactor float64
	// SlopeFactor is the fractional increase in the rate of spread from the
	// slope.
	SlopeFactor float64
	// HeatSink is the heat needed to bring the fuel to ignition (BTU/ft3).
	HeatSink float64

	// ResidenceTime is the time the flaming front needs to pass a point (min).
	ResidenceTime float64
	// HeatPerUnitArea is the heat release per unit area of the flaming front
	// (BTU/ft2).
	HeatPerUnitArea float64

	// LiveMoistureOfExtinction is the live fuel moisture of extinction that the
	// model computed. It is zero if the fuel model has no live fuel.
	LiveMoistureOfExtinction firewx.Percent
	// WindLimited reports whether the model limited the wind factor. Above the
	// wind limit, more wind does not increase the rate of spread.
	WindLimited bool
}

// category holds the surface-area weighted aggregates of one fuel category
// (dead or live).
type category struct {
	totalArea    float64 // total surface area (eq. 54)
	netLoad      float64 // weighted net fuel load (eq. 59, 60)
	charSAV      float64 // characteristic SAV (eq. 72)
	charMoisture float64 // characteristic moisture content, fraction (eq. 66)
	charHeat     float64 // characteristic heat content (eq. 61)
	charMineral  float64 // characteristic effective mineral content (eq. 63)
	heatSinkTerm float64 // sum of f_i * eps_i * Q_ig_i (part of eq. 77)
	ovenDryLoad  float64 // total oven-dry load
	packingSum   float64 // sum of load_i / density_i, for the packing ratio
}

// aggregate computes the surface-area weighted aggregates of one category. The
// moistures align with the particles and are fractions.
func aggregate(particles []Particle, moistures []float64) category {
	var c category
	areas := make([]float64, len(particles))
	for i, p := range particles {
		// Mean total surface area of the particle (eq. 53).
		areas[i] = float64(p.SAV) * float64(p.Load) / p.density()
		c.totalArea += areas[i]
		c.ovenDryLoad += float64(p.Load)
		c.packingSum += float64(p.Load) / p.density()
	}
	if c.totalArea <= 0 {
		return c
	}
	for i, p := range particles {
		// Weighting factor within the category (eq. 56).
		f := areas[i] / c.totalArea
		sav := float64(p.SAV)
		mf := moistures[i]

		c.charSAV += f * sav                                        // eq. 72
		c.netLoad += f * float64(p.Load) * (1.0 - p.totalMineral()) // eq. 59, 60
		c.charMoisture += f * mf                                    // eq. 66
		c.charHeat += f * p.heatContent()                           // eq. 61
		c.charMineral += f * p.effectiveMineral()                   // eq. 63

		// Heat sink terms of the particle: effective heating number (eq. 14)
		// and heat of preignition (eq. 78).
		eps := math.Exp(-138.0 / sav)
		qig := 250.0 + 1116.0*mf
		c.heatSinkTerm += f * eps * qig // part of eq. 77
	}
	return c
}

// Spread computes the Rothermel rate of spread for the fuel model under the
// conditions. It follows Andrews (2018); the equation numbers in the code refer
// to that report.
//
// The fuel moisture slices in the conditions must align with the dead and live
// particles of the fuel model. Spread panics if they do not, because a mismatch
// is a programming error, not a data error.
func (fm FuelModel) Spread(c Conditions) Result {
	if len(c.Moisture.Dead) != len(fm.Dead) {
		panic("rothermel: dead moisture count does not match the fuel model")
	}
	if len(c.Moisture.Live) != len(fm.Live) {
		panic("rothermel: live moisture count does not match the fuel model")
	}

	deadMf := percentsToFractions(c.Moisture.Dead)
	liveMf := percentsToFractions(c.Moisture.Live)

	dead := aggregate(fm.Dead, deadMf)
	live := aggregate(fm.Live, liveMf)

	var res Result
	totalArea := dead.totalArea + live.totalArea
	if totalArea <= 0 {
		return res
	}

	// Category weighting factors (eq. 57).
	fDead := dead.totalArea / totalArea
	fLive := live.totalArea / totalArea

	// Characteristic surface-area-to-volume ratio of the fuel bed (eq. 71).
	sigma := fDead*dead.charSAV + fLive*live.charSAV
	res.CharacteristicSAV = sigma

	depth := float64(fm.Depth)
	ovenDryTotal := dead.ovenDryLoad + live.ovenDryLoad

	// Mean bulk density (eq. 74).
	res.BulkDensity = ovenDryTotal / depth
	// Mean packing ratio (eq. 73).
	beta := (dead.packingSum + live.packingSum) / depth
	res.PackingRatio = beta

	// Optimum packing ratio (eq. 69).
	betaOp := 3.348 * math.Pow(sigma, -0.8189)
	res.OptimumPackingRatio = betaOp
	relPacking := beta / betaOp
	res.RelativePackingRatio = relPacking

	// Maximum reaction velocity (eq. 68).
	sigma15 := math.Pow(sigma, 1.5)
	gammaMax := sigma15 / (495.0 + 0.0594*sigma15)
	// Optimum reaction velocity (eq. 67).
	aCoef := 133.0 * math.Pow(sigma, -0.7913)
	gamma := gammaMax * math.Pow(relPacking, aCoef) * math.Exp(aCoef*(1.0-relPacking))

	// Propagating flux ratio (eq. 76).
	xi := math.Exp((0.792+0.681*math.Sqrt(sigma))*(beta+0.1)) / (192.0 + 0.2595*sigma)
	res.PropagatingFluxRatio = xi

	// Live fuel moisture of extinction (eq. 88).
	deadMx := float64(fm.DeadMoistureOfExtinction) / 100.0
	liveMx := liveMoistureOfExtinction(fm, deadMf, deadMx)
	res.LiveMoistureOfExtinction = firewx.Percent(liveMx * 100.0)

	// Moisture damping coefficient of each category (eq. 64, 65).
	etaMDead := moistureDamping(dead.charMoisture, deadMx)
	etaMLive := moistureDamping(live.charMoisture, liveMx)

	// Mineral damping coefficient of each category (eq. 62), from the
	// characteristic effective mineral content of the category (eq. 63).
	etaSDead := mineralDamping(dead.charMineral)
	etaSLive := mineralDamping(live.charMineral)

	// Reaction intensity (eq. 58).
	ir := gamma * (dead.netLoad*dead.charHeat*etaMDead*etaSDead +
		live.netLoad*live.charHeat*etaMLive*etaSLive)
	res.ReactionIntensity = ir

	// Heat sink (eq. 77).
	heatSink := res.BulkDensity * (fDead*dead.heatSinkTerm + fLive*live.heatSinkTerm)
	res.HeatSink = heatSink

	// Wind factor (eq. 47-50, 79, 81, 82). The wind speed is in feet per minute.
	u := float64(c.MidflameWind) * feetPerMinutePerMph
	bCoef := 0.02526 * math.Pow(sigma, 0.54)
	cCoef := 7.47 * math.Exp(-0.133*math.Pow(sigma, 0.55))
	eCoef := 0.715 * math.Exp(-3.59e-4*sigma)
	// Wind limit (eq. 86): above 0.9 times the reaction intensity, more wind
	// does not increase the rate of spread. It is off by default.
	if c.ApplyWindLimit {
		windLimit := 0.9 * ir
		if u > windLimit {
			u = windLimit
			res.WindLimited = true
		}
	}
	phiW := cCoef * math.Pow(u, bCoef) * math.Pow(relPacking, -eCoef)
	res.WindFactor = phiW

	// Slope factor (eq. 80).
	tanSlope := math.Tan(float64(c.Slope) * math.Pi / 180.0)
	phiS := 5.275 * math.Pow(beta, -0.3) * tanSlope * tanSlope
	res.SlopeFactor = phiS

	// Rate of spread (eq. 75).
	if heatSink > 0 {
		r0 := ir * xi / heatSink
		res.RateOfSpread = FeetPerMinute(r0 * (1.0 + phiW + phiS))
	}

	// Residence time (Anderson 1969) and heat per unit area.
	res.ResidenceTime = 384.0 / sigma
	res.HeatPerUnitArea = ir * res.ResidenceTime

	return res
}

// moistureDamping returns the moisture damping coefficient for a category
// (eq. 64, 65). The ratio of the moisture content to the moisture of extinction
// is clamped, and the result is clamped to the range 0 to 1.
func moistureDamping(moisture, mx float64) float64 {
	if mx <= 0 {
		return 0
	}
	r := moisture / mx
	// At or above the moisture of extinction the fuel does not react.
	if r >= 1 {
		return 0
	}
	eta := 1.0 - 2.59*r + 5.11*r*r - 3.52*r*r*r
	if eta < 0 {
		eta = 0
	}
	if eta > 1 {
		eta = 1
	}
	return eta
}

// mineralDamping returns the mineral damping coefficient for a category from its
// characteristic effective mineral content (eq. 62). The result is clamped to a
// maximum of 1.
func mineralDamping(effectiveMineral float64) float64 {
	if effectiveMineral <= 0 {
		return 1
	}
	eta := 0.174 * math.Pow(effectiveMineral, -0.19)
	if eta > 1 {
		eta = 1
	}
	return eta
}

// liveMoistureOfExtinction returns the live fuel moisture of extinction
// (eq. 88, after Albini 1976). It is the dead moisture of extinction if the fuel
// model has no live fuel. The dead moistures are fractions.
func liveMoistureOfExtinction(fm FuelModel, deadMf []float64, deadMx float64) float64 {
	if len(fm.Live) == 0 {
		return 0
	}
	// Fine dead fuel heating and moisture, weighted by exp(-138/SAV).
	var deadFine, deadFineMoisture, liveFine float64
	for i, p := range fm.Dead {
		w := float64(p.Load) * (1.0 - p.totalMineral()) * math.Exp(-138.0/float64(p.SAV))
		deadFine += w
		deadFineMoisture += w * deadMf[i]
	}
	for _, p := range fm.Live {
		// Guard against underflow for very coarse live fuel.
		if 500.0/float64(p.SAV) > 180.218 {
			continue
		}
		liveFine += float64(p.Load) * (1.0 - p.totalMineral()) * math.Exp(-500.0/float64(p.SAV))
	}
	if liveFine <= 0 || deadFine <= 0 {
		return deadMx
	}
	wRat := deadFine / liveFine
	mcLifeFine := deadFineMoisture / deadFine
	mx := 2.9*wRat*(1.0-mcLifeFine/deadMx) - 0.226
	if mx < deadMx {
		mx = deadMx
	}
	return mx
}

// percentsToFractions converts a slice of percentages to a slice of fractions.
func percentsToFractions(ps []firewx.Percent) []float64 {
	out := make([]float64, len(ps))
	for i, p := range ps {
		out[i] = float64(p) / 100.0
	}
	return out
}
