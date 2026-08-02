package firewx

import "time"

// Station describes the site an observation came from, including the terms
// needed to correct a non-standard sensor exposure toward the reference
// exposure each fire danger model assumes.
type Station struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`

	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Elevation Meters  `json:"elevation"`

	// LSTOffset is the fixed offset from UTC that defines local standard time
	// at this station.
	//
	// This is deliberately a fixed offset and not a *time.Location. Both the
	// Canadian FWI system and NFDRS define their observation windows in local
	// standard time, which does not shift with daylight saving. Using a
	// zoneinfo location here introduces a one hour discontinuity in the
	// carryover codes twice a year that is nearly invisible in the output and
	// extremely annoying to find.
	LSTOffset time.Duration `json:"lst_offset"`

	// AnemometerHeight is the height of the wind sensor above ground.
	//
	// The Canadian FWI system assumes 10 m open exposure; NFDRS assumes 20 ft
	// (6.096 m). A backyard installation is usually lower than both and more
	// sheltered than either, and the Initial Spread Index is sensitive enough
	// to wind that ignoring this materially changes the output.
	AnemometerHeight Meters `json:"anemometer_height"`

	// Roughness is the aerodynamic roughness length of the surrounding
	// terrain, used to correct wind speed between heights.
	Roughness Roughness `json:"roughness"`
}

// LocalStandardTime converts t to the station's local standard time. The
// returned time carries a fixed zone and never observes daylight saving.
func (s Station) LocalStandardTime(t time.Time) time.Time {
	return t.In(time.FixedZone("LST", int(s.LSTOffset/time.Second)))
}

// Obs is a single weather observation.
//
// Time is always UTC. Every measurement is optional, because every sensor can
// fail independently of the others; a station that has lost its pyranometer
// can still drive the Canadian FWI system, and one that has lost its rain
// gauge cannot drive either system correctly but should fail loudly rather
// than silently assume zero.
type Obs struct {
	Time time.Time `json:"time"`

	Temperature      Opt[Celsius]             `json:"temperature"`
	RelativeHumidity Opt[Percent]             `json:"relative_humidity"`
	WindSpeed        Opt[MetersPerSecond]     `json:"wind_speed"`
	WindDirection    Opt[Degrees]             `json:"wind_direction"`
	WindGust         Opt[MetersPerSecond]     `json:"wind_gust"`
	SolarRadiation   Opt[WattsPerSquareMeter] `json:"solar_radiation"`
	Pressure         Opt[Hectopascals]        `json:"pressure"`

	// Precipitation accumulated since the previous observation, not since
	// midnight and not a rate. Accumulation windows are a common source of
	// silent error when merging sources; normalise on ingest.
	Precipitation Opt[Millimeters] `json:"precipitation"`

	// PrecipDuration is the time during which precipitation actually fell
	// within the accumulation window. NFDRS requires this, and most stations
	// can only estimate it. A tipping bucket logged at a short archive
	// interval can derive it directly, which is one of the few respects in
	// which a well-configured personal station beats a RAWS.
	PrecipDuration Opt[time.Duration] `json:"precip_duration"`

	// SnowCovered suppresses fire danger output when set. NFDRS treats a
	// snow-covered station as having no fire danger regardless of the other
	// measurements.
	SnowCovered Opt[bool] `json:"snow_covered"`
}

// DewPoint returns the dew point derived from temperature and relative
// humidity, or an empty Opt if either input is missing.
func (o Obs) DewPoint() Opt[Celsius] {
	t, okT := o.Temperature.Get()
	rh, okRH := o.RelativeHumidity.Get()
	if !okT || !okRH {
		return None[Celsius]()
	}
	return Some(DewPoint(t, rh))
}

// VaporPressureDeficit returns the VPD derived from temperature and relative
// humidity, or an empty Opt if either input is missing.
func (o Obs) VaporPressureDeficit() Opt[Kilopascals] {
	t, okT := o.Temperature.Get()
	rh, okRH := o.RelativeHumidity.Get()
	if !okT || !okRH {
		return None[Kilopascals]()
	}
	return Some(VaporPressureDeficit(t, rh))
}
