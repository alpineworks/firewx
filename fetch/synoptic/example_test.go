package synoptic_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"alpineworks.io/firewx/fetch/synoptic"
)

// ExampleClient_TimeSeries shows how to read station observations. A real caller
// uses synoptic.New(synoptic.WithToken("...")) and the default base URL. This
// example points the client at a test server so it can run.
func ExampleClient_TimeSeries() {
	// The test server stands in for the Synoptic Weather API.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"SUMMARY":{"RESPONSE_CODE":1},"STATION":[{"STID":"TT934",`+
			`"NAME":"TOLT RAWS","LATITUDE":47.677,"LONGITUDE":-121.642,"ELEVATION":400,`+
			`"OBSERVATIONS":{"date_time":["2024-08-01T19:00:00Z"],`+
			`"air_temp_set_1":[24.4],"relative_humidity_set_1":[38]}}]}`)
	}))
	defer srv.Close()

	c := synoptic.New(
		synoptic.WithToken("your-token"),
		synoptic.WithBaseURL(srv.URL),
		synoptic.WithHTTPClient(srv.Client()),
	)

	series, err := c.TimeSeries(context.Background(), synoptic.TimeSeriesRequest{
		Stations: []string{"TT934"},
		Start:    time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC),
		End:      time.Date(2024, 8, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	station := series[0]
	temp, _ := station.Observations[0].Temperature.Get()
	fmt.Printf("%s: %.1f C\n", station.Station.ID, temp)
	// Output: TT934: 24.4 C
}
