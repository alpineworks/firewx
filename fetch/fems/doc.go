// Package fems is a client for the Fire Environment Mapping System (FEMS).
//
// FEMS is the United States Forest Service system that archives RAWS weather and
// runs the National Fire Danger Rating System. It is the authoritative source of
// both the observations and the computed fire danger output, so it is the ground
// truth for a check of the nfdrs package.
//
// This client reads two endpoints:
//
//   - Weather returns the RAWS weather observations as firewx.Station and
//     firewx.Obs values. FEMS reports weather in Fahrenheit, inches, and miles
//     per hour; the client converts to SI at the boundary.
//   - NFDR returns the computed NFDRS output for each hour: the dead and live
//     fuel moistures, the Keetch-Byram Drought Index, and the ignition, energy
//     release, spread, and burning indices.
//
// A public request returns the most recent two weeks. The full archive needs an
// authenticated account, which this client does not support yet.
//
// Reference: FEMS, https://fems.fs2c.usda.gov.
package fems
