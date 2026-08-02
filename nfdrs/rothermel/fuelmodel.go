package rothermel

import firewx "alpineworks.io/firewx"

// Particle is one fuel size class in a fuel bed. The load and the
// surface-area-to-volume ratio are required. The other fields have standard
// defaults; a zero value means "use the default".
type Particle struct {
	// Load is the oven-dry fuel load (lb/ft2).
	Load FuelLoad
	// SAV is the surface-area-to-volume ratio (1/ft).
	SAV SAV

	// HeatContent is the low heat of combustion (BTU/lb). The default is 8000.
	HeatContent HeatContent
	// Density is the oven-dry particle density (lb/ft3). The default is 32.
	Density float64
	// TotalMineral is the total mineral content (fraction). The default is
	// 0.0555.
	TotalMineral float64
	// EffectiveMineral is the effective mineral content (fraction). The default
	// is 0.010.
	EffectiveMineral float64
}

// heatContent returns the heat content, or the standard value if the field is
// zero.
func (p Particle) heatContent() float64 {
	if p.HeatContent == 0 {
		return float64(StandardHeatContent)
	}
	return float64(p.HeatContent)
}

// density returns the particle density, or the standard value if the field is
// zero.
func (p Particle) density() float64 {
	if p.Density == 0 {
		return standardParticleDensity
	}
	return p.Density
}

// totalMineral returns the total mineral content, or the standard value if the
// field is zero.
func (p Particle) totalMineral() float64 {
	if p.TotalMineral == 0 {
		return standardTotalMineral
	}
	return p.TotalMineral
}

// effectiveMineral returns the effective mineral content, or the standard value
// if the field is zero.
func (p Particle) effectiveMineral() float64 {
	if p.EffectiveMineral == 0 {
		return standardEffectiveMineral
	}
	return p.EffectiveMineral
}

// FuelModel is a fuel bed for the Rothermel model. The dead particles and the
// live particles are the two categories. The depth is the height of the fuel
// bed. The dead moisture of extinction is the dead fuel moisture above which the
// fire does not spread.
type FuelModel struct {
	Name string

	// Dead holds the dead fuel size classes.
	Dead []Particle
	// Live holds the live fuel size classes. It may be empty.
	Live []Particle

	// Depth is the fuel bed depth.
	Depth firewx.Feet
	// DeadMoistureOfExtinction is the dead fuel moisture of extinction.
	DeadMoistureOfExtinction firewx.Percent
}

// Moisture is the fuel moisture for a spread computation. The dead and live
// slices align with the Dead and Live particles of the fuel model.
type Moisture struct {
	Dead []firewx.Percent
	Live []firewx.Percent
}

// Conditions is the environment for a spread computation.
type Conditions struct {
	Moisture Moisture

	// MidflameWind is the wind speed at the midflame height.
	MidflameWind firewx.MilesPerHour

	// Slope is the terrain slope angle. Zero is flat ground.
	Slope firewx.Degrees

	// ApplyWindLimit limits the wind factor to the wind limit of Andrews (2018),
	// equation 86. Above the limit, more wind does not increase the rate of
	// spread. The default is false, which matches the current BehavePlus
	// default. The National Fire Danger Rating System sets it to true.
	ApplyWindLimit bool
}
