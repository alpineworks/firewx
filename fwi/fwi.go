package fwi

import "math"

// FWI is the Fire Weather Index. It combines the Initial Spread Index and the
// Buildup Index into a relative measure of the fire intensity per unit length
// of the fire front. It is the index that sets the daily public fire danger.
type FWI float64

// FireWeatherIndex computes the Fire Weather Index from the Initial Spread Index
// and the Buildup Index.
//
// Reference: Van Wagner and Pickett 1985, Forestry Technical Report 33. The
// equations match the cffdrs R package.
func FireWeatherIndex(i ISI, b BUI) FWI {
	isi := float64(i)
	bui := float64(b)

	var bb float64
	if bui > 80 {
		bb = 0.1 * isi * (1000 / (25 + 108.64/math.Exp(0.023*bui)))
	} else {
		bb = 0.1 * isi * (0.626*math.Pow(bui, 0.809) + 2)
	}

	if bb <= 1 {
		return FWI(bb)
	}
	return FWI(math.Exp(2.72 * math.Pow(0.434*math.Log(bb), 0.647)))
}
