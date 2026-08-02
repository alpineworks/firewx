package simple

import (
	"math"
	"testing"
)

// Validation sources for this package. The full references are in README.md.
//
//   - Chandler and Angstrom: the formulas are identical to the firebehavioR R
//     package (Ziegler et al. 2019; Sharples et al. 2009). A value here is
//     reproducible by that prior implementation. The R expression is given at
//     each test.
//   - Fosberg: original Fosberg (1978) piecewise equilibrium moisture content,
//     as recorded by the Fire Weather Indices Wiki (wikifire.wsl.ch). Note that
//     firebehavioR substitutes the Simard fuel moisture model, so its Fosberg
//     values differ.
//   - Hot-Dry-Windy: Srock et al. 2018 (Atmosphere 9(7):279), vapour pressure
//     deficit in hectopascals. firebehavioR uses kilopascals and reports a value
//     10 times smaller.
//   - KBDI: the corrected Keetch and Byram (1968, Res. Pap. SE-38) analytic
//     equation, with the constant 8.30 from Alexander (1990, Fire Management
//     Notes 51(4):23-25). firebehavioR uses a metric lookup-table variant, so it
//     is not a numeric oracle for this analytic form.
//   - Nesterov (1949; Groisman et al. 2007): the classical sum, temperature
//     times dew point depression, reset above 3 mm of precipitation.

// closeTo fails the test when got and want differ by more than tol.
func closeTo(t *testing.T, got, want, tol float64, what string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s: got %v, want %v (tol %v)", what, got, want, tol)
	}
}
