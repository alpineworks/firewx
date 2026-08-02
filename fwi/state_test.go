package fwi

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"

	firewx "alpineworks.io/firewx"
)

func TestStepGoldenDay1(t *testing.T) {
	// Day 1 of the standard test dataset: April 13 1985, T=17C, RH=42%,
	// wind=25 km/h, rain=0, lat=40, from the standard start-up codes. Computed
	// from the cffdrs equations.
	s := NewState()
	got := s.Step(Weather{Temperature: 17, Humidity: 42, Wind: 25, Rain: 0, Month: time.April, Latitude: 40})

	cases := []struct {
		name string
		got  float64
		want float64
		tol  float64
	}{
		{"FFMC", float64(got.FFMC), 87.65, 0.02},
		{"DMC", float64(got.DMC), 8.55, 0.02},
		{"DC", float64(got.DC), 19.01, 0.02},
		{"ISI", float64(got.ISI), 10.78, 0.02},
		{"BUI", float64(got.BUI), 8.49, 0.02},
		{"FWI", float64(got.FWI), 10.04, 0.05},
		{"DSR", float64(got.DSR), 1.61, 0.05},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closeTo(t, tc.got, tc.want, tc.tol, tc.name)
		})
	}

	// The driver must carry the codes forward into its own state.
	if s.FFMC != got.FFMC || s.DMC != got.DMC || s.DC != got.DC {
		t.Error("state codes must equal the returned output codes")
	}
}

func TestStepGoldenDay2(t *testing.T) {
	// Day 2 of the standard dataset: April 14, T=20C, RH=21%, wind=25 km/h,
	// rain=2.4 mm, from the day-1 codes. This pins one carryover step and
	// exercises the FFMC rain branch (rain > 0.5) and the DMC rain branch
	// (rain > 1.5). Computed from the cffdrs equations.
	s := NewState()
	s.Step(Weather{Temperature: 17, Humidity: 42, Wind: 25, Rain: 0, Month: time.April, Latitude: 40})
	got := s.Step(Weather{Temperature: 20, Humidity: 21, Wind: 25, Rain: 2.4, Month: time.April, Latitude: 40})

	cases := []struct {
		name string
		got  float64
		want float64
		tol  float64
	}{
		{"FFMC", float64(got.FFMC), 86.20, 0.02},
		{"DMC", float64(got.DMC), 10.40, 0.02},
		{"DC", float64(got.DC), 23.57, 0.02},
		{"ISI", float64(got.ISI), 8.77, 0.02},
		{"BUI", float64(got.BUI), 10.35, 0.02},
		{"FWI", float64(got.FWI), 9.22, 0.05},
		{"DSR", float64(got.DSR), 1.39, 0.05},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closeTo(t, tc.got, tc.want, tc.tol, tc.name)
		})
	}
}

// day is one row of testdata/test_fwi.csv.
type day struct {
	mon              time.Month
	lat              float64
	temp, rh, ws, pr float64
}

func loadTestFWI(t *testing.T) []day {
	t.Helper()
	f, err := os.Open("testdata/test_fwi.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	num := func(s string) float64 {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			t.Fatalf("bad number %q: %v", s, err)
		}
		return v
	}
	var days []day
	for _, r := range rows[1:] { // skip header: long,lat,yr,mon,day,temp,rh,ws,prec
		days = append(days, day{
			lat:  num(r[1]),
			mon:  time.Month(int(num(r[3]))),
			temp: num(r[5]),
			rh:   num(r[6]),
			ws:   num(r[7]),
			pr:   num(r[8]),
		})
	}
	return days
}

