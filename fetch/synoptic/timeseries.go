package synoptic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	firewx "alpineworks.io/firewx"
)

// fireWeatherVars are the observation variables this client requests when a
// caller does not name its own. They cover the inputs of the fire danger models.
var fireWeatherVars = []string{
	"air_temp",
	"relative_humidity",
	"wind_speed",
	"wind_gust",
	"wind_direction",
	"solar_radiation",
	"precip_accum",
}

// TimeSeriesRequest asks for the observations of one or more stations over a
// time range. Start and End are in any time zone; the client converts them to
// UTC for the API.
type TimeSeriesRequest struct {
	// Stations is the list of station identifiers, for example "KSLC".
	Stations []string

	// Start and End bound the time range.
	Start, End time.Time

	// Variables is the list of API variable names to request. If it is empty,
	// the client requests the fire weather set.
	Variables []string
}

// StationSeries is the observations of one station, with the station metadata.
type StationSeries struct {
	Station      firewx.Station
	Observations []firewx.Obs
}

// TimeSeries reads the observations for the request and returns one
// StationSeries per station. It returns an error if the token is absent, the
// request fails, or the API reports an error.
func (c *Client) TimeSeries(ctx context.Context, req TimeSeriesRequest) ([]StationSeries, error) {
	if c.token == "" {
		return nil, fmt.Errorf("synoptic: no token; use WithToken")
	}
	if len(req.Stations) == 0 {
		return nil, fmt.Errorf("synoptic: no stations in the request")
	}

	vars := req.Variables
	if len(vars) == 0 {
		vars = fireWeatherVars
	}

	q := url.Values{}
	q.Set("token", c.token)
	q.Set("stid", strings.Join(req.Stations, ","))
	q.Set("start", req.Start.UTC().Format("200601021504"))
	q.Set("end", req.End.UTC().Format("200601021504"))
	q.Set("vars", strings.Join(vars, ","))
	q.Set("units", "metric")
	q.Set("obtimezone", "utc")
	q.Set("output", "json")

	endpoint := c.baseURL + "/stations/timeseries?" + q.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("synoptic: build request: %w", err)
	}
	httpReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("synoptic: do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, fmt.Errorf("synoptic: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("synoptic: http status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed timeSeriesResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("synoptic: decode body: %w", err)
	}
	if parsed.Summary.ResponseCode != 1 {
		return nil, fmt.Errorf("synoptic: api error %d: %s", parsed.Summary.ResponseCode, parsed.Summary.ResponseMessage)
	}

	out := make([]StationSeries, 0, len(parsed.Station))
	for _, st := range parsed.Station {
		series, err := st.toStationSeries()
		if err != nil {
			return nil, err
		}
		out = append(out, series)
	}
	return out, nil
}

// timeSeriesResponse is the JSON body of the time series endpoint.
type timeSeriesResponse struct {
	Summary summary           `json:"SUMMARY"`
	Units   map[string]string `json:"UNITS"`
	Station []rawStation      `json:"STATION"`
}

type summary struct {
	ResponseCode    int    `json:"RESPONSE_CODE"`
	ResponseMessage string `json:"RESPONSE_MESSAGE"`
}

type rawStation struct {
	STID         string                     `json:"STID"`
	Name         string                     `json:"NAME"`
	Latitude     float64                    `json:"LATITUDE"`
	Longitude    float64                    `json:"LONGITUDE"`
	Elevation    float64                    `json:"ELEVATION"`
	Observations map[string]json.RawMessage `json:"OBSERVATIONS"`
}

// toStationSeries maps one raw station to a StationSeries. The API reports the
// station elevation in feet, so the client converts it to metres.
func (s rawStation) toStationSeries() (StationSeries, error) {
	station := firewx.Station{
		ID:        s.STID,
		Name:      s.Name,
		Latitude:  s.Latitude,
		Longitude: s.Longitude,
		Elevation: firewx.Feet(s.Elevation).Meters(),
	}

	times, err := decodeStringSeries(s.Observations["date_time"])
	if err != nil {
		return StationSeries{}, fmt.Errorf("synoptic: station %s: date_time: %w", s.STID, err)
	}

	temp := s.series("air_temp")
	rh := s.series("relative_humidity")
	wind := s.series("wind_speed")
	gust := s.series("wind_gust")
	dir := s.series("wind_direction")
	solar := s.series("solar_radiation")
	precip := s.series("precip_accum")

	obs := make([]firewx.Obs, 0, len(times))
	for i, ts := range times {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			return StationSeries{}, fmt.Errorf("synoptic: station %s: time %q: %w", s.STID, ts, err)
		}
		o := firewx.Obs{Time: t.UTC()}
		if v := at(temp, i); v != nil {
			o.Temperature = firewx.Some(firewx.Celsius(*v))
		}
		if v := at(rh, i); v != nil {
			o.RelativeHumidity = firewx.Some(firewx.Percent(*v))
		}
		if v := at(wind, i); v != nil {
			o.WindSpeed = firewx.Some(firewx.MetersPerSecond(*v))
		}
		if v := at(gust, i); v != nil {
			o.WindGust = firewx.Some(firewx.MetersPerSecond(*v))
		}
		if v := at(dir, i); v != nil {
			o.WindDirection = firewx.Some(firewx.Degrees(*v))
		}
		if v := at(solar, i); v != nil {
			o.SolarRadiation = firewx.Some(firewx.WattsPerSquareMeter(*v))
		}
		if v := at(precip, i); v != nil {
			o.Precipitation = firewx.Some(firewx.Millimeters(*v))
		}
		obs = append(obs, o)
	}

	return StationSeries{Station: station, Observations: obs}, nil
}

// series returns the value array for a variable. The API names a variable's
// array "<name>_set_<sensor>", so the client matches by prefix and takes the
// first sensor it finds. A returned nil means the station did not report the
// variable.
func (s rawStation) series(name string) []*float64 {
	prefix := name + "_set_"
	for key, raw := range s.Observations {
		if strings.HasPrefix(key, prefix) {
			v, err := decodeFloatSeries(raw)
			if err != nil {
				return nil
			}
			return v
		}
	}
	return nil
}

func decodeStringSeries(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var s []string
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return s, nil
}

func decodeFloatSeries(raw json.RawMessage) ([]*float64, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var s []*float64
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	return s, nil
}

// at returns the pointer at index i, or nil if the array is too short.
func at(s []*float64, i int) *float64 {
	if i < len(s) {
		return s[i]
	}
	return nil
}
