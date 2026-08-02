package fwi

import (
	"testing"
	"time"
)

func TestDMCDayLengthBands(t *testing.T) {
	// April values across the five latitude bands (cffdrs tables).
	cases := []struct {
		name string
		lat  float64
		want float64
	}{
		{"north of 30", 40, 12.8},
		{"10 to 30", 20, 9.5},
		{"tropics", 0, 9},
		{"-10 to -30", -20, 8.5},
		{"south of -30", -40, 7.9},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := dmcDayLength(time.April, tc.lat); got != tc.want {
				t.Errorf("dmcDayLength(April,%v)=%v, want %v", tc.lat, got, tc.want)
			}
		})
	}
}

func TestDCDayLengthBands(t *testing.T) {
	cases := []struct {
		name string
		lat  float64
		want float64
	}{
		{"north of 20", 40, 0.9},
		{"tropics", 0, 1.4},
		{"south of -20", -40, 0.4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := dcDayLength(time.April, tc.lat); got != tc.want {
				t.Errorf("dcDayLength(April,%v)=%v, want %v", tc.lat, got, tc.want)
			}
		})
	}
}
