package fems_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	firewx "alpineworks.io/firewx"
	"alpineworks.io/firewx/fetch/fems"
)

func TestNFDR(t *testing.T) {
	srv, lastQuery := fixtureServer(t, "testdata/nfdr.csv", http.StatusOK)
	c := fems.New(fems.WithBaseURL(srv.URL), fems.WithHTTPClient(srv.Client()))

	out, err := c.NFDR(context.Background(), fems.Request{
		Stations: []string{"20284"},
		Start:    time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC),
		End:      time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"stationIds=20284", "dataFormat=csv", "dataset=all"} {
		if !strings.Contains(*lastQuery, want) {
			t.Errorf("query %q missing %q", *lastQuery, want)
		}
	}

	if len(out) != 3 {
		t.Fatalf("got %d rows, want 3", len(out))
	}
	first := out[0]

	if first.StationID != "20284" || first.StationName != "GREENBASE" || first.FuelModel != "Y" {
		t.Errorf("station/model: got %q %q %q", first.StationID, first.StationName, first.FuelModel)
	}
	if !first.Time.Equal(time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("time: got %v", first.Time)
	}

	cases := []struct {
		name string
		got  firewx.Opt[float64]
		want float64
	}{
		{"energy release component", first.EnergyReleaseComponent, 27.91},
		{"burning index", first.BurningIndex, 23.61},
		{"spread component", first.SpreadComponent, 3.15},
		{"ignition component", first.IgnitionComponent, 37.55},
		{"kbdi", first.KBDI, 166},
		{"gsi", first.GSI, 0.33},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, ok := tc.got.Get()
			if !ok {
				t.Fatalf("%s absent", tc.name)
			}
			closeTo(t, v, tc.want, 1e-9, tc.name)
		})
	}

	if v, ok := first.OneHourFuelMoisture.Get(); !ok || v != 6.06 {
		t.Errorf("1-hr fuel moisture: got %v %v, want 6.06", v, ok)
	}
}
