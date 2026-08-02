package fwi

import "time"

// The Duff Moisture Code and the Drought Code both use a day-length factor that
// changes with the month and the latitude. The tables and the latitude bands
// come from the cffdrs R package, after Lawson and Armitage (2008). Latitude is
// positive in the northern hemisphere and negative in the southern hemisphere.

// Effective day length (Le) for the Duff Moisture Code, by month, for four
// latitude bands.
var (
	dmcLe30N = [12]float64{6.5, 7.5, 9, 12.8, 13.9, 13.9, 12.4, 10.9, 9.4, 8, 7, 6}       // latitude above 30
	dmcLe10N = [12]float64{7.9, 8.4, 8.9, 9.5, 9.9, 10.2, 10.1, 9.7, 9.1, 8.6, 8.1, 7.8}  // 10 to 30
	dmcLe10S = [12]float64{10.1, 9.6, 9.1, 8.5, 8.1, 7.8, 7.9, 8.3, 8.9, 9.4, 9.9, 10.2}  // -10 to -30
	dmcLe30S = [12]float64{11.5, 10.5, 9.2, 7.9, 6.8, 6.2, 6.5, 7.4, 8.7, 10, 11.2, 11.8} // below -30
)

// Day-length factor (Lf) for the Drought Code, by month, for the two seasonal
// hemispheres.
var (
	dcLfNorth = [12]float64{-1.6, -1.6, -1.6, 0.9, 3.8, 5.8, 6.4, 5, 2.4, 0.4, -1.6, -1.6}
	dcLfSouth = [12]float64{6.4, 5, 2.4, 0.4, -1.6, -1.6, -1.6, -1.6, -1.6, 0.9, 3.8, 5.8}
)

// dmcDayLength returns the effective day length Le for the Duff Moisture Code.
func dmcDayLength(month time.Month, lat float64) float64 {
	m := int(month) - 1
	switch {
	case lat > 30:
		return dmcLe30N[m]
	case lat > 10:
		return dmcLe10N[m]
	case lat > -10:
		return 9
	case lat > -30:
		return dmcLe10S[m]
	default:
		return dmcLe30S[m]
	}
}

// dcDayLength returns the day-length factor Lf for the Drought Code.
func dcDayLength(month time.Month, lat float64) float64 {
	m := int(month) - 1
	switch {
	case lat > 20:
		return dcLfNorth[m]
	case lat > -20:
		return 1.4
	default:
		return dcLfSouth[m]
	}
}
