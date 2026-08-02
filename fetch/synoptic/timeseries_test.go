package synoptic_test

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"alpineworks.io/firewx/fetch/synoptic"
)

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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv, &lastQuery
}

func TestTimeSeries(t *testing.T) {
	// The fixture in testdata/ is one RAWS station, modelled on the Synoptic
	// Weather API time series response (metric units, UTC times).
	srv, lastQuery := fixtureServer(t, "testdata/timeseries.json", http.StatusOK)

	c := synoptic.New(
		synoptic.WithToken("test-token"),
		synoptic.WithBaseURL(srv.URL),
		synoptic.WithHTTPClient(srv.Client()),
	)

	series, err := c.TimeSeries(context.Background(), synoptic.TimeSeriesRequest{
		Stations: []string{"TT934"},
		Start:    time.Date(2024, 8, 1, 19, 0, 0, 0, time.UTC),
		End:      time.Date(2024, 8, 1, 21, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	// The client must send the token, station, and metric/UTC parameters.
	for _, want := range []string{"token=test-token", "stid=TT934", "units=metric", "obtimezone=utc"} {
		if !strings.Contains(*lastQuery, want) {
			t.Errorf("query %q missing %q", *lastQuery, want)
		}
	}

	if len(series) != 1 {
		t.Fatalf("got %d stations, want 1", len(series))
	}
	st := series[0]

	if st.Station.ID != "TT934" {
		t.Errorf("station ID: got %q", st.Station.ID)
	}
	// The API reports elevation in feet; 400 ft is 121.92 m.
	if got := float64(st.Station.Elevation); math.Abs(got-121.92) > 0.01 {
		t.Errorf("elevation: got %v m, want 121.92", got)
	}
	if len(st.Observations) != 3 {
		t.Fatalf("got %d observations, want 3", len(st.Observations))
	}

	first := st.Observations[0]
	if v, ok := first.Temperature.Get(); !ok || v != 24.4 {
		t.Errorf("obs 0 temperature: got %v %v, want 24.4", v, ok)
	}
	if v, ok := first.WindSpeed.Get(); !ok || v != 3.6 {
		t.Errorf("obs 0 wind speed: got %v %v, want 3.6 m/s", v, ok)
	}
	if v, ok := first.Precipitation.Get(); !ok || v != 0 {
		t.Errorf("obs 0 precipitation: got %v %v, want 0", v, ok)
	}
	if !first.Time.Equal(time.Date(2024, 8, 1, 19, 0, 0, 0, time.UTC)) {
		t.Errorf("obs 0 time: got %v", first.Time)
	}

	// A null value in the API array must become an absent Opt, never a zero.
	if st.Observations[2].Temperature.Valid() {
		t.Error("obs 2 temperature was null and must be absent")
	}
}

func TestTimeSeriesErrors(t *testing.T) {
	t.Run("no token", func(t *testing.T) {
		c := synoptic.New(synoptic.WithBaseURL("http://example.invalid"))
		_, err := c.TimeSeries(context.Background(), synoptic.TimeSeriesRequest{Stations: []string{"X"}})
		if err == nil {
			t.Fatal("want an error without a token")
		}
	})

	t.Run("no stations", func(t *testing.T) {
		c := synoptic.New(synoptic.WithToken("t"))
		_, err := c.TimeSeries(context.Background(), synoptic.TimeSeriesRequest{})
		if err == nil {
			t.Fatal("want an error without stations")
		}
	})

	t.Run("api error code", func(t *testing.T) {
		srv, _ := fixtureServer(t, "testdata/error.json", http.StatusOK)
		c := synoptic.New(synoptic.WithToken("t"), synoptic.WithBaseURL(srv.URL), synoptic.WithHTTPClient(srv.Client()))
		_, err := c.TimeSeries(context.Background(), synoptic.TimeSeriesRequest{Stations: []string{"X"}})
		if err == nil {
			t.Fatal("want an error when the API reports a bad response code")
		}
	})

	t.Run("http status", func(t *testing.T) {
		srv, _ := fixtureServer(t, "testdata/timeseries.json", http.StatusInternalServerError)
		c := synoptic.New(synoptic.WithToken("t"), synoptic.WithBaseURL(srv.URL), synoptic.WithHTTPClient(srv.Client()))
		_, err := c.TimeSeries(context.Background(), synoptic.TimeSeriesRequest{Stations: []string{"X"}})
		if err == nil {
			t.Fatal("want an error on a non-200 status")
		}
	})
}

// driveModel confirms the parsed observation feeds the firewx models.
func TestParsedObsDrivesModel(t *testing.T) {
	srv, _ := fixtureServer(t, "testdata/timeseries.json", http.StatusOK)
	c := synoptic.New(synoptic.WithToken("t"), synoptic.WithBaseURL(srv.URL), synoptic.WithHTTPClient(srv.Client()))
	series, err := c.TimeSeries(context.Background(), synoptic.TimeSeriesRequest{Stations: []string{"TT934"}})
	if err != nil {
		t.Fatal(err)
	}
	o := series[0].Observations[0]
	// Temperature and humidity are present, so the dew point is present.
	if _, ok := o.DewPoint().Get(); !ok {
		t.Error("dew point should be present from a parsed observation")
	}
}
