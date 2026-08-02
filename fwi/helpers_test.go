package fwi

import (
	"math"
	"testing"
)

// Validation source for this package: the equations are transcribed from the
// cffdrs R package (Van Wagner and Pickett 1985, Forestry Technical Report 33).
// The golden values are computed from those equations for the standard test
// dataset in testdata/test_fwi.csv, which is the Van Wagner and Pickett (1985)
// example set shipped with cffdrs.

// closeTo fails the test when got and want differ by more than tol.
func closeTo(t *testing.T, got, want, tol float64, what string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s: got %v, want %v (tol %v)", what, got, want, tol)
	}
}

// isNaNOrInf reports whether v is not a real, finite number.
func isNaNOrInf(v float64) bool {
	return math.IsNaN(v) || math.IsInf(v, 0)
}
