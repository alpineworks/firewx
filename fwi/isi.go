package fwi

import (
	"math"

	firewx "alpineworks.io/firewx"
)

// ISI is the Initial Spread Index. It combines the Fine Fuel Moisture Code and
// the wind speed into a relative measure of the expected rate of fire spread.
type ISI float64

// InitialSpreadIndex computes the Initial Spread Index from the Fine Fuel
// Moisture Code and the wind speed (kilometres per hour).
//
// Reference: Van Wagner and Pickett 1985, Forestry Technical Report 33. This is
// the standard form; the cffdrs package also has a variant for the Fire
// Behaviour Prediction System at high wind speeds, which this function does not
// use.
func InitialSpreadIndex(f FFMC, wind firewx.KilometersPerHour) ISI {
	fm := ffmcMoisture(f)
	w := float64(wind)
	fW := math.Exp(0.05039 * w)
	fF := 91.9 * math.Exp(-0.1386*fm) * (1 + math.Pow(fm, 5.31)/49300000)
	return ISI(0.208 * fW * fF)
}
