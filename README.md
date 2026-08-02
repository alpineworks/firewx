# firewx

Fire danger and fire weather calculations in Go.

Implements the Canadian Forest Fire Weather Index System, the United States
National Fire Danger Rating System, and the single-equation indices (Fosberg,
Chandler, Hot-Dry-Windy, Keetch-Byram and others).

There is no existing Go implementation of any of these. The reference
implementations are C++ (`firelab/NFDRS4`), R (`cffdrs/cffdrs_r`) and Python
(`cffdrs/cffdrs_py`, `nrcan-cfs-fire/cffdrs-ng`).

## Status

The root module is released at v1.0.0. It carries the shared types and the
`simple` indices. The `fwi`, `nfdrs`, and `fetch` modules are stubs.

| Module | Import path | Status |
| --- | --- | --- |
| root | `alpineworks.io/firewx` | Released v1.0.0: types, units, and `simple` |
| `simple` | `alpineworks.io/firewx/simple` | Implemented (a package in root) |
| `fwi` | `alpineworks.io/firewx/fwi` | Implemented |
| `nfdrs` | `alpineworks.io/firewx/nfdrs` | Stub |
| `fetch` | `alpineworks.io/firewx/fetch` | Synoptic and FEMS clients done; WRCC pending |

## Layout

Four modules, split on dependency footprint rather than topic. A consumer who
wants the Fine Fuel Moisture Code should not inherit an HTTP client and its
transitive dependencies.

```
firewx/
├── go.work
├── go.mod                      alpineworks.io/firewx
├── units.go obs.go opt.go wind.go
├── simple/                     package, not a module
├── fwi/     go.mod             alpineworks.io/firewx/fwi
├── nfdrs/   go.mod             alpineworks.io/firewx/nfdrs
│   ├── nelson/                 dead fuel moisture solver
│   ├── rothermel/              spread model and size-class weighting
│   ├── gsi/                    growing season index
│   └── fuelmodel/              parameter tables
└── fetch/   go.mod             alpineworks.io/firewx/fetch
    ├── fems/
    ├── synoptic/
    └── wrcc/
```

Modules are versioning units; packages are import units. `nelson`, `rothermel`
and `gsi` always change together and have no reason to version independently,
so they are packages inside `nfdrs` rather than modules of their own.

`fwi`, `nfdrs` and `fetch` all depend on the root module for shared types. The
mitigation for the coordinated-bump problem this creates is to design the
root's public surface once and then freeze it. If the root is churning
monthly, the types are wrong.

## Design decisions

**Named unit types everywhere.** The public API never takes a bare `float64`
for a measurement. The Canadian system is specified in Celsius, km/h and
millimetres; NFDRS and Rothermel are specified in Fahrenheit, mph, inches and
imperial fuel loadings. Observations are carried in SI and each model converts
at its own boundary.

**Missing data is `Opt[T]`, not NaN.** RAWS drop hours routinely and consumer
stations drop them more often. NaN propagates silently through the
finite-difference loop in the Nelson model and produces output that looks
plausible but is not.

**Local standard time is a fixed offset, never a `time.Location`.** Both
systems define their observation windows in LST, which does not shift with
daylight saving. Using zoneinfo introduces a one hour discontinuity in the
carryover codes twice a year that is nearly invisible in the output.

**Two API layers.** Pure functions for the mathematics, testable against golden
vectors with no hidden state, and a stateful driver on top that carries the
codes forward. Model state is a plain struct with exported fields, a schema
version, and a property-tested JSON round trip, because it is persisted between
daily runs and the Nelson stick's internal radial nodes must survive exactly or
the model drifts.

## Phasing

0. **Skeleton.** Workspace, four modules, CI, Release Please, frozen root types.
1. **`simple`.** Single-equation indices. Small enough that debugging the
   release pipeline here costs nothing, which is the point of doing it first.
2. **`fwi`.** Golden vectors ported from `cffdrs_r` test data. Evaluate
   `cffdrs-ng`'s hourly formulation before committing to the daily one.
3. **`fetch`.** FEMS and Synoptic clients. Comes before NFDRS because it is
   what makes NFDRS verifiable.
