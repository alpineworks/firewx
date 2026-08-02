// Package gsi implements the Growing Season Index live fuel moisture model.
//
// The model predicts the moisture content of live herbaceous fuel and live
// woody fuel from the daily weather. It computes a Growing Season Index (GSI)
// for each day from three indicator functions: the daily minimum temperature,
// the vapor pressure deficit, and the day length. The GSI is the product of the
// three indicators, so it has a range of 0 to 1. The model keeps a running
// average of the GSI, then it maps the running average to a live fuel moisture
// content with a linear ramp.
//
// The live herbaceous fuel has an annual curing state. The herbaceous moisture
// rises after the green-up, then it does not rise again after the fuel cures.
//
// The model is stateful. The Model type holds the running average window and the
// curing state. It has exported fields, a schema version, and an exact JSON
// round trip, so a caller can persist the model between runs.
//
// The code is a port of the LiveFuelMoisture class in the firelab/NFDRS4 C++
// program, which follows Jolly (2005). The day length function is from MT-CLIM
// (Thornton and Running 1999).
//
// References:
//   - Jolly, W.M.; Nemani, R.; Running, S.W. 2005. A generalized, bioclimatic
//     index to predict foliar phenology in response to climate. Global Change
//     Biology 11(4):619-632.
//   - Thornton, P.E.; Running, S.W. 1999. An improved algorithm for estimating
//     incident daily solar radiation from measurements of temperature, humidity,
//     and precipitation. Agricultural and Forest Meteorology 93:211-228.
//     (The day length function of MT-CLIM.)
package gsi
