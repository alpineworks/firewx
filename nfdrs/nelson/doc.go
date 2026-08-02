// Package nelson implements the Nelson (2000) dead fuel moisture model.
//
// The model treats a dead fuel particle as a cylindrical stick with a set of
// radial nodes. It solves the coupled heat and moisture transfer between the
// stick and the air, so it needs the temperature, the relative humidity, the
// solar radiation, and the rainfall. It replaces the equilibrium moisture
// content method of the older NFDRS for the dead fuel time-lag classes (1-hour,
// 10-hour, 100-hour, and 1000-hour).
//
// The code is ported from the firelab/NFDRS4 C++ program
// (deadfuelmoisture.cpp), which follows Nelson (2000) with the modifications of
// Bevins (2005).
//
// References:
//   - Nelson, R.M. 2000. Prediction of diurnal change in 10-hr fuel stick
//     moisture content. Canadian Journal of Forest Research 30:1071-1087.
//   - Bevins, C.D. 2005. dead fuel moisture model modifications, as coded in
//     firelab/NFDRS4.
package nelson
