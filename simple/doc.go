// Package simple gives the single-equation fire danger indices. These are the
// Fosberg, Chandler, Angström, Nesterov, Hot-Dry-Windy, and Keetch-Byram
// Drought indices.
//
// These indices are quick to calculate. You can calculate them for each
// observation. They are the real-time tier of a dashboard. The Canadian and the
// NFDRS outputs are the daily and the seasonal tiers.
//
// Each index has two forms. A pure function takes the measurement in the units
// of its published equation. It returns a named result type: Fosberg, Chandler,
// Angstrom, HDW, Nesterov, or KBDI. A FromObs helper takes a firewx.Obs. It
// changes the units from SI at the boundary. It returns a firewx.Opt. An absent
// input gives an absent result and not a wrong zero.
//
// The two cumulative indices are Nesterov and KBDI. Each one also has a stateful
// driver: NesterovState and KBDIState. The driver holds the running value across
// days. The driver state is a plain struct. Its fields are exported and have
// JSON tags. It has a schema version. It goes to and from JSON without a change.
package simple
