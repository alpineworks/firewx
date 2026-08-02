package fwi

import (
	"math"
	"time"

	firewx "alpineworks.io/firewx"
)

// DMC is the Duff Moisture Code. It tracks the moisture of loosely compacted
// organic layers of moderate depth and reacts over about twelve days. The scale
// has no upper limit; a higher value means a drier duff layer.
type DMC float64

// DuffMoistureCode computes today's Duff Moisture Code from yesterday's code and
// today's noon weather: temperature (Celsius), relative humidity (percent), and
// 24-hour rainfall (millimetres). The month and the latitude set the day-length
// factor.
//
// Reference: Van Wagner and Pickett 1985, Forestry Technical Report 33. The
// equations match the cffdrs R package.
func DuffMoistureCode(prev DMC, t firewx.Celsius, rh firewx.Percent, rain firewx.Millimeters, month time.Month, lat float64) DMC {
	temp := math.Max(float64(t), -1.1)
	h := float64(rh)
	ro := float64(rain)
	p := float64(prev)

	le := dmcDayLength(month, lat)
	rk := 1.894 * (temp + 1.1) * (100 - h) * le * 1e-4

	pr := p
	if ro > 1.5 {
		rw := 0.92*ro - 1.27
		wmi := 20 + 280/math.Exp(0.023*p)
		var b float64
		switch {
		case p <= 33:
			b = 100 / (0.5 + 0.3*p)
		case p <= 65:
			b = 14 - 1.3*math.Log(p)
		default:
			b = 6.2*math.Log(p) - 17.2
		}
		wmr := wmi + 1000*rw/(48.77+b*rw)
		pr = 43.43 * (5.6348 - math.Log(wmr-20))
	}
	if pr < 0 {
		pr = 0
	}

	d := pr + rk
	if d < 0 {
		d = 0
	}
	return DMC(d)
}
