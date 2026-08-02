// Package simple implements the single-equation fire danger indices: Fosberg,
// Chandler, Angstrom, Nesterov, Hot-Dry-Windy, and the Keetch-Byram Drought
// Index.
//
// These are cheap enough to compute on every observation rather than once a
// day, which makes them the real-time tier of a dashboard while the Canadian
// and NFDRS outputs occupy the daily and seasonal tiers.
//
// Status: not yet implemented. See the repository README for phasing.
package simple
