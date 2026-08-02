// Package fetch provides clients for the public fire weather data sources.
//
// Subpackages cover FEMS (the authoritative source for RAWS observations,
// station metadata and computed NFDRS output), Synoptic Data (raw RAWS
// observations, useful for gap filling) and WRCC (the station inventory).
//
// These live in a separate module so that consumers who only want the
// calculations do not inherit an HTTP client and its dependency tree.
//
// Status: not yet implemented. See the repository README for phasing.
package fetch
