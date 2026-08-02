package firewx

import (
	"testing"
	"time"
)

func TestLocalStandardTimeIgnoresDST(t *testing.T) {
	s := Station{LSTOffset: -8 * time.Hour} // Pacific standard, year round
	cases := []struct {
		name     string
		utc      time.Time
		wantHour int
	}{
		{"summer", time.Date(2026, 7, 15, 20, 0, 0, 0, time.UTC), 12},
		{"winter", time.Date(2026, 1, 15, 20, 0, 0, 0, time.UTC), 12},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if h := s.LocalStandardTime(tc.utc).Hour(); h != tc.wantHour {
				t.Errorf("%s: got hour %d, want %d", tc.name, h, tc.wantHour)
			}
		})
	}
}

func TestObsDewPoint(t *testing.T) {
	// DewPoint is absent without humidity, and absent at zero humidity where the
	// bare function returns NaN. Obs.DewPoint must never hold a present NaN.
	cases := []struct {
		name      string
		obs       Obs
		wantValid bool
	}{
		{"absent without RH", Obs{Temperature: Some(Celsius(20))}, false},
		{"absent at zero RH", Obs{Temperature: Some(Celsius(20)), RelativeHumidity: Some(Percent(0))}, false},
		{"present with T and RH", Obs{Temperature: Some(Celsius(20)), RelativeHumidity: Some(Percent(50))}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.obs.DewPoint(); got.Valid() != tc.wantValid {
				t.Errorf("%s: Valid()=%v, want %v", tc.name, got.Valid(), tc.wantValid)
			}
		})
	}
}

func TestObsVaporPressureDeficit(t *testing.T) {
	cases := []struct {
		name      string
		obs       Obs
		wantValid bool
	}{
		{"absent without RH", Obs{Temperature: Some(Celsius(20))}, false},
		{"present with T and RH", Obs{Temperature: Some(Celsius(20)), RelativeHumidity: Some(Percent(50))}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.obs.VaporPressureDeficit(); got.Valid() != tc.wantValid {
				t.Errorf("%s: Valid()=%v, want %v", tc.name, got.Valid(), tc.wantValid)
			}
		})
	}
}
