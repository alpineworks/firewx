package nfdrs

// FuelModel holds the parameters of an NFDRS fuel model. The loads are in tons
// per acre. The surface-area-to-volume ratios are in reciprocal feet. The heat
// content is in British thermal units per pound. The extinction moisture is a
// percentage. The depth is in feet.
type FuelModel struct {
	// Code is the letter of the fuel model, for example 'Y'.
	Code rune
	// Name describes the fuel model.
	Name string

	// Dead fuel loads by time-lag class (tons/acre).
	Load1, Load10, Load100, Load1000 float64
	// Live fuel loads (tons/acre).
	LoadHerb, LoadWoody float64
	// LoadDrought is the extra dead fuel load at the maximum drought
	// (tons/acre). The drought index adds it to the four dead classes.
	LoadDrought float64

	// Surface-area-to-volume ratios (1/ft).
	SAV1, SAV10, SAV100, SAV1000 float64
	SAVHerb, SAVWoody            float64

	// HeatContent is the low heat of combustion (BTU/lb).
	HeatContent float64
	// ExtinctionMoisture is the dead fuel moisture of extinction (percent).
	ExtinctionMoisture float64
	// Depth is the fuel bed depth (ft).
	Depth float64
	// WindReductionFactor scales the 20-foot wind to the midflame wind.
	WindReductionFactor float64
	// MaxSpreadComponent is the spread component above which every ignition
	// becomes a reportable fire. The ignition component uses it.
	MaxSpreadComponent float64
}

// The five standard NFDRS2016 fuel models. The parameters are from the National
// Wildfire Coordinating Group fuel model parameter table
// (https://www.wildfire.gov/page/nfdrs-fuel-model-parameters), which follows
// Jolly and others (2019). Every model shares the same surface-area-to-volume
// ratios and the same heat content. The drought fuel load adds to all four dead
// fuel load classes.
var (
	// FuelModelV is grass (based on the fire behavior model GR2).
	FuelModelV = FuelModel{
		Code: 'V', Name: "Grass",
		Load1: 0.1, Load10: 0, Load100: 0, Load1000: 0,
		LoadHerb: 1, LoadWoody: 0, LoadDrought: 0,
		SAV1: 2000, SAV10: 109, SAV100: 30, SAV1000: 8, SAVHerb: 2000, SAVWoody: 1500,
		HeatContent: 8000, ExtinctionMoisture: 15, Depth: 1, WindReductionFactor: 0.6, MaxSpreadComponent: 108,
	}
	// FuelModelW is grass-shrub (based on the fire behavior model GS2).
	FuelModelW = FuelModel{
		Code: 'W', Name: "Grass-Shrub",
		Load1: 0.5, Load10: 0.5, Load100: 0, Load1000: 0,
		LoadHerb: 0.6, LoadWoody: 1, LoadDrought: 1,
		SAV1: 2000, SAV10: 109, SAV100: 30, SAV1000: 8, SAVHerb: 2000, SAVWoody: 1500,
		HeatContent: 8000, ExtinctionMoisture: 15, Depth: 1.5, WindReductionFactor: 0.4, MaxSpreadComponent: 62,
	}
	// FuelModelX is brush (based on the fire behavior model SH9).
	FuelModelX = FuelModel{
		Code: 'X', Name: "Brush",
		Load1: 4.5, Load10: 2.45, Load100: 0, Load1000: 0,
		LoadHerb: 1.55, LoadWoody: 7, LoadDrought: 2.5,
		SAV1: 2000, SAV10: 109, SAV100: 30, SAV1000: 8, SAVHerb: 2000, SAVWoody: 1500,
		HeatContent: 8000, ExtinctionMoisture: 25, Depth: 4.4, WindReductionFactor: 0.4, MaxSpreadComponent: 104,
	}
	// FuelModelY is timber (based on the fire behavior model TL1).
	FuelModelY = FuelModel{
		Code: 'Y', Name: "Timber",
		Load1: 2.5, Load10: 2.2, Load100: 3.6, Load1000: 10.16,
		LoadHerb: 0, LoadWoody: 0, LoadDrought: 5,
		SAV1: 2000, SAV10: 109, SAV100: 30, SAV1000: 8, SAVHerb: 2000, SAVWoody: 1500,
		HeatContent: 8000, ExtinctionMoisture: 25, Depth: 0.6, WindReductionFactor: 0.2, MaxSpreadComponent: 5,
	}
	// FuelModelZ is slash (based on the fire behavior model SB2).
	FuelModelZ = FuelModel{
		Code: 'Z', Name: "Slash",
		Load1: 4.5, Load10: 4.25, Load100: 4, Load1000: 4,
		LoadHerb: 0, LoadWoody: 0, LoadDrought: 7,
		SAV1: 2000, SAV10: 109, SAV100: 30, SAV1000: 8, SAVHerb: 2000, SAVWoody: 1500,
		HeatContent: 8000, ExtinctionMoisture: 25, Depth: 1, WindReductionFactor: 0.4, MaxSpreadComponent: 19,
	}
)
