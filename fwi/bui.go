package fwi

import "math"

// BUI is the Buildup Index. It combines the Duff Moisture Code and the Drought
// Code into a relative measure of the total fuel available to the fire.
type BUI float64

// BuildupIndex computes the Buildup Index from the Duff Moisture Code and the
// Drought Code.
//
// Reference: Van Wagner and Pickett 1985, Forestry Technical Report 33. The
// equations match the cffdrs R package.
func BuildupIndex(d DMC, c DC) BUI {
	dmc := float64(d)
	dc := float64(c)

	var bui float64
	if dmc == 0 && dc == 0 {
		bui = 0
	} else {
		bui = 0.8 * dc * dmc / (dmc + 0.4*dc)
	}

	var p float64
	if dmc != 0 {
		p = (dmc - bui) / dmc
	}
	cc := 0.92 + math.Pow(0.0114*dmc, 1.7)
	bui0 := dmc - cc*p
	if bui0 < 0 {
		bui0 = 0
	}
	if bui < dmc {
		bui = bui0
	}
	return BUI(bui)
}
