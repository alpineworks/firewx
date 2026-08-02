package fems

import (
	"context"
	"fmt"
	"net/url"
	"time"

	firewx "alpineworks.io/firewx"
)

// StationSeries is the observations of one station, with the station metadata.
// The weather download does not include the station coordinates, so the
// Latitude, Longitude, and Elevation of the Station are zero. Use the FEMS site
// metadata for the coordinates.
type StationSeries struct {
	Station      firewx.Station
	Observations []firewx.Obs
}

// Weather reads the RAWS weather observations for the request and returns one
// StationSeries per station. FEMS reports weather in Fahrenheit, inches, and
// miles per hour; the client converts each value to SI.
func (c *Client) Weather(ctx context.Context, req Request) ([]StationSeries, error) {
	extra := url.Values{}
	extra.Set("dataset", "observation")
	extra.Set("dataIncrement", "hourly")
	extra.Set("stationtypes", "RAWS(SATNFDRS)")

	table, err := c.fetchCSV(ctx, "download-weather", extra, req)
	if err != nil {
		return nil, err
	}

	// Group the rows by station, in the order the stations first appear.
	order := []string{}
	byStation := map[string]*StationSeries{}

	for _, row := range table.rows {
		id := table.cell(row, "StationId")
		if id == "" {
			continue
		}
		series, ok := byStation[id]
		if !ok {
			series = &StationSeries{Station: firewx.Station{ID: id, Name: table.cell(row, "StationName")}}
			byStation[id] = series
			order = append(order, id)
		}

		o, err := weatherObs(table, row)
		if err != nil {
			return nil, err
		}
		series.Observations = append(series.Observations, o)
	}

	out := make([]StationSeries, 0, len(order))
	for _, id := range order {
		out = append(out, *byStation[id])
	}
	return out, nil
}

// weatherObs maps one CSV row to a firewx.Obs. An empty cell becomes an absent
// Opt, never a zero.
func weatherObs(table *csvTable, row []string) (firewx.Obs, error) {
	t, err := time.Parse(time.RFC3339, table.cell(row, "DateTime"))
	if err != nil {
		return firewx.Obs{}, fmt.Errorf("fems: time %q: %w", table.cell(row, "DateTime"), err)
	}
	o := firewx.Obs{Time: t.UTC()}

	if v, ok := optFloat(table.cell(row, "Temperature(F)")); ok {
		o.Temperature = firewx.Some(firewx.Fahrenheit(v).Celsius())
	}
	if v, ok := optFloat(table.cell(row, "RelativeHumidity(%)")); ok {
		o.RelativeHumidity = firewx.Some(firewx.Percent(v))
	}
	if v, ok := optFloat(table.cell(row, "Precipitation(in)")); ok {
		o.Precipitation = firewx.Some(firewx.Inches(v).Millimeters())
	}
	if v, ok := optFloat(table.cell(row, "WindSpeed(mph)")); ok {
		o.WindSpeed = firewx.Some(firewx.MilesPerHour(v).MetersPerSecond())
	}
	if v, ok := optFloat(table.cell(row, "WindAzimuth(degrees)")); ok {
		o.WindDirection = firewx.Some(firewx.Degrees(v))
	}
	if v, ok := optFloat(table.cell(row, "GustSpeed(mph)")); ok {
		o.WindGust = firewx.Some(firewx.MilesPerHour(v).MetersPerSecond())
	}
	if v, ok := optFloat(table.cell(row, "SolarRadiation(W/m2)")); ok {
		o.SolarRadiation = firewx.Some(firewx.WattsPerSquareMeter(v))
	}
	if v, ok := optFloat(table.cell(row, "SnowFlag")); ok {
		o.SnowCovered = firewx.Some(v != 0)
	}

	return o, nil
}
