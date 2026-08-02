package fwi

import "math"

// DSR is the Daily Severity Rating. It is a transform of the Fire Weather Index
// that scales more evenly with the difficulty of fire control, so a caller can
// average it over time.
type DSR float64

// DailySeverityRating computes the Daily Severity Rating from the Fire Weather
// Index.
//
// Reference: Van Wagner 1987, Forestry Technical Report 35. The equation matches
// the cffdrs R package.
func DailySeverityRating(f FWI) DSR {
	return DSR(0.0272 * math.Pow(float64(f), 1.77))
}
