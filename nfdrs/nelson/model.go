package nelson

import (
	"math"
	"sort"
	"time"

	firewx "alpineworks.io/firewx"
)

// schemaVersion is the on-disk schema version stamped into a Stick. Increase it
// when the meaning of a field changes, so the code can identify an old persisted
// stick and change it correctly.
const schemaVersion = 1

// defaultBarometricPressure is the stick barometric pressure (cal/cm3) that
// firelab/NFDRS4 uses when it initializes the environment. The Nelson model
// runs at a constant pressure when the caller gives no measured pressure.
const defaultBarometricPressure = 0.0218

// seaLevelHpa and seaLevelCalPerCm3 convert a measured barometric pressure from
// hectopascals to the cal/cm3 unit that the model equations use. The comment in
// firelab/NFDRS4 gives 0.0242 cal/cm3 as the sea-level atmospheric pressure.
const (
	seaLevelHpa       = 1013.25
	seaLevelCalPerCm3 = 0.0242
)

// State is the prevailing dead fuel moisture state after an update. The state
// names the physical process that governs the surface node during the step.
// The values match the DFM_State enumeration in firelab/NFDRS4.
type State int

const (
	StateNone State = iota
	StateAdsorption
	StateDesorption
	StateCondensation1
	StateCondensation2
	StateEvaporation
	StateRainfall1
	StateRainfall2
	StateRainstorm
	StateStagnation
	StateError
)

// numStates is the number of State values. It matches DFM_States in
// firelab/NFDRS4.
const numStates = 11

