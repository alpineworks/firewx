// Package fwi implements the Canadian Forest Fire Weather Index System.
//
// The system has three fuel moisture codes with different time constants: the
// Fine Fuel Moisture Code (about two thirds of a day), the Duff Moisture Code
// (about twelve days), and the Drought Code (about fifty-two days). It has three
// derived indices: the Initial Spread Index, the Buildup Index, and the Fire
// Weather Index. The Fire Weather Index also gives the Daily Severity Rating.
//
// The three moisture codes are carryover codes. Each day's value depends on the
// previous day's value and one noon LST weather observation of temperature,
// relative humidity, wind speed, and 24-hour rainfall. A caller must give an
// uninterrupted daily stream, because one day is the model time step.
//
// The package has two API layers. Each code and index has a pure function for
// the mathematics. A stateful State driver carries the three codes forward from
// day to day and returns the full daily output.
//
// Reference: Van Wagner, C.E. and Pickett, T.L. 1985. Equations and FORTRAN
// program for the Canadian Forest Fire Weather Index System. Forestry Technical
// Report 33. The equations here match the cffdrs R package, which is based on
// the same report. See Van Wagner, C.E. 1987, Forestry Technical Report 35, for
// the structure of the system.
package fwi
