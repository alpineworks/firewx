// Package nfdrs computes the fire danger indices of the US National Fire Danger
// Rating System, version 2016.
//
// The system produces four indices for a fuel model and the daily fire weather:
//
//   - the spread component (SC), the forward rate of spread of the head fire;
//   - the energy release component (ERC), the energy release per unit area of
//     the flaming front;
//   - the burning index (BI), a combination of the spread component and the
//     energy release component;
//   - the ignition component (IC), the chance that a firebrand starts a fire.
//
// The subpackages compute the fuel moisture that the indices need. The nelson
// package computes the dead fuel moisture. The gsi package computes the live
// fuel moisture. This package holds the fuel models and the index equations.
//
// The FuelModel type holds the parameters of a fuel model. The package has the
// five standard NFDRS2016 fuel models: V, W, X, Y, and Z. The Compute method
// takes the fuel moisture, the drought, the wind, and the slope, then it returns
// the four indices.
//
// The Driver type is the stateful driver. It takes the hourly weather, runs the
// fuel moisture models, then it computes the indices. Use Compute for the pure
// index equations, or Driver for the full system from the raw weather.
//
// The equations are a port of the iCalcIndexes function of the firelab/NFDRS4
// C++ program, which follows the National Fire Danger Rating System of 2016.
//
// References:
//   - Bradshaw, L.S.; Deeming, J.E.; Burgan, R.E.; Cohen, J.D. 1984. The 1978
//     National Fire-Danger Rating System: technical documentation. General
//     Technical Report INT-169. Ogden, UT: USDA Forest Service.
//   - Jolly, W.M.; Freeborn, P.H.; and others. 2019. Simplified fuel models and
//     improved live and dead fuel moisture for the US National Fire Danger
//     Rating System. (The NFDRS2016 fuel models V, W, X, Y, and Z.)
//   - Andrews, P.L. 2018. The Rothermel surface fire spread model and associated
//     developments: a comprehensive explanation. General Technical Report
//     RMRS-GTR-371. Fort Collins, CO: USDA Forest Service.
package nfdrs
