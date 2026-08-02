package fems_test

import (
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// The fixtures in testdata/ are real responses from the FEMS climatology API for
// station GREENBASE (20284) on 2026-07-25. The last weather row has empty
// temperature and humidity cells, a real sensor dropout, to check that an empty
// cell becomes an absent Opt.

// fixtureServer serves the recorded response and records the last query it saw.
func fixtureServer(t *testing.T, path string, status int) (*httptest.Server, *string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var lastQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv, &lastQuery
}

func closeTo(t *testing.T, got, want, tol float64, what string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s: got %v, want %v (tol %v)", what, got, want, tol)
	}
}
