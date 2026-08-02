# CLAUDE.md

Guidance for Claude Code working in this repository.

## What this is

A Go library implementing the Canadian Forest Fire Weather Index System, the
US National Fire Danger Rating System, and the single-equation fire danger
indices. There is no prior Go implementation of any of these; the references
are C++ (`firelab/NFDRS4`), R (`cffdrs/cffdrs_r`) and Python
(`cffdrs/cffdrs_py`, `nrcan-cfs-fire/cffdrs-ng`).

Read `README.md` first. It carries the module map, the phasing and the
validation strategy, and it is the source of truth for both.

## Repository shape

Four modules in a Go workspace. Modules are versioning units; packages are
import units. Do not add a module just to group related code — `nelson`,
`rothermel` and `gsi` always change together and are packages inside `nfdrs`.

Modules split on **dependency footprint**, not topic. `fetch` exists solely so
that a consumer who wants FFMC does not inherit an HTTP client.

## Non-negotiables

These are settled decisions. If a change requires breaking one, stop and raise
it rather than working around it.

**Named unit types.** The public API never accepts a bare `float64` for a
physical quantity. FWI is Celsius/km·h⁻¹/mm; NFDRS and Rothermel are
Fahrenheit/mph/inches/imperial loadings. Observations carry SI; each model
converts at its own boundary.

**`Opt[T]` for missing data, never NaN.** NaN propagates silently through
Nelson's finite-difference loop and yields plausible-looking garbage.

**Local standard time is a fixed offset, never a `time.Location`.** Both
systems define observation windows in LST. Zoneinfo introduces a one hour
discontinuity in the carryover codes twice a year.

**Two API layers.** Pure functions for the mathematics with no hidden state,
plus a stateful driver. Model state is a plain struct, exported fields, schema
version field, property-tested JSON round trip. State is persisted between
runs; Nelson's radial nodes must survive exactly.

**Functional options for every client.** A client constructor (for example the
`fetch` clients for FEMS, Synoptic and WRCC) uses the functional options
pattern. The `New` function takes a variadic list of `Option` values, and each
option sets one field. Do not add a constructor that takes a long parameter
list or a config struct. A caller must be able to construct a client with no
options and get sensible defaults.

**Semantic versioning from v1.0.0.** The modules are released at v1.0.0. A
change that breaks the public API of a module needs a new major version. Design
the public surface with care, because a major bump is expensive for consumers.

**All documentation is ASD-STE100 Simplified Technical English.** See the
Documentation language section. This applies to every word we write for a
reader: `README.md`, this file, package and function doc comments, changelog
prose, and pull request descriptions.

## Documentation language

You must write all documentation in ASD-STE100 Simplified Technical English
(STE). This is a strict requirement. It applies to Markdown files, Go doc
comments, and any other prose that a reader sees.

Obey these rules:

