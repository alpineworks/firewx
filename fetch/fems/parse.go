package fems

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Request asks for the data of one or more stations over a time range. Start and
// End are in any time zone; the client converts them to UTC for the API.
type Request struct {
	Stations   []string
	Start, End time.Time
}

// fetchCSV builds the request, sends it, and parses the CSV body. The path is
// the endpoint under the base URL, for example "download-weather". The extra
// values are the endpoint-specific query parameters.
func (c *Client) fetchCSV(ctx context.Context, path string, extra url.Values, req Request) (*csvTable, error) {
	if len(req.Stations) == 0 {
		return nil, fmt.Errorf("fems: no stations in the request")
	}

	q := url.Values{}
	q.Set("stationIds", strings.Join(req.Stations, ","))
	q.Set("startDate", req.Start.UTC().Format(time.RFC3339))
	q.Set("endDate", req.End.UTC().Format(time.RFC3339))
	q.Set("dataFormat", "csv")
	for key, vals := range extra {
		for _, v := range vals {
			q.Add(key, v)
		}
	}

	endpoint := c.baseURL + "/" + path + "?" + q.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("fems: build request: %w", err)
	}
	httpReq.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("fems: do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<20))
	if err != nil {
		return nil, fmt.Errorf("fems: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fems: http status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return parseCSV(bytes.NewReader(body))
}

// csvTable is a parsed CSV body with a lookup from a column name to its index.
type csvTable struct {
	index map[string]int
	rows  [][]string
}

func parseCSV(r io.Reader) (*csvTable, error) {
	rd := csv.NewReader(r)
	rd.FieldsPerRecord = -1
	recs, err := rd.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("fems: parse csv: %w", err)
	}
	if len(recs) == 0 {
		return &csvTable{index: map[string]int{}}, nil
	}
	idx := make(map[string]int, len(recs[0]))
	for i, h := range recs[0] {
		idx[strings.TrimSpace(h)] = i
	}
	return &csvTable{index: idx, rows: recs[1:]}, nil
}

// cell returns the value of a column in a row, trimmed. It returns an empty
// string if the column or the value is absent.
func (t *csvTable) cell(row []string, col string) string {
	i, ok := t.index[col]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

// optFloat parses a cell as a float. An empty cell returns ok false, which the
// caller turns into an absent Opt.
func optFloat(s string) (float64, bool) {
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}