4. **`nfdrs`.** In order: `nelson`, `rothermel`, `gsi`, assembly. Each
   validated independently against published references before integration.

## Validation strategy

| Component | Validated against |
| --- | --- |
| `simple` | `ClimInd` R package outputs and published worked examples |
| `fwi` | `cffdrs_r` test vectors |
| `nelson` | RAWS stations carrying a physical 10-hr fuel stick sensor |
| `rothermel` | BehavePlus outputs |
| `nfdrs` end to end | FEMS-computed ERC for the same station and period |

Build `firelab/NFDRS4`, run it over a year of FW21 data, and freeze the output
as `testdata/` golden CSV before writing any Go.

Every package must have tests that use real data. The data can be historical.
Use a test case from a prior implementation of the algorithm when you can,
because it is a stronger reference than a value that you calculate yourself.

## Development

```sh
go work sync
go test ./...

# Verify each module in isolation, as a downstream consumer sees it.
for m in . fwi nfdrs fetch; do (cd "$m" && GOWORK=off go test ./...); done
```

## Documentation

You must write all documentation in ASD-STE100 Simplified Technical English.
This applies to the Markdown files, the Go doc comments, the changelog prose,
and the pull request text. `CLAUDE.md` gives the rules. The older prose in this
file is not compliant yet.

## Releases

Release Please, manifest mode, one release per module. Tags are `v1.2.3` for
the root module and `fwi/v1.2.3` for submodules, which is exactly the form the
Go module proxy expects for submodule versions.

Commit with conventional commits. Release Please routes changes to components
by file path, but scoping messages (`feat(nfdrs):`) makes the changelogs
readable.

## References

Every algorithm and every test vector cites its source. The primary references
are listed here.

Canadian FWI system, NFDRS, and their components:

- Van Wagner, C.E. 1987. *Development and Structure of the Canadian Forest Fire
  Weather Index System.* Forestry Technical Report 35.
- Nelson, R.M. 2000. Prediction of diurnal change in 10-hr fuel stick moisture
  content. *Canadian Journal of Forest Research* 30.
- Andrews, P.L. 2018. *The Rothermel Surface Fire Spread Model and Associated
  Developments: A Comprehensive Explanation.* RMRS-GTR-371.
- Jolly, W.M. et al. 2024. Modernizing the US National Fire Danger Rating
  System (version 4). *Environmental Modelling and Software.*

Single-equation indices (the `simple` package):

- Fosberg, M.A. 1978. Weather in wildland fire management: the fire weather
  index. *Proceedings of the Conference on Sierra Nevada Meteorology*, 1-4.
- Chandler, C., Cheney, P., Thomas, P., Trabaud, L., Williams, D. 1983. *Fire in
  Forestry, Volume 1: Forest Fire Behavior and Effects.* Wiley.
- Sharples, J.J., McRae, R.H.D., Weber, R.O., Gill, A.M. 2009. A simple index for
  assessing fire danger rating. *Environmental Modelling & Software* 24(6):
  764-774. (Chandler and Ångström forms.)
- Srock, A.F., Charney, J.J., Potter, B.E., Goodrick, S.L. 2018. The
  Hot-Dry-Windy Index: a new fire weather index. *Atmosphere* 9(7):279.
- Nesterov, V.G. 1949. *Combustibility of the forest and methods for its
  determination.* Goslesbumizdat, Moscow. Modern description in Groisman, P.Y.
  et al. 2007, *Global and Planetary Change* 56(3-4):371-386.
- Keetch, J.J., Byram, G.M. 1968. *A drought index for forest fire control.*
  Res. Pap. SE-38. USDA Forest Service, Southeastern Forest Experiment Station.
- Alexander, M.E. 1990. Computer calculation of the Keetch-Byram Drought
  Index — programmers beware! *Fire Management Notes* 51(4):23-25. (Corrects the
  8.30 constant.)

Prior implementations used as test references:

- Ziegler, J.P. et al. 2019. firebehavioR: an R package for fire behavior and
  danger analysis. *Fire* 2(3):41. (Chandler and Ångström match this package.)
- ClimInd R package. (Nesterov and KBDI cross-checks.)

## Licence

MIT. See [LICENSE](LICENSE).
