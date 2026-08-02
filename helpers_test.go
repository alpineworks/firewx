package firewx

import (
	"math"
	"testing"
)

// closeTo fails the test when got and want differ by more than tol.
func closeTo(t *testing.T, got, want, tol float64, what string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s: got %v, want %v (tol %v)", what, got, want, tol)
	}
}