// String returns the name of the state.
func (s State) String() string {
	switch s {
	case StateNone:
		return "None"
	case StateAdsorption:
		return "Adsorption"
	case StateDesorption:
		return "Desorption"
	case StateCondensation1:
		return "Condensation1"
	case StateCondensation2:
		return "Condensation2"
	case StateEvaporation:
		return "Evaporation"
	case StateRainfall1:
		return "Rainfall1"
	case StateRainfall2:
		return "Rainfall2"
	case StateRainstorm:
		return "Rainstorm"
	case StateStagnation:
		return "Stagnation"
	case StateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Weather is one weather observation for a Stick update. Every field has a
// named unit type. The rainfall is the amount since the previous observation,
// not the amount since midnight and not a rate.
type Weather struct {
	// Elapsed is the time since the previous observation.
	Elapsed time.Duration

	Temperature      firewx.Celsius
	RelativeHumidity firewx.Percent
	SolarRadiation   firewx.WattsPerSquareMeter

	// Rainfall is the precipitation amount since the previous observation.
	Rainfall firewx.Millimeters

	// Pressure is the barometric pressure. When it is absent, the model uses a
	// fixed default pressure, which matches firelab/NFDRS4.
	Pressure firewx.Opt[firewx.Hectopascals]
}

// Stick is a stateful Nelson dead fuel moisture stick. It carries the full
// radial profile of temperature, moisture, and diffusivity between updates.
//
// The exported fields hold the complete model state. A Stick marshals to JSON
// and back with an exact round trip, so a caller can persist the stick between
// runs and restore the radial nodes exactly. The unexported scratch slices are
// working buffers; each update overwrites them, so they need no persistence.
//
// A Stick is not safe for concurrent use.
type Stick struct {
	SchemaVersion int    `json:"schema_version"`
	Name          string `json:"name"`

	// Fixed model parameters, set at construction.
	Radius          float64 `json:"radius"`           // stick radius (cm)
	Length          float64 `json:"length"`           // stick length (cm)
	Density         float64 `json:"density"`          // stick density (g/cm3)
	Nodes           int     `json:"nodes"`            // radial node count
	DSteps          int     `json:"d_steps"`          // diffusivity steps per observation
	MSteps          int     `json:"m_steps"`          // moisture steps per observation
	Hc              float64 `json:"hc"`               // planar heat transfer rate (cal/cm2-h-C)
	Rai0            float64 `json:"rai0"`             // rain runoff factor, first hour
	Rai1            float64 `json:"rai1"`             // rain runoff factor, later hours
	Stca            float64 `json:"stca"`             // adsorption mass transfer rate
	Stcd            float64 `json:"stcd"`             // desorption mass transfer rate
	Stv             float64 `json:"stv"`              // storm transition value (cm/h)
	Wfilmk          float64 `json:"wfilmk"`           // water film contribution (g/g)
	Wmx             float64 `json:"wmx"`              // maximum local moisture (g/g)
	AllowRainfall2  bool    `json:"allow_rainfall2"`  // apply Nelson later-hour runoff logic
	AllowRainstorm  bool    `json:"allow_rainstorm"`  // apply Nelson rainstorm logic
	PertubateColumn bool    `json:"pertubate_column"` // perturb continuous liquid columns
	RampRai0        bool    `json:"ramp_rai0"`        // ramp runoff after Bevins, not Nelson

	// Derived parameters, computed at construction.
	Dx   float64   `json:"dx"`   // internodal distance (cm)
	Dx2  float64   `json:"dx2"`  // two times Dx (cm)
	Wmax float64   `json:"wmax"` // maximum possible moisture (g/g)
	Amlf float64   `json:"amlf"` // evaporation and condensation factor
	Capf float64   `json:"capf"` // capillary factor
	Hwf  float64   `json:"hwf"`  // vapor diffusion factor
	Vf   float64   `json:"vf"`   // free water transport factor
	X    []float64 `json:"x"`    // nodal radial distances from centre (cm)
	V    []float64 `json:"v"`    // nodal volume fractions (dimensionless)

	// Previous observation.
	Ta0  float64 `json:"ta0"`  // air temperature (C)
	Ha0  float64 `json:"ha0"`  // air humidity (g/g)
	Sv0  float64 `json:"sv0"`  // solar radiation (mV)
	Rc0  float64 `json:"rc0"`  // cumulative rainfall (cm)
	Ra0  float64 `json:"ra0"`  // period rainfall (cm)
	Bp0  float64 `json:"bp0"`  // barometric pressure (cal/cm3)
	Init bool    `json:"init"` // environment initialized

	// Current observation.
	Ta1 float64 `json:"ta1"` // air temperature (C)
	Ha1 float64 `json:"ha1"` // air humidity (g/g)
	Sv1 float64 `json:"sv1"` // solar radiation (mV)
	Rc1 float64 `json:"rc1"` // cumulative rainfall (cm)
	Bp1 float64 `json:"bp1"` // barometric pressure (cal/cm3)
	Et  float64 `json:"et"`  // elapsed time since previous observation (h)

	// Time-step intermediates.
	Ddt     float64 `json:"ddt"`     // diffusivity time step (h)
	Mdt     float64 `json:"mdt"`     // moisture time step (h)
	Mdt2    float64 `json:"mdt2"`    // two times Mdt (h)
	Pptrate float64 `json:"pptrate"` // rainfall rate over Pi (cm/h)
	Ra1     float64 `json:"ra1"`     // period rainfall (cm)
	Rdur    float64 `json:"rdur"`    // rainfall duration (h)
	Sf      float64 `json:"sf"`      // Nelson "s" factor

	// Nodal state. These arrays must survive the JSON round trip exactly.
	T []float64 `json:"t"` // nodal temperatures (C)
	S []float64 `json:"s"` // nodal fiber saturation points (g/g)
	D []float64 `json:"d"` // nodal bound water diffusivities (cm2/h)
	W []float64 `json:"w"` // nodal moisture contents (g/g)

	// Surface and output intermediates.
	Hf      float64 `json:"hf"`      // stick surface humidity (g/g)
	Wsa     float64 `json:"wsa"`     // fiber saturation point (g/g)
	Sem     float64 `json:"sem"`     // equilibrium moisture content (g/g)
	Wfilm   float64 `json:"wfilm"`   // water film amount (g/g)
	Elapsed float64 `json:"elapsed"` // total simulation time (h)
	Updates int64   `json:"updates"` // number of updates
	State   State   `json:"state"`   // prevailing state

	// Scratch buffers. Each update overwrites these, so they are not persisted.
	twold []float64
	tsold []float64
	ttold []float64
	tv    []float64
	to    []float64
	tg    []float64
}

// NewStandardStick returns a stick for one of the four standard time-lag
// classes. The parameters follow the initDeadFuelMoisture factory methods in
// firelab/NFDRS4.
func NewStandardStick(tl TimeLag) *Stick {
	spec := standardSticks[tl]
	name := map[TimeLag]string{
		OneHour:      "One Hour",
		TenHour:      "Ten Hour",
		HundredHour:  "Hundred Hour",
		ThousandHour: "Thousand Hour",
	}[tl]

	s := &Stick{}
	// First set the derived parameters from the radius.
	s.initializeParameters(spec.radius, name)
	// Then override the adsorption rate and the maximum local moisture, and
	// initialize the stick a second time. This matches the factory methods.
	// The factory also calls setMoisture(0.2), but the second initializeStick
	// overwrites the nodal moisture, so that call has no effect.
	s.Stca = spec.adsorptionRate
	s.Wmx = spec.maxLocalMoisture
	s.initializeStick()
	return s
}

// initializeParameters sets the stick parameters from the radius, after the
// derive functions of Bevins (2005), then initializes the stick. The fixed
// parameter values match initializeParameters in firelab/NFDRS4.
func (s *Stick) initializeParameters(radius float64, name string) {
	s.SchemaVersion = schemaVersion
	s.Name = name
	s.Radius = radius
	s.Length = 41.0
	s.Density = 0.400
	s.Nodes = deriveStickNodes(radius)
	s.MSteps = deriveMoistureSteps(radius)
	s.DSteps = deriveDiffusivitySteps(radius)
	s.Hc = derivePlanarHeatTransferRate(radius)
	s.Stca = deriveAdsorptionRate(radius)
	s.Rai0 = deriveRainfallRunoffFactor(radius)
	s.Wfilmk = 0.0
	s.Wmx = 0.60
	s.Stcd = 0.06
	s.Rai1 = 0.5
	s.Stv = 9999.0
	s.AllowRainfall2 = false
	s.AllowRainstorm = false
	s.PertubateColumn = false
	s.RampRai0 = false
	s.initializeStick()
}

// initializeStick derives the nodal geometry and the optimization factors, then
// initializes the environment. It matches initializeStick in firelab/NFDRS4.
func (s *Stick) initializeStick() {
	// Internodal distance (cm).
	s.Dx = s.Radius / float64(s.Nodes-1)
	s.Dx2 = s.Dx * 2.0

	// Maximum possible stick moisture content (g/g).
	s.Wmax = (1.0 / s.Density) - (1.0 / 1.53)

	// Initialize the nodal arrays to their default profiles.
	s.T = fillSlice(s.Nodes, 20.0)
	s.S = fillSlice(s.Nodes, 0.0)
	s.D = fillSlice(s.Nodes, 0.0)
	s.W = fillSlice(s.Nodes, 0.5*s.Wmx)

	// Derive nodal radial distances (cm), from the surface to the centre.
	s.X = make([]float64, 0, s.Nodes)
	for i := 0; i < s.Nodes-1; i++ {
		s.X = append(s.X, s.Radius-(s.Dx*float64(i)))
	}
	s.X = append(s.X, 0.0)

	// Derive nodal volume fractions (dimensionless).
	s.V = make([]float64, 0, s.Nodes)
	ro := s.Radius
	ri := ro - 0.5*s.Dx
	a2 := s.Radius * s.Radius
	s.V = append(s.V, (ro*ro-ri*ri)/a2)
	for i := 1; i < s.Nodes-1; i++ {
		ro = ri
		ri = ro - s.Dx
		s.V = append(s.V, (ro*ro-ri*ri)/a2)
	}
	s.V = append(s.V, ri*ri/a2)

	// Allocate the scratch buffers.
	s.allocScratch()

	// Initialize the environment, then clear the initialized flag.
	s.initializeEnvironment(20.0, 0.20, 0.0, 0.0, 20.0, 0.20, 0.5*s.Wmx, defaultBarometricPressure)
	s.Init = false

	// Computation optimization factors.
	s.Hwf = 0.622 * s.Hc * math.Pow(pr/sc, 0.667)
	s.Amlf = s.Hwf / (0.24 * s.Density * s.Radius)
	rcav := 0.5 * aw * wl
	s.Capf = 3600.0 * pi * st * rcav * rcav / (16.0 * s.Radius * s.Radius * s.Length * s.Density)
	s.Vf = st / (s.Density * wl * scr)
}

// initializeEnvironment sets the previous and current observation values and the
// initial nodal profiles, then computes the diffusivity. It matches
// initializeEnvironment in firelab/NFDRS4. Arguments: air temperature (C), air
// humidity (g/g), solar radiation (W/m2), cumulative rainfall (cm), initial
// stick temperature (C), initial surface humidity (g/g), initial moisture
// content (g/g), and barometric pressure (cal/cm3).
func (s *Stick) initializeEnvironment(ta, ha, sr, rc, ti, hi, wi, bp float64) {
	s.Ta0, s.Ta1 = ta, ta
	s.Ha0, s.Ha1 = ha, ha
	s.Sv0, s.Sv1 = sr/smv, sr/smv
	s.Rc0, s.Rc1 = rc, rc
	s.Ra0, s.Ra1 = 0.0, 0.0
	s.Bp0, s.Bp1 = bp, bp

	s.Hf = hi
	if s.Hf > hfs {
		s.Hf = hfs
	}
	s.Wfilm = 0.0
	s.Wsa = wi + 0.1
	fillInto(s.T, ti)
	fillInto(s.W, wi)
	fillInto(s.S, 0.0)

	s.diffusivity(s.Bp0)
	s.Init = true
}

// diffusivity computes the bound water diffusivity at each node for the given
// barometric pressure (cal/cm3). It matches diffusivity in firelab/NFDRS4.
//
// Reference: Nelson, R.M. 2000. Prediction of diurnal change in 10-hr fuel
// stick moisture content. Canadian Journal of Forest Research 30:1071-1087.
func (s *Stick) diffusivity(bp float64) {
	for i := 0; i < s.Nodes; i++ {
		tk := s.T[i] + 273.2
		qv := 13550.0 - 10.22*tk
		cpv := 7.22 + 0.002374*tk + 2.67e-07*tk*tk
		dv := 0.22 * 3600.0 * (0.0242 / bp) * math.Pow(tk/273.2, 1.75)
		ps1 := 0.0000239 * math.Exp(20.58-(5205.0/tk))
		c1 := 0.1617 - 0.001419*s.T[i]
		c2 := 0.4657 + 0.003578*s.T[i]

		var wc, dhdm float64
		if s.W[i] < s.Wsa {
			wc = s.W[i]
			if c2 != 1.0 && s.Hf < 1.0 && c1 != 0.0 && c2 != 0.0 {
				dhdm = (1.0 - s.Hf) * math.Pow(-math.Log(1.0-s.Hf), 1.0-c2) / (c1 * c2)
			}
		} else {
			wc = s.Wsa
			if c2 != 1.0 && hfs < 1.0 && c1 != 0.0 && c2 != 0.0 {
				dhdm = (1.0 - hfs) * math.Pow(wsf, 1.0-c2) / (c1 * c2)
			}
		}
		daw := 1.3 - 0.64*wc
		svaw := 1.0 / daw
		vfaw := svaw * wc / (0.685 + svaw*wc)
		vfcw := (0.685 + svaw*wc) / ((1.0 / s.Density) + svaw*wc)
		rfcw := 1.0 - math.Sqrt(1.0-vfcw)
		fac := 1.0 / (rfcw * vfcw)
		con := 1.0 / (2.0 - vfaw)
		qw := 5040.0 * math.Exp(-14.0*wc)
		e := (qv + qw - cpv*tk) / 1.2

		dvpr := 18.0 * 0.016 * (1.0 - vfcw) * dv * ps1 * dhdm / (s.Density * 1.987 * tk)
		s.D[i] = dvpr + 3600.0*0.0985*con*fac*math.Exp(-e/(1.987*tk))
	}
}

// MoistureContent returns the mean moisture content of the radial profile. It
// uses Simpson's rule over the nodes, then adds the water film. This is the
// stick's fuel moisture, as meanMoisture in firelab/NFDRS4 defines it.
func (s *Stick) MoistureContent() firewx.Percent {
	wec := s.W[0]
	wei := s.Dx / (3.0 * s.Radius)
	for i := 1; i < s.Nodes-1; i += 2 {
		wea := 4.0 * s.W[i]
		web := 2.0 * s.W[i+1]
		if (i + 1) == (s.Nodes - 1) {
			web = s.W[s.Nodes-1]
		}
		wec += web + wea
	}
	wbr := wei * wec
	if wbr > s.Wmx {
		wbr = s.Wmx
	}
	wbr += s.Wfilm
	return firewx.Percent(wbr * 100.0)
}

// MedianRadialMoisture returns the median moisture content of the radial nodes,
// as a percentage. This is the dead fuel moisture that the National Fire Danger
// Rating System uses, as medianRadialMoisture in firelab/NFDRS4 defines it.
func (s *Stick) MedianRadialMoisture() firewx.Percent {
	sorted := make([]float64, len(s.W))
	copy(sorted, s.W)
	sort.Float64s(sorted)
	return firewx.Percent(sorted[len(sorted)/2] * 100.0)
}

// SurfaceMoisture returns the moisture content of the surface node (g/g as a
// percentage).
func (s *Stick) SurfaceMoisture() firewx.Percent {
	return firewx.Percent(s.W[0] * 100.0)
}

// SurfaceTemperature returns the temperature of the surface node. The National
// Fire Danger Rating System uses it as the fuel surface temperature for the
// ignition component.
func (s *Stick) SurfaceTemperature() firewx.Celsius {
	return firewx.Celsius(s.T[0])
}

// UpdateObs advances the stick from a firewx.Obs over the elapsed time. The
// observation must carry temperature, relative humidity, solar radiation, and
// precipitation; an absent value leaves the stick unchanged and returns false.
// The precipitation is the amount since the previous observation.
func (s *Stick) UpdateObs(o firewx.Obs, elapsed time.Duration) bool {
	t, okT := o.Temperature.Get()
	rh, okH := o.RelativeHumidity.Get()
	solar, okS := o.SolarRadiation.Get()
	rain, okR := o.Precipitation.Get()
	if !okT || !okH || !okS || !okR {
		return false
	}
	return s.Update(Weather{
		Elapsed:          elapsed,
		Temperature:      t,
		RelativeHumidity: rh,
		SolarRadiation:   solar,
		Rainfall:         rain,
		Pressure:         o.Pressure,
	})
}

// Update advances the stick by one weather observation. It returns false and
// leaves the stick unchanged if the observation is out of range. The
// precipitation is the amount since the previous observation.
//
// Reference: this is a port of DeadFuelMoisture::update in firelab/NFDRS4,
// after Nelson (2000) with the modifications of Bevins (2005).
func (s *Stick) Update(w Weather) bool {
	et := w.Elapsed.Hours()
	at := float64(w.Temperature)
	rh := float64(w.RelativeHumidity) / 100.0
	sW := float64(w.SolarRadiation)
	// Precipitation amount since the previous observation, in centimetres.
	ramt := float64(w.Rainfall) / 10.0
	bpr := defaultBarometricPressure
	if hpa, ok := w.Pressure.Get(); ok {
		bpr = float64(hpa) * seaLevelCalPerCm3 / seaLevelHpa
	}

	s.ensureScratch()

	s.Updates++
	s.Elapsed += et

	// Reject bad data. The caller should catch bad records before this call.
	if et < 0.0000027 {
		return false
	}
	if rh < 0.001 || rh > 1.0 {
		return false
	}
	if at < -60.0 || at > 60.0 {
		return false
	}
	if sW < 0.0 {
		sW = 0.0
	}
	if sW > 2000.0 {
		return false
	}

	// Save the previous observation values.
	s.Ta0 = s.Ta1
	s.Ha0 = s.Ha1
	s.Sv0 = s.Sv1
	s.Rc0 = s.Rc1
	s.Ra0 = s.Ra1
	s.Bp0 = s.Bp1

	// Save the current observation values.
	s.Ta1 = at
	s.Ha1 = rh
	s.Sv1 = sW / smv
	s.Rc1 = s.Rc0 + ramt // cumulative, so accessors stay consistent
	s.Bp1 = bpr
	s.Et = et

	// The precipitation is passed as an amount since the previous observation.
	s.Ra1 = ramt

	// If there is no precipitation, reset the precipitation duration timer.
	if s.Ra1 < 0.0001 {
		s.Rdur = 0.0
	}
	// Precipitation rate since the last observation, adjusted by Pi (cm/h).
	s.Pptrate = s.Ra1 / et / pi
	// Moisture computation time step (h).
	s.Mdt = et / float64(s.MSteps)
	s.Mdt2 = s.Mdt * 2.0
	// Nelson's "s" factor.
	s.Sf = 3600.0 * s.Mdt / (s.Dx2 * s.Density)
	// Bound water diffusivity time step (h).
	s.Ddt = et / float64(s.DSteps)
	// First hour runoff factor.
	rai0 := s.Mdt * s.Rai0 * (1.0 - math.Exp(-100.0*s.Pptrate))
	// Adjust the runoff factor when the humidity is dropping.
	if s.Ha1 < s.Ha0 {
		if s.RampRai0 {
			rai0 *= 1.0 - ((s.Ha0 - s.Ha1) / s.Ha0)
		} else {
			rai0 *= 0.15
		}
	}
	// Later hour runoff factor.
	rai1 := s.Mdt * s.Rai1 * s.Pptrate

	// State counter for this observation.
	var tstate [numStates]int

	// Next time to run the diffusivity computation.
	ddtNext := s.Ddt
	// Elapsed moisture computation time (h).
	tt := s.Mdt
	// Loop for each moisture time step between the environmental inputs. The
	// loop mirrors the C for statement in firelab/NFDRS4, whose post-expression
	// "tt = nstep*m_mdt, nstep++" sets tt from the current nstep, then
	// increments nstep.
	nstep := 1
	for tt <= et {
		tfract := tt / et
		ta := s.Ta0 + (s.Ta1-s.Ta0)*tfract
		ha := s.Ha0 + (s.Ha1-s.Ha0)*tfract
		sv := s.Sv0 + (s.Sv1-s.Sv0)*tfract
		bp := s.Bp0 + (s.Bp1-s.Bp0)*tfract
		fsc := sv / srf
		tka := ta + kelvin
		tdw := 5205.0 / ((5205.0 / tka) - math.Log(ha))
		tdp := tdw - kelvin

		var tsk, hr, sr float64
		if fsc < 0.000001 {
			tsk = tcn + kelvin
			hr = hrn
			sr = 0.0
		} else {
			tsk = tcd + kelvin
			hr = hrd
			sr = srf * fsc
		}
		psa := 0.0000239 * math.Exp(20.58-(5205.0/tka))
		pa := ha * psa
		psd := 0.0000239 * math.Exp(20.58-(5205.0/tdw))
		if s.Ra1 > 0.0001 {
			s.Rdur += s.Mdt
		} else {
			s.Rdur = 0.0
		}

		// Stick surface temperature and humidity.
		tfd := ta + (sr-hr*(ta-tsk+kelvin))/(hr+s.Hc)
		qv := 13550.0 - 10.22*(tfd+kelvin)
		hw := (s.Hwf * ap / 0.24) * qv / 18.0
		s.T[0] = tfd - (hw*(tfd-ta))/(hr+s.Hc+hw)

		qw := 5040.0 * math.Exp(-14.0*s.W[0])
		tkf := s.T[0] + kelvin
		gnu := 0.00439 + 0.00000177*math.Pow(338.76-tkf, 2.1237)

		c1 := 0.1617 - 0.001419*s.T[0]
		c2 := 0.4657 + 0.003578*s.T[0]
		s.Wsa = c1 * math.Pow(wsf, c2)
		wdiff := s.Wmax - s.Wsa
		if wdiff < 0.000001 {
			wdiff = 0.000001
		}
		ps1 := 0.0000239 * math.Exp(20.58-(5205.0/tkf))
		p1 := pa + ap*bp*(qv/(qv+qw))*(tka-tkf)
		if p1 < 0.000001 {
			p1 = 0.000001
		}

		s.Hf = p1 / ps1
		if s.Hf > hfs {
			s.Hf = hfs
		}
		hfLog := -math.Log(1.0 - s.Hf)
		s.Sem = c1 * math.Pow(hfLog, c2)

		// Stick surface moisture content.
		s.State = StateNone
		s.Wfilm = 0.0
		var aml, bi float64
		sNew := s.S[0]
		wNew := s.W[0]
		wOld := s.W[0]

		if s.Ra1 > 0.0 {
			// It is raining.
			if s.AllowRainstorm && s.Pptrate >= s.Stv {
				s.State = StateRainstorm
				s.Wfilm = s.Wfilmk
				wNew = s.Wmx
			} else {
				// It is rainfall.
				if s.Rdur < 1.0 || !s.AllowRainfall2 {
					s.State = StateRainfall1
					wNew = wOld + rai0
				} else {
					s.State = StateRainfall2
					wNew = wOld + rai1
				}
			}
			s.Wfilm = s.Wfilmk
			sNew = (wNew - s.Wsa) / wdiff
			s.T[0] = tfd
			s.Hf = hfs
		} else {
			// It is not raining.
			if wOld > s.Wsa {
				// The moisture content exceeds the fiber saturation point.
				p1 = ps1
				s.Hf = hfs
				aml = s.Amlf * (ps1 - psd) / bp
				if s.T[0] <= tdp && p1 > psd {
					aml = 0.0
				}
				wNew = wOld - aml*s.Mdt2
				if aml > 0.0 {
					wNew -= s.Mdt * s.Capf / gnu
				}
				if wNew > s.Wmx {
					wNew = s.Wmx
				}
				sNew = (wNew - s.Wsa) / wdiff
				if wNew > wOld {
					s.State = StateCondensation1
				} else if wNew == wOld {
					s.State = StateStagnation
				} else {
					s.State = StateEvaporation
				}
			} else if s.T[0] <= tdp {
				// The fuel temperature is below the dew point: condensation.
				s.State = StateCondensation2
				if p1 > psd {
					aml = 0.0
				} else {
					aml = s.Amlf * (p1 - psd) / bp
				}
				wNew = wOld - aml*s.Mdt2
				sNew = (wNew - s.Wsa) / wdiff
			} else {
				// The surface moisture is below the fiber saturation point and
				// the stick temperature is above the dew point.
				if wOld >= s.Sem {
					s.State = StateDesorption
					bi = s.Stcd * s.Dx / s.D[0]
				} else {
					s.State = StateAdsorption
					bi = s.Stca * s.Dx / s.D[0]
				}
				wNew = (s.W[1] + bi*s.Sem) / (1.0 + bi)
				sNew = 0.0
			}
		}

		// Store the new surface moisture and saturation.
		if wNew > s.Wmx {
			s.W[0] = s.Wmx
		} else {
			s.W[0] = wNew
		}
		if sNew < 0.0 {
			s.S[0] = 0.0
		} else {
			s.S[0] = sNew
		}
		tstate[s.State]++

		// Save the current nodal values for the propagation step.
		for i := 0; i < s.Nodes; i++ {
			s.twold[i] = s.W[i]
			s.tsold[i] = s.S[i]
			s.ttold[i] = s.T[i]
			s.tv[i] = thdiff * s.X[i]
			s.to[i] = s.D[i] * s.X[i]
		}

		// Propagate the moisture content changes.
		if s.State != StateStagnation {
			for i := 0; i < s.Nodes; i++ {
				s.tg[i] = 0.0
				svp := (s.W[i] - s.Wsa) / wdiff
				if svp >= sir && svp <= scr {
					ak := aks * (2.0*math.Sqrt(svp/scr) - 1.0)
					s.tg[i] = (ak / (gnu * wdiff)) * s.X[i] * s.Vf * math.Pow(scr/svp, 1.5)
				}
			}

			// Propagate the fiber saturation moisture content changes.
			for i := 1; i < s.Nodes-1; i++ {
				ae := s.tg[i+1] / s.Dx
				aw := s.tg[i-1] / s.Dx
				ar := s.X[i] * s.Dx / s.Mdt
				ap := ae + aw + ar
				s.S[i] = (ae*s.tsold[i+1] + aw*s.tsold[i-1] + ar*s.tsold[i]) / ap
				// Constrain to Sir, not 1.0, so the stick can leave saturation.
				if s.S[i] > sir {
					s.S[i] = sir
				}
				if s.S[i] < 0.0 {
					s.S[i] = 0.0
				}
			}
			s.S[s.Nodes-1] = s.S[s.Nodes-2]

			// Check for continuous liquid columns at every interior node.
			continuousLiquid := true
			for i := 1; i < s.Nodes-1; i++ {
				if s.S[i] < sir {
					continuousLiquid = false
					break
				}
			}

			if continuousLiquid {
				for i := 1; i < s.Nodes-1; i++ {
					s.W[i] = s.Wsa + s.S[i]*wdiff
					if s.W[i] > s.Wmx {
						s.W[i] = s.Wmx
					}
					if s.W[i] < 0.0 {
						s.W[i] = 0.0
					}
				}
			} else {
				for i := 1; i < s.Nodes-1; i++ {
					ae := s.to[i+1] / s.Dx
					aw := s.to[i-1] / s.Dx
					ar := s.X[i] * s.Dx / s.Mdt
					ap := ae + aw + ar
					s.W[i] = (ae*s.twold[i+1] + aw*s.twold[i-1] + ar*s.twold[i]) / ap
					if s.W[i] > s.Wmx {
						s.W[i] = s.Wmx
					}
					if s.W[i] < 0.0 {
						s.W[i] = 0.0
					}
				}
			}
			s.W[s.Nodes-1] = s.W[s.Nodes-2]
		}

		// Propagate the fuel temperature changes.
		for i := 1; i < s.Nodes-1; i++ {
			ae := s.tv[i+1] / s.Dx
			aw := s.tv[i-1] / s.Dx
			ar := s.X[i] * s.Dx / s.Mdt
			ap := ae + aw + ar
			s.T[i] = (ae*s.ttold[i+1] + aw*s.ttold[i-1] + ar*s.ttold[i]) / ap
			if s.T[i] > 71.0 {
				s.T[i] = 71.0
			}
		}
		s.T[s.Nodes-1] = s.T[s.Nodes-2]

		// Update the moisture diffusivity within half a time step.
		if (ddtNext - tt) < (0.5 * s.Mdt) {
			s.diffusivity(bp)
			ddtNext += s.Ddt
		}

		// Post-expression: set tt from the current nstep, then increment nstep.
		tt = float64(nstep) * s.Mdt
		nstep++
	}

	// Store the prevailing state, the most frequent state over the steps.
	s.State = StateNone
	maxCount := tstate[0]
	for i := 1; i < numStates; i++ {
		if tstate[i] > maxCount {
			s.State = State(i)
			maxCount = tstate[i]
		}
	}
	return true
}

// allocScratch allocates the scratch buffers to the node count.
func (s *Stick) allocScratch() {
	s.twold = make([]float64, s.Nodes)
	s.tsold = make([]float64, s.Nodes)
	s.ttold = make([]float64, s.Nodes)
	s.tv = make([]float64, s.Nodes)
	s.to = make([]float64, s.Nodes)
	s.tg = make([]float64, s.Nodes)
}

// ensureScratch allocates the scratch buffers if they are absent, for example
// after the stick is restored from JSON.
func (s *Stick) ensureScratch() {
	if len(s.twold) != s.Nodes {
		s.allocScratch()
	}
}

// fillSlice returns a new slice of length n with every element set to v.
func fillSlice(n int, v float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = v
	}
	return out
}

// fillInto sets every element of dst to v.
func fillInto(dst []float64, v float64) {
	for i := range dst {
		dst[i] = v
	}
}
