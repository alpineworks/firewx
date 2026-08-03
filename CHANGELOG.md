# Changelog

## [1.2.1](https://github.com/alpineworks/firewx/compare/v1.2.0...v1.2.1) (2026-08-03)


### Dependencies

* bump alpineworks.io/firewx to v1.2.0 in the submodules ([f522f3e](https://github.com/alpineworks/firewx/commit/f522f3e220f50efa6327b5cb4b99897409c2620b))

## [1.2.0](https://github.com/alpineworks/firewx/compare/v1.1.0...v1.2.0) (2026-08-03)


### Features

* **fetch:** add FEMS client (weather + NFDRS output) ([d66db94](https://github.com/alpineworks/firewx/commit/d66db941a830ec4d91a143fcacc29bb1091467d7))
* **fetch:** add Synoptic Weather API client with functional options ([2f7893d](https://github.com/alpineworks/firewx/commit/2f7893ddbb0dcb8d643648d4e0d9c627cd265bf3))
* **fetch:** Synoptic and FEMS clients (functional options) ([a973f55](https://github.com/alpineworks/firewx/commit/a973f5586ee763dde49ea98451b64910357490ec))
* **nfdrs:** add Driver state persistence ([55c388c](https://github.com/alpineworks/firewx/commit/55c388cf5996e25006937f8d370329f47930f6d8))
* **nfdrs:** add Driver state persistence ([4090360](https://github.com/alpineworks/firewx/commit/4090360cc45ba65c78e90dbf897ae86f20efb472))
* **nfdrs:** add GSI live fuel moisture model ([516610e](https://github.com/alpineworks/firewx/commit/516610ec88aa5be3e1d19f7cde61ac8f86fb100c))
* **nfdrs:** add Nelson dead fuel moisture solver ([909e431](https://github.com/alpineworks/firewx/commit/909e431f2f4971fcfe46aed5ed913420305309d8))
* **nfdrs:** add Rothermel surface fire spread model ([2dd0936](https://github.com/alpineworks/firewx/commit/2dd0936dc1a5e2b27e70da92eb5515e718db1f4e))
* **nfdrs:** add the assembly driver ([6f58120](https://github.com/alpineworks/firewx/commit/6f58120421a57ef4f2d2c86728266c48cf4a5374))
* **nfdrs:** add the fire danger index equations ([cb03b62](https://github.com/alpineworks/firewx/commit/cb03b6206e6434f1c2b1af609d35e472e0789c75))
* **nfdrs:** nelson foundation — constants, derived params, sticks ([7396ade](https://github.com/alpineworks/firewx/commit/7396ade6b72fb93160a9c607f1a0919438fcc604))
* **nfdrs:** Nelson, Rothermel, and GSI models ([cc1212a](https://github.com/alpineworks/firewx/commit/cc1212a7dc9fcbd46d39e81546a0d89f6d01a927))
* **nfdrs:** NFDRS assembly — index equations and driver ([8547431](https://github.com/alpineworks/firewx/commit/85474312a452cec188b5305a9d2169f676c4c432))

## [1.1.0](https://github.com/alpineworks/firewx/compare/v1.0.1...v1.1.0) (2026-08-02)


### Features

* **fwi:** implement the Canadian Forest Fire Weather Index System ([6d9eb73](https://github.com/alpineworks/firewx/commit/6d9eb7389b4c7853bf9636931f49386fafda5dad))
* **fwi:** implement the Canadian Forest Fire Weather Index System ([d3bc5df](https://github.com/alpineworks/firewx/commit/d3bc5df9f528bb2652f90984b29ca3e8bb575ec9))


### Documentation

* add runnable examples for firewx, simple, and fwi ([f73eb42](https://github.com/alpineworks/firewx/commit/f73eb42cb633e450847f8e65d71b5960b48f8232))

## [1.0.1](https://github.com/alpineworks/firewx/compare/v1.0.0...v1.0.1) (2026-08-02)


### Documentation

* reflect the v1.0.0 release ([4664ef8](https://github.com/alpineworks/firewx/commit/4664ef8764403dcef61395affb13ad38556cf5f8))

## 1.0.0 (2026-08-02)


### Features

* **simple:** implement single-equation fire danger indices ([c599eae](https://github.com/alpineworks/firewx/commit/c599eae94c78c5a9e477aa6673874c7d996e59f8))


### Bug Fixes

* Obs.DewPoint returns absent when the dew point is undefined ([9820fa1](https://github.com/alpineworks/firewx/commit/9820fa119709e1dbabc6100141cd17af250eaa61))
* set module path to github.com/alpineworks/firewx ([3cc3892](https://github.com/alpineworks/firewx/commit/3cc3892fd585fc16f21810ec6b6f45c1b0cff95e))


### Documentation

* cite algorithm and test-data references throughout ([3a32845](https://github.com/alpineworks/firewx/commit/3a32845deab2e73c00951f79484301c4395edf83))
* document root unit conversions and Opt JSON methods ([6a068d3](https://github.com/alpineworks/firewx/commit/6a068d39be706d15d74a43e449a469ddf1693b96))
* record test-file-per-source and functional-options conventions ([6a8f558](https://github.com/alpineworks/firewx/commit/6a8f55853d41db3057307983e716063bbb51016a))
* require ASD-STE100 Simplified Technical English for all docs ([731ee2a](https://github.com/alpineworks/firewx/commit/731ee2a1bac6bdad296b11f862490afd7ea5fadf))
* require rigorous real-data tests, prefer prior-implementation vectors ([084bb0a](https://github.com/alpineworks/firewx/commit/084bb0a61f0923e7d83b33a43af4a19a15876366))
* state MIT licence in README ([b00dcd3](https://github.com/alpineworks/firewx/commit/b00dcd3a317f042d5072f82d76c15ff1afe91970))
