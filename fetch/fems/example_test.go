package fems_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"alpineworks.io/firewx/fetch/fems"
)

// ExampleClient_NFDR shows how to read the computed NFDRS output. This is the
// FEMS ground truth: a caller compares it against the nfdrs package. A real
// caller uses fems.New() and the default base URL; this example points the
// client at a test server so it can run.
func ExampleClient_NFDR() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "\"stationName\",\"observationTime\",\"fuelModelType\","+
			"\"energyReleaseComponent\",\"burningIndex\",\"stationId\"\n"+
			"\"GREENBASE\",\"2026-07-25T00:00:00Z\",\"Y\",27.91,23.61,20284\n")
	}))
	defer srv.Close()

	c := fems.New(fems.WithBaseURL(srv.URL), fems.WithHTTPClient(srv.Client()))

	out, err := c.NFDR(context.Background(), fems.Request{
		Stations: []string{"20284"},
		Start:    time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC),
		End:      time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	erc, _ := out[0].EnergyReleaseComponent.Get()
	fmt.Printf("%s ERC=%.1f\n", out[0].StationID, erc)
	// Output: 20284 ERC=27.9
}