- Use only approved words from the STE dictionary. Use approved technical names
  and technical verbs for domain terms (for example "moisture code", "dew
  point", "interpolate").
- Write short sentences. Use a maximum of 20 words for an instruction and 25
  words for a description.
- Write one instruction in one sentence.
- Use the active voice.
- Use the articles "a", "an", and "the".
- Use simple verb tenses: the present, the past, and the future. Do not use the
  "-ing" form, unless the word is a technical name.
- Use one word for one meaning. Do not use synonyms. Do not use a noun as a
  verb.
- Start a paragraph with its main point. Keep a paragraph to one topic.
- Use a vertical list for a set of conditions or steps.
- Do not use slang, jargon, idioms, or contractions.
- Do not omit words to make a sentence shorter.
- Write positive statements. Give a warning before you give the action.

Note (2026-08-01): the older prose in this file, in `README.md`, and in the
root package doc comments is not STE yet. It is a retrofit that is not done. Do
not copy its style. New documentation must follow the rules above.

## Bootstrap constraint

The root module is now tagged at v1.0.0. Before this tag, the submodules had
**no `require` on the root module**, because the isolated CI job could not
resolve an unpublished version.

A submodule that imports the root now adds `require alpineworks.io/firewx
v1.0.0`. Add the require only to a module that imports the root.

## Working conventions

- Conventional commits, scoped to the component: `feat(nfdrs):`, `fix(fwi):`.
  Release Please routes by file path but the scope makes changelogs readable.
- `gofmt` clean. CI fails otherwise.
- One test file per source file. Put the tests for `units.go` in `units_test.go`
  and the tests for `obs.go` in `obs_test.go`. Put shared test helpers, such as
  `closeTo`, in `helpers_test.go`.
- Write table-driven tests. Use a slice of named cases and a loop with `t.Run`
  for each case. Use a different form only when you cannot write the test as a
  table (for example a single sequence of stateful steps).
- Verify both ways before claiming a change works:
  ```sh
  go test ./...                                        # workspace
  for m in . fwi nfdrs fetch; do (cd "$m" && GOWORK=off go test ./...); done
  ```
  The workspace masks missing `require` directives. The isolated run is the one
  that reflects what a downstream consumer gets from the proxy.
- Never edit `.release-please-manifest.json` by hand.

## Validation

Do not write model code before the golden fixtures exist. Each component has a
published reference; use it.

| Component | Reference |
| --- | --- |
| `simple` | ClimInd R package outputs and published worked examples |
| `fwi` | `cffdrs_r` test vectors |
| `nelson` | RAWS stations with a physical 10-hr fuel stick sensor |
| `rothermel` | BehavePlus outputs |
| `nfdrs` end to end | FEMS-computed ERC, same station and period |

Every package must have tests that use real data. The data can be historical.
The tests must be rigorous, because they show that the code meets the published
standard.

Use a test case from a prior implementation of the algorithm when you can. A
prior implementation is a stronger reference than a value that you calculate
yourself. The prior implementations are:

- the `cffdrs` R and Python packages, for the Canadian system;
- the `firelab/NFDRS4` C++ program, for NFDRS;
- the `ClimInd` R package and published worked examples, for the single-equation
  indices.

Write the source of each test vector in a comment. Put a large reference set in
`testdata/`.

For NFDRS: build `firelab/NFDRS4`, run it over a year of FW21 data for a single
station, freeze the output as `testdata/` golden CSV, then write Go against it.

Property tests worth having alongside the golden vectors: moisture codes stay
in range, FFMC rises as RH falls, state serialisation round-trips exactly, and
absent inputs produce absent outputs rather than zeros.

## References and citations

You must cite every reference that you use. This applies to the algorithm and to
the test data. This is a strict requirement.

- For an algorithm, put the primary reference in the doc comment of the function
  or the type. Give the author, the year, and the title.
- For a test vector, put the source in a comment at the test. Name the prior
  implementation or the published example. Give enough detail to find it again.
- List the full references in `README.md`.

Do not use an equation or a test value without a citation.

## Current phase

Phase 0 complete: workspace, four modules, CI, Release Please, root types.

Phase 1 is `simple` — Fosberg, Chandler, Angström, Nesterov, Hot-Dry-Windy,
KBDI. The point of doing the trivial indices first is to prove the release
pipeline while a broken tag costs nothing.

Open question deferred from Phase 0: whether `simple` should be its own module.
It is pure stdlib and always changes with the root types, which argues for
keeping it in root — but it also means the root module churns during Phase 1,
and the root is the module that most needs to freeze early.

## Reference data

- Nearest comparable RAWS is Tolt (47.677 N, 121.642 W, 400 ft), about 27 miles
  east. Same side of the Cascade crest, same maritime regime, Douglas-fir and
  western hemlock.
- Station NFDRS descriptors come from FEMS (Site Metadata subject area) or from
  `NFDRS_National_Settings_for_Fire_Family_Plus_Catalogs.xlsx` on wildfire.gov.
- FEMS exposes computed fire danger over HTTP, so official ERC for a station is
  available without implementing NFDRS. Use it as ground truth.