func TestStepSequenceStaysInRange(t *testing.T) {
	// Run the whole standard dataset and check the invariants over every day:
	// FFMC stays in [0,101], the codes stay non-negative, and every output is a
	// real number.
	days := loadTestFWI(t)
	s := NewState()
	for i, d := range days {
		o := s.Step(Weather{
			Temperature: firewx.Celsius(d.temp),
			Humidity:    firewx.Percent(d.rh),
			Wind:        firewx.KilometersPerHour(d.ws),
			Rain:        firewx.Millimeters(d.pr),
			Month:       d.mon,
			Latitude:    d.lat,
		})
		if o.FFMC < 0 || o.FFMC > 101 {
			t.Errorf("day %d: FFMC out of range: %v", i+1, o.FFMC)
		}
		for _, v := range []struct {
			name string
			val  float64
		}{
			{"DMC", float64(o.DMC)}, {"DC", float64(o.DC)}, {"ISI", float64(o.ISI)},
			{"BUI", float64(o.BUI)}, {"FWI", float64(o.FWI)}, {"DSR", float64(o.DSR)},
		} {
			if v.val < 0 {
				t.Errorf("day %d: %s negative: %v", i+1, v.name, v.val)
			}
			if isNaNOrInf(v.val) {
				t.Errorf("day %d: %s not finite: %v", i+1, v.name, v.val)
			}
		}
	}
}

func TestStepObs(t *testing.T) {
	cases := []struct {
		name        string
		obs         firewx.Obs
		wantApplied bool
	}{
		{
			name: "applies with all four inputs",
			obs: firewx.Obs{
				Temperature:      firewx.Some(firewx.Celsius(17)),
				RelativeHumidity: firewx.Some(firewx.Percent(42)),
				WindSpeed:        firewx.Some(firewx.MetersPerSecond(6.944444)), // 25 km/h
				Precipitation:    firewx.Some(firewx.Millimeters(0)),
			},
			wantApplied: true,
		},
		{
			name: "skips without wind",
			obs: firewx.Obs{
				Temperature:      firewx.Some(firewx.Celsius(17)),
				RelativeHumidity: firewx.Some(firewx.Percent(42)),
				Precipitation:    firewx.Some(firewx.Millimeters(0)),
			},
			wantApplied: false,
		},
		{
			name: "skips without temperature",
			obs: firewx.Obs{
				RelativeHumidity: firewx.Some(firewx.Percent(42)),
				WindSpeed:        firewx.Some(firewx.MetersPerSecond(6.944444)),
				Precipitation:    firewx.Some(firewx.Millimeters(0)),
			},
			wantApplied: false,
		},
		{
			name: "skips without humidity",
			obs: firewx.Obs{
				Temperature:   firewx.Some(firewx.Celsius(17)),
				WindSpeed:     firewx.Some(firewx.MetersPerSecond(6.944444)),
				Precipitation: firewx.Some(firewx.Millimeters(0)),
			},
			wantApplied: false,
		},
		{
			name: "skips without precipitation",
			obs: firewx.Obs{
				Temperature:      firewx.Some(firewx.Celsius(17)),
				RelativeHumidity: firewx.Some(firewx.Percent(42)),
				WindSpeed:        firewx.Some(firewx.MetersPerSecond(6.944444)),
			},
			wantApplied: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewState()
			_, applied := s.StepObs(tc.obs, time.April, 40)
			if applied != tc.wantApplied {
				t.Fatalf("applied=%v, want %v", applied, tc.wantApplied)
			}
			if !tc.wantApplied && (s.FFMC != StartupFFMC || s.DMC != StartupDMC || s.DC != StartupDC) {
				t.Error("state must be untouched when an input is absent")
			}
		})
	}

	// The obs path must match the explicit path: 6.944444 m/s = 25 km/h.
	s1 := NewState()
	o1, _ := s1.StepObs(cases[0].obs, time.April, 40)
	s2 := NewState()
	o2 := s2.Step(Weather{Temperature: 17, Humidity: 42, Wind: 25, Rain: 0, Month: time.April, Latitude: 40})
	closeTo(t, float64(o1.FFMC), float64(o2.FFMC), 1e-6, "StepObs FFMC matches Step")
}

func TestStateJSONRoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		state State
	}{
		{"start-up", NewState()},
		{"mid season", State{SchemaVersion: schemaVersion, FFMC: 91.234, DMC: 45.6, DC: 312.75}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.state)
			if err != nil {
				t.Fatal(err)
			}
			var back State
			if err := json.Unmarshal(b, &back); err != nil {
				t.Fatal(err)
			}
			if back != tc.state {
				t.Errorf("round trip: got %+v, want %+v", back, tc.state)
			}
		})
	}
}
