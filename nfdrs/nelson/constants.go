package nelson

// Physical constants for the Nelson (2000) dead fuel moisture model. The values
// are ported exactly from the firelab/NFDRS4 C++ program (deadfuelmoisture.cpp),
// so the model output matches that program. Do not change a value; each one is
// a fixed part of the published model.
const (
	aks    = 2.0e-13     // Permeability of water in wood (cm2)
	alb    = 0.6         // Shortwave albedo (dimensionless)
	alpha  = 0.25        // Fraction of cell length that overlaps adjacent cells
	ap     = 0.000772    // Psychrometric constant (1/oC)
	aw     = 0.8         // Ratio of cell cavity to fibre width
	eps    = 0.85        // Long-wave emissivity of the stick surface
	hfs    = 0.99        // Saturation value of the stick surface humidity
	kelvin = 273.2       // Celsius to Kelvin offset
	pi     = 3.141592654 // Pi, matched to the C++ value for exact output
	pr     = 0.7         // Prandtl number
	sbc    = 1.37e-12    // Stefan-Boltzmann constant (cal/cm2-s-K4)
	sc     = 0.58        // Schmidt number
	smv    = 94.743      // Saturated moisture content over water (percent)
	st     = 72.8        // Surface tension (dynes/cm)
	tcd    = 6.0         // Day-time clear-sky transmittance
	tcn    = 3.0         // Night-time clear-sky transmittance
	thdiff = 8.0         // Thermal diffusivity (cm2/h)
	wl     = 0.0023      // Diameter of a water molecule layer (cm)
	srf    = 14.82052    // Solar radiation flux constant
	wsf    = 4.60517     // Wind speed function constant
	hrd    = 0.116171    // Day-time relative humidity constant
	hrn    = 0.112467    // Night-time relative humidity constant
	sir    = 0.0714285   // Stick internal ratio
	scr    = 0.285714    // Stick cavity ratio
)
