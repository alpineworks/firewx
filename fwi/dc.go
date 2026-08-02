package fwi

import (
	"math"
	"time"

	firewx "alpineworks.io/firewx"
)

// DC is the Drought Code. It tracks the moisture of deep, compact organic
// layers and reacts over about fifty-two days. The scale has no upper limit; a
// higher value means a deeper, longer drought.
type DC float64

// DroughtCode computes today's Drought Code from yesterday's code and today's
// noon weather: temperature (Celsius) and 24-hour rainfall (millimetres). The
// month and the latitude set the day-length factor. The Drought Code does not
// use relative humidity.
//
// Reference: Van Wagner and Pickett 1985, Forestry Technical Report 33. The
// equations match the cffdrs R package.
func DroughtCode(prev DC, t firewx.Celsius, rain firewx.Millimeters, month time.Month, lat float64) DC {
	temp := math.Max(float64(t), -2.8)
	ro := float64(rain)
	p := float64(prev)

	lf := dcDayLength(month, lat)
	pe := (0.36*(temp+2.8) + lf) / 2
	if pe < 0 {
		pe = 0
	}

	dr := p
	if ro > 2.8 {
		rw := 0.83*ro - 1.27
		smi := 800 * math.Exp(-p/400)
		dr = p - 400*math.Log(1+3.937*rw/smi)
		if dr < 0 {
			dr = 0
		}
	}

	d := dr + pe
	if d < 0 {
		d = 0
	}
	return DC(d)
}
