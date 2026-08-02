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

**Everything stays v0.x until NFDRS is complete.**

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

Submodules currently have **no `require` on the root module**, deliberately. A
submodule cannot build outside the workspace until the root has a published
tag, so the isolated CI job fails on an unresolvable version if the require is
added early.

Add the requires only after root `v0.1.0` is tagged, and only to modules that
genuinely import the root.

## Working conventions

- Conventional commits, scoped to the component: `feat(nfdrs):`, `fix(fwi):`.
  Release Please routes by file path but the scope makes changelogs readable.
- `gofmt` clean. CI fails otherwise.
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
| `fwi` | `cffdrs_r` test vectors |
| `nelson` | RAWS stations with a physical 10-hr fuel stick sensor |
| `rothermel` | BehavePlus outputs |
| `nfdrs` end to end | FEMS-computed ERC, same station and period |

For NFDRS: build `firelab/NFDRS4`, run it over a year of FW21 data for a single
station, freeze the output as `testdata/` golden CSV, then write Go against it.

Property tests worth having alongside the golden vectors: moisture codes stay
in range, FFMC rises as RH falls, state serialisation round-trips exactly, and
absent inputs produce absent outputs rather than zeros.

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
