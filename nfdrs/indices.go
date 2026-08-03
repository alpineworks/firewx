package nfdrs

import (
	"math"

	firewx "alpineworks.io/firewx"
)

// Physical constants of the index equations, from firelab/NFDRS4.
const (
	totalMineral      = 0.0555    // total mineral content of a fuel particle (fraction)
	particleDensity   = 32.0      // oven-dry particle density (lb/ft3)
	mineralDamping    = 0.4173969 // mineral damping coefficient (dimensionless)
	tonsPerAcreToLoad = 2000.0 / 43560.0
	droughtMaxKBDI    = 800.0 // the drought index at the maximum drought load
)

// Indices holds the four fire danger indices of the National Fire Danger Rating
// System.
type Indices struct {
	// SpreadComponent is the forward rate of spread of the head fire, related
	// to feet per minute.
	SpreadComponent float64 `json:"spread_component"`
	// EnergyReleaseComponent is the energy release per unit area of the flaming
	// front.
	EnergyReleaseComponent float64 `json:"energy_release_component"`
	// BurningIndex combines the spread component and the energy release
	// component.
	BurningIndex float64 `json:"burning_index"`
	// IgnitionComponent is the chance that a firebrand starts a fire, from 0 to
	// 100.
	IgnitionComponent float64 `json:"ignition_component"`
}

// Conditions holds the fuel moisture and the weather for an index computation.
// The moistures are percentages of the dry weight.
type Conditions struct {
	// Dead fuel moisture by time-lag class (percent).
	Moisture1, Moisture10, Moisture100, Moisture1000 float64
	// Live fuel moisture (percent).
	MoistureHerb, MoistureWoody float64

	// KBDI is the Keetch-Byram Drought Index (0 to 800).
	KBDI float64
	// KBDIThreshold is the drought index above which the drought fuel load
	// starts. It is a site setting.
	KBDIThreshold float64

	// WindSpeed is the wind speed at the 20-foot height. The model reduces it to
	// the midflame height with the wind reduction factor of the fuel model.
	WindSpeed firewx.MilesPerHour
	// SlopeClass is the slope steepness class, 1 to 5. A value outside this
	// range makes Compute return zero indices.
	SlopeClass int

	// FuelTemperature is the fuel surface temperature for the ignition
	// component. Use the temperature of the Nelson one-hour stick.
	FuelTemperature firewx.Celsius

	// Curing inputs for the herbaceous load transfer. The model ignores them
	// when the fuel model has no herbaceous load.
	GSI, GSIMax, GreenupThreshold float64
}

// slopeFactorForClass returns the slope factor for a slope class, from
// firelab/NFDRS4.
func slopeFactorForClass(class int) float64 {
	switch class {
	case 1:
		return 0.267
	case 2:
		return 0.533
	case 3:
		return 1.068
	case 4:
		return 2.134
	case 5:
		return 4.273
	default:
		return 0.267
	}
}

// dampingSC is the moisture damping coefficient for the spread component, from
// firelab/NFDRS4. The ratio is the moisture divided by the moisture of
// extinction. The result is clamped to the range 0 to 1.
func dampingSC(ratio float64) float64 {
	eta := 1.0 - 2.59*ratio + 5.11*ratio*ratio - 3.52*ratio*ratio*ratio
	return clamp01(eta)
}

