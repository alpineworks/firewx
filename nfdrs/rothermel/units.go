package rothermel

// The Rothermel model uses imperial units throughout: fuel load in pounds per
// square foot, the surface-area-to-volume ratio in reciprocal feet, heat content
// in British thermal units per pound, and rate of spread in feet per minute.
// Each quantity has a named type, so the caller cannot pass a value in the wrong
// unit.

// FuelLoad is an oven-dry fuel load per unit ground area (lb/ft2).
type FuelLoad float64

// TonsPerAcre converts a fuel load to tons per acre. Fuel models are often
// tabulated in tons per acre.
func (l FuelLoad) TonsPerAcre() float64 { return float64(l) / tonsPerAcreToLoad }

// FuelLoadFromTonsPerAcre converts a load in tons per acre to pounds per square
// foot.
func FuelLoadFromTonsPerAcre(t float64) FuelLoad { return FuelLoad(t * tonsPerAcreToLoad) }

// SAV is a surface-area-to-volume ratio (ft2/ft3, that is 1/ft). A higher ratio
// means a finer fuel.
type SAV float64

// HeatContent is the low heat of combustion of a fuel (BTU/lb).
type HeatContent float64

// FeetPerMinute is a rate of spread.
type FeetPerMinute float64

// ChainsPerHour converts a rate of spread to chains per hour. One chain is 66
// feet, so one chain per hour is 1.1 feet per minute. Fire behavior reports
// often use chains per hour.
func (r FeetPerMinute) ChainsPerHour() float64 { return float64(r) / feetPerChain * 60.0 }

// Model constants, with their standard values from Rothermel (1972) and
// Andrews (2018).
const (
	// tonsPerAcreToLoad converts tons per acre to pounds per square foot. One
	// ton is 2000 lb and one acre is 43560 ft2.
	tonsPerAcreToLoad = 2000.0 / 43560.0

	// feetPerChain is the number of feet in a chain.
	feetPerChain = 66.0

	// feetPerMinutePerMph converts miles per hour to feet per minute. One mile
	// is 5280 ft and one hour is 60 min.
	feetPerMinutePerMph = 5280.0 / 60.0

	// StandardHeatContent is the heat content that most fuel models use.
	StandardHeatContent HeatContent = 8000.0

	// standardParticleDensity is the oven-dry density of a fuel particle
	// (lb/ft3).
	standardParticleDensity = 32.0

	// standardTotalMineral is the total mineral content of a fuel particle
	// (fraction).
	standardTotalMineral = 0.0555

	// standardEffectiveMineral is the effective mineral content of a fuel
	// particle (fraction), that is the mineral content without the silica.
	standardEffectiveMineral = 0.010
)
