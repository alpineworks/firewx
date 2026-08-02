package fems_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"alpineworks.io/firewx/fetch/fems"
)

func TestWeather(t *testing.T) {
	srv, lastQuery := fixtureServer(t, "testdata/weather.csv", http.StatusOK)
	c := fems.New(fems.WithBaseURL(srv.URL), fems.WithHTTPClient(srv.Client()))

	series, err := c.Weather(context.Background(), fems.Request{
		Stations: []string{"20284"},
		Start:    time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC),
		End:      time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"stationIds=20284", "dataFormat=csv", "dataset=observation"} {
		if !strings.Contains(*lastQuery, want) {
			t.Errorf("query %q missing %q", *lastQuery, want)
		}
	}

	if len(series) != 1 {
		t.Fatalf("got %d stations, want 1", len(series))
	}
	st := series[0]
	if st.Station.ID != "20284" || st.Station.Name != "GREENBASE" {
		t.Errorf("station: got %q %q", st.Station.ID, st.Station.Name)
	}
	if len(st.Observations) != 4 {
		t.Fatalf("got %d observations, want 4", len(st.Observations))
	}

	// Row 1: units convert from English to SI. 87 F = 30.556 C; 9 mph = 4.023 m/s.
	first := st.Observations[0]
	if v, ok := first.Temperature.Get(); !ok {
		t.Error("obs 0 temperature absent")
	} else {
		closeTo(t, float64(v), 30.556, 0.01, "temperature F->C")
	}
	if v, ok := first.WindSpeed.Get(); !ok {
		t.Error("obs 0 wind speed absent")
	} else {
		closeTo(t, float64(v), 4.023, 0.01, "wind speed mph->m/s")
	}
	if v, ok := first.RelativeHumidity.Get(); !ok || v != 25 {
		t.Errorf("obs 0 RH: got %v %v, want 25", v, ok)
	}
	if v, ok := first.SolarRadiation.Get(); !ok || v != 555 {
		t.Errorf("obs 0 solar: got %v %v, want 555", v, ok)
	}
	if snow, ok := first.SnowCovered.Get(); !ok || snow {
		t.Errorf("obs 0 snow: got %v %v, want false", snow, ok)
	}
	if !first.Time.Equal(time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("obs 0 time: got %v", first.Time)
	}

	// Row 4 is a sensor dropout: empty temperature and humidity cells must be
	// absent, but the wind, which was reported, must be present.
	last := st.Observations[3]
	if last.Temperature.Valid() || last.RelativeHumidity.Valid() {
		t.Error("obs 3 temperature and humidity were empty and must be absent")
	}
	if !last.WindSpeed.Valid() {
		t.Error("obs 3 wind speed was reported and must be present")
	}
}

func TestWeatherDrivesModel(t *testing.T) {
	srv, _ := fixtureServer(t, "testdata/weather.csv", http.StatusOK)
	c := fems.New(fems.WithBaseURL(srv.URL), fems.WithHTTPClient(srv.Client()))
	series, err := c.Weather(context.Background(), fems.Request{Stations: []string{"20284"}})
	if err != nil {
		t.Fatal(err)
	}
	// Temperature and humidity are present, so the dew point is present.
	if _, ok := series[0].Observations[0].DewPoint().Get(); !ok {
		t.Error("dew point should be present from a parsed observation")
	}
}

func TestWeatherErrors(t *testing.T) {
	t.Run("no stations", func(t *testing.T) {
		c := fems.New()
		_, err := c.Weather(context.Background(), fems.Request{})
		if err == nil {
			t.Fatal("want an error without stations")
		}
	})
	t.Run("http status", func(t *testing.T) {
		srv, _ := fixtureServer(t, "testdata/weather.csv", http.StatusBadGateway)
		c := fems.New(fems.WithBaseURL(srv.URL), fems.WithHTTPClient(srv.Client()))
		_, err := c.Weather(context.Background(), fems.Request{Stations: []string{"20284"}})
		if err == nil {
			t.Fatal("want an error on a non-200 status")
		}
	})
}