// dampingERC is the moisture damping coefficient for the energy release
// component. It uses a different curve from the spread component.
func dampingERC(ratio float64) float64 {
	eta := 1.0 - 2.0*ratio + 1.5*ratio*ratio - 0.5*ratio*ratio*ratio
	return clamp01(eta)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// Compute returns the four fire danger indices for the fuel model under the
// conditions. It is a port of the iCalcIndexes function of firelab/NFDRS4.
//
// Compute returns zero indices for an invalid input, which matches
// firelab/NFDRS4. The slope class must be 1 to 5, the wind speed must not be
// negative, and the fuel bed depth must be more than zero.
func (fm FuelModel) Compute(c Conditions) Indices {
	if c.SlopeClass < 1 || c.SlopeClass > 5 || c.WindSpeed < 0 || fm.Depth <= 0 {
		return Indices{}
	}

	const cta = tonsPerAcreToLoad

	// Constrain the drought index to its defined range of 0 to 800.
	kbdi := clamp(c.KBDI, 0, droughtMaxKBDI)

	// Convert the loads to pounds per square foot.
	w1 := fm.Load1 * cta
	w10 := fm.Load10 * cta
	w100 := fm.Load100 * cta
	w1000 := fm.Load1000 * cta
	wHerb := fm.LoadHerb * cta
	wWood := fm.LoadWoody * cta
	wDrought := fm.LoadDrought * cta
	depth := fm.Depth

	// Add the drought fuel load above the drought threshold. The load is added
	// to the four dead classes in proportion to their loads.
	if kbdi > c.KBDIThreshold {
		wtotd := w1 + w10 + w100
		wtotl := wHerb + wWood
		wtot := wtotd + wtotl
		packing := wtot / depth
		if packing == 0 {
			packing = 1.0
		}
		wtotd += w1000
		droughtUnit := wDrought / (droughtMaxKBDI - c.KBDIThreshold)
		add := (kbdi - c.KBDIThreshold) * droughtUnit
		w1 += (w1 / wtotd) * add
		w10 += (w10 / wtotd) * add
		w100 += (w100 / wtotd) * add
		w1000 += (w1000 / wtotd) * add
		wtot = w1 + w10 + w100 + w1000 + wtotl
		depth = (wtot - w1000) / packing
	}

	// Transfer cured herbaceous fuel to the dead one-hour class.
	w1p := w1
	wHerbP := wHerb
	if wHerb > 0 && c.GSIMax > 0 && c.GreenupThreshold < 1 {
		var cure float64
		if c.GSI < c.GreenupThreshold {
			cure = 1
		} else {
			cure = -1.0/(1.0-c.GreenupThreshold)*(c.GSI/c.GSIMax) + 1.0/(1.0-c.GreenupThreshold)
		}
		cure = clamp01(cure)
		w1p = w1 + wHerb*cure
		wHerbP = wHerb * (1 - cure)
	}

	// Totals and net loads.
	wtotd := w1p + w10 + w100 + w1000
	wtotl := wHerbP + wWood
	wtot := wtotd + wtotl
	w1n := w1p * (1 - totalMineral)
	w10n := w10 * (1 - totalMineral)
	w100n := w100 * (1 - totalMineral)
	wHerbN := wHerbP * (1 - totalMineral)
	wWoodN := wWood * (1 - totalMineral)
	wtotln := wtotl * (1 - totalMineral)

	rhobed := (wtot - w1000) / depth
	rhobar := (wtotl*particleDensity + wtotd*particleDensity) / wtot
	betbar := rhobed / rhobar

	// Live fuel moisture of extinction.
	mxd := fm.ExtinctionMoisture
	mxl := 0.0
	if wtotln > 0 {
		hn1 := w1n * math.Exp(-138.0/fm.SAV1)
		hn10 := w10n * math.Exp(-138.0/fm.SAV10)
		hn100 := w100n * math.Exp(-138.0/fm.SAV100)
		hnHerb := 0.0
		if -500.0/fm.SAVHerb >= -180.218 {
			hnHerb = wHerbN * math.Exp(-500.0/fm.SAVHerb)
		}
		hnWood := 0.0
		if -500.0/fm.SAVWoody >= -180.218 {
			hnWood = wWoodN * math.Exp(-500.0/fm.SAVWoody)
		}
		wrat := 0.0
		if hnHerb+hnWood != 0 {
			wrat = (hn1 + hn10 + hn100) / (hnHerb + hnWood)
		}
		mclfe := (c.Moisture1*hn1 + c.Moisture10*hn10 + c.Moisture100*hn100) / (hn1 + hn10 + hn100)
		mxl = (2.9*wrat*(1.0-mclfe/mxd) - 0.226) * 100
	}
	if mxl < mxd {
		mxl = mxd
	}

	// Surface areas and weighting factors for the spread component.
	sa1 := (w1p / particleDensity) * fm.SAV1
	sa10 := (w10 / particleDensity) * fm.SAV10
	sa100 := (w100 / particleDensity) * fm.SAV100
	saHerb := (wHerbP / particleDensity) * fm.SAVHerb
	saWood := (wWood / particleDensity) * fm.SAVWoody
	sadead := sa1 + sa10 + sa100
	salive := saHerb + saWood
	if sadead <= 0 {
		return Indices{}
	}
	f1 := sa1 / sadead
	f10 := sa10 / sadead
	f100 := sa100 / sadead
	fHerb, fWood := 0.0, 0.0
	if wtotl > 0 {
		fHerb = saHerb / salive
		fWood = saWood / salive
	}
	fdead := sadead / (sadead + salive)
	flive := salive / (sadead + salive)
	wdeadn := f1*w1n + f10*w10n + f100*w100n
	var wliven float64
	if fm.SAVWoody > 1200 && fm.SAVHerb > 1200 {
		wliven = wtotln
	} else {
		wliven = fWood*wWoodN + fHerb*wHerbN
	}

	sgbrd := f1*fm.SAV1 + f10*fm.SAV10 + f100*fm.SAV100
	sgbrl := fHerb*fm.SAVHerb + fWood*fm.SAVWoody
	sgbrt := fdead*sgbrd + flive*sgbrl

	betop := 3.348 * math.Pow(sgbrt, -0.8189)
	sgbrt15 := math.Pow(sgbrt, 1.5)
	gmamx := sgbrt15 / (495.0 + 0.0594*sgbrt15)
	ad := 133.0 * math.Pow(sgbrt, -0.7913)
	gmaop := gmamx * math.Pow(betbar/betop, ad) * math.Exp(ad*(1.0-betbar/betop))
	zeta := math.Exp((0.792+0.681*math.Sqrt(sgbrt))*(betbar+0.1)) / (192.0 + 0.2595*sgbrt)

	wtmcd := f1*c.Moisture1 + f10*c.Moisture10 + f100*c.Moisture100
	wtmcl := fHerb*c.MoistureHerb + fWood*c.MoistureWoody
	etamd := dampingSC(wtmcd / mxd)
	etaml := 0.0
	if mxl > 0 {
		etaml = dampingSC(wtmcl / mxl)
	}

	b := 0.02526 * math.Pow(sgbrt, 0.54)
	cc := 7.47 * math.Exp(-0.133*math.Pow(sgbrt, 0.55))
	e := 0.715 * math.Exp(-3.59e-4*sgbrt)
	ufact := cc * math.Pow(betbar/betop, -e)

	ir := gmaop * (wdeadn*fm.HeatContent*mineralDamping*etamd + wliven*fm.HeatContent*mineralDamping*etaml)

	// Wind factor, with the wind limit.
	ws := math.Trunc(float64(c.WindSpeed))
	var phiwnd float64
	if 88.0*ws*fm.WindReductionFactor > 0.9*ir {
		phiwnd = ufact * math.Pow(0.9*ir, b)
	} else {
		phiwnd = ufact * math.Pow(ws*88.0*fm.WindReductionFactor, b)
	}

	phislp := slopeFactorForClass(c.SlopeClass) * math.Pow(betbar, -0.3)

	xf1 := f1 * math.Exp(-138.0/fm.SAV1) * (250.0 + 11.16*c.Moisture1)
	xf10 := f10 * math.Exp(-138.0/fm.SAV10) * (250.0 + 11.16*c.Moisture10)
	xf100 := f100 * math.Exp(-138.0/fm.SAV100) * (250.0 + 11.16*c.Moisture100)
	xfHerb := fHerb * math.Exp(-138.0/fm.SAVHerb) * (250.0 + 11.16*c.MoistureHerb)
	xfWood := fWood * math.Exp(-138.0/fm.SAVWoody) * (250.0 + 11.16*c.MoistureWoody)
	htsink := rhobed * (fdead*(xf1+xf10+xf100) + flive*(xfHerb+xfWood))

	sc := ir * zeta * (1.0 + phislp + phiwnd) / htsink

	// Energy release component, with the load-fraction weighting.
	f1e := w1p / wtotd
	f10e := w10 / wtotd
	f100e := w100 / wtotd
	f1000e := w1000 / wtotd
	fHerbE, fWoodE := 0.0, 0.0
	if wtotl > 0 {
		fHerbE = wHerbP / wtotl
		fWoodE = wWood / wtotl
	}
	fdeade := wtotd / wtot
	flivee := wtotl / wtot
	wdedne := wtotd * (1 - totalMineral)
	wlivne := wtotl * (1 - totalMineral)
	sgbrde := f1e*fm.SAV1 + f10e*fm.SAV10 + f100e*fm.SAV100 + f1000e*fm.SAV1000
	sgbrle := fHerbE*fm.SAVHerb + fWoodE*fm.SAVWoody
	sgbrte := fdeade*sgbrde + flivee*sgbrle
	betope := 3.348 * math.Pow(sgbrte, -0.8189)
	sgbrte15 := math.Pow(sgbrte, 1.5)
	gmamxe := sgbrte15 / (495.0 + 0.0594*sgbrte15)
	ade := 133.0 * math.Pow(sgbrte, -0.7913)
	gmaope := gmamxe * math.Pow(betbar/betope, ade) * math.Exp(ade*(1.0-betbar/betope))
	wtmcde := f1e*c.Moisture1 + f10e*c.Moisture10 + f100e*c.Moisture100 + f1000e*c.Moisture1000
	wtmcle := fHerbE*c.MoistureHerb + fWoodE*c.MoistureWoody
	etamde := dampingERC(wtmcde / mxd)
	etamle := 0.0
	if mxl > 0 {
		etamle = dampingERC(wtmcle / mxl)
	}
	ire := fdeade * wdedne * fm.HeatContent * mineralDamping * etamde
	ire = gmaope * (ire + flivee*wlivne*fm.HeatContent*mineralDamping*etamle)
	tau := 384.0 / sgbrt
	erc := 0.04 * ire * tau

	bi := 0.301 * math.Pow(sc*erc, 0.46) * 10.0

	ic := ignitionComponent(fm, c, sc)

	return Indices{
		SpreadComponent:        sc,
		EnergyReleaseComponent: erc,
		BurningIndex:           bi,
		IgnitionComponent:      ic,
	}
}

// ignitionComponent computes the ignition component from the fuel surface
// temperature, the one-hour moisture, the spread component, and the maximum
// spread component. It is a port of the ignition component code in
// firelab/NFDRS4.
func ignitionComponent(fm FuelModel, c Conditions, sc float64) float64 {
	const pnorm1, pnorm2 = 0.00232, 0.99767
	if fm.MaxSpreadComponent <= 0 {
		return 0
	}
	tmpprm := float64(c.FuelTemperature)
	mc1 := c.Moisture1
	qign := 144.5 - 0.266*tmpprm - 0.00058*tmpprm*tmpprm - 0.01*tmpprm*mc1 +
		18.54*(1.0-math.Exp(-0.151*mc1)) + 6.4*mc1
	if qign >= 344.0 {
		return 0
	}
	chi := (344.0 - qign) / 10.0
	term := math.Pow(chi, 3.66) * 0.000923 / 50.0
	if term <= pnorm1 {
		return 0
	}
	pi := (term - pnorm1) * 100.0 / pnorm2
	pi = clamp(pi, 0, 100)
	scn := 100.0 * sc / fm.MaxSpreadComponent
	if scn > 100.0 {
		scn = 100.0
	}
	pfi := math.Sqrt(scn)
	ic := 0.10 * pi * pfi
	if sc < 0.00001 {
		ic = 0
	}
	return ic
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
