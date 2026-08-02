// Package fetch groups the clients for the public fire weather data sources.
//
// The clients live in subpackages, one per source:
//
//   - synoptic: raw RAWS observations from the Synoptic Weather API.
//   - fems: RAWS observations, station metadata, and computed NFDRS output from
//     the Fire Environment Mapping System. Not implemented yet.
//   - wrcc: the RAWS station inventory from the Western Regional Climate Center.
//     Not implemented yet.
//
// The clients live in a separate module, so a consumer who only wants the fire
// danger calculations does not inherit an HTTP client and its dependencies.
//
// Every client constructor uses the functional options pattern. The New
// function takes a variadic list of Option values. A caller can construct a
// client with no options and get sensible defaults.
package fetch
