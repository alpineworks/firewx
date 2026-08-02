// Package synoptic is a client for the Synoptic Weather API.
//
// The Synoptic Weather API gives raw RAWS observations from a large mesonet.
// This client reads the time series endpoint and returns the observations as
// firewx.Station and firewx.Obs values, so a caller can drive the fire danger
// models directly.
//
// The API needs a token. Get one from Synoptic and give it with WithToken. The
// client requests metric units and UTC times, which match the SI convention of
// the firewx observation types.
//
// Reference: Synoptic Weather API documentation, https://docs.synopticdata.com.
package synoptic
