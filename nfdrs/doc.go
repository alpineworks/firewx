// Package nfdrs implements the United States National Fire Danger Rating
// System.
//
// Unlike the Canadian system, NFDRS outputs are in physical units -- the
// Energy Release Component is in BTU per square foot and the Spread Component
// is a rate of spread -- which means the implementation is a stack of models
// rather than a set of fitted curves: the Nelson dead fuel moisture solver,
// the Rothermel surface spread model with Albini size-class weighting, the
// Growing Season Index for live fuel moisture, and the Keetch-Byram Drought
// Index for drought fuel loading.
//
// Status: not yet implemented. See the repository README for phasing.
package nfdrs
