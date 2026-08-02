package simple

import (
	"encoding/json"
	"math"
	"testing"

	firewx "alpineworks.io/firewx"
)

// Validation sources for this package. The full references are in README.md.
//
//   - Chandler and Angstrom: the formulas are identical to the firebehavioR R
//     package (Ziegler et al. 2019; Sharples et al. 2009). A value here is
//     reproducible by that prior implementation. The R expression is given at
//     each test.
//   - Fosberg: original Fosberg (1978) piecewise equilibrium moisture content,
//     as recorded by the Fire Weather Indices Wiki (wikifire.wsl.ch). Note that
//     firebehavioR substitutes the Simard fuel moisture model, so its Fosberg
//     values differ.
//   - Hot-Dry-Windy: Srock et al. 2018 (Atmosphere 9(7):279), vapour pressure
//     deficit in hectopascals. firebehavioR uses kilopascals and reports a value
//     10 times smaller.
//   - KBDI: the corrected Keetch and Byram (1968, Res. Pap. SE-38) analytic
//     equation, with the constant 8.30 from Alexander (1990, Fire Management
//     Notes 51(4):23-25). firebehavioR uses a metric lookup-table variant, so it
//     is not a numeric oracle for this analytic form.
//   - Nesterov (1949; Groisman et al. 2007): the classical sum, temperature
//     times dew point depression, reset above 3 mm of precipitation.

func closeTo(t *testing.T, got, want, tol float64, what string) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s: got %v, want %v (tol %v)", what, got, want, tol)
	}
}

// --- Fosberg ------------------------------------------------------------

func TestFosbergGolden(t *testing.T) {
	// T=85F, RH=20%, wind=15mph. RH is in the 10-50 EMC band. Recomputed
	// independently against the Fosberg (1978) equations: 37.53.
	got := FosbergIndex(85, 20, 15)
	closeTo(t, float64(got), 37.53, 0.05, "FFWI(85F,20%,15mph)")
}

func TestFosbergHighHumidityBranch(t *testing.T) {
	// RH=70% exercises the EMC branch for RH>50. T=60F, wind=10mph -> 12.49.
	got := FosbergIndex(60, 70, 10)
	closeTo(t, float64(got), 12.49, 0.05, "FFWI(60F,70%,10mph) high-RH branch")
}

func TestFosbergSaturationClampsToZero(t *testing.T) {
	// A saturated, very cold input drives the equilibrium moisture content above
	// 30 percent. The clamp holds it at 30, where the moisture term is zero, so
	// the index is zero. This is the only path that exercises the m>30 clamp.
	if emc := fosbergEMC(-60, 100); emc <= 30 {
		t.Fatalf("test setup: expected EMC>30 to exercise the clamp, got %v", emc)
	}
	got := FosbergIndex(-60, 100, 10)
	closeTo(t, float64(got), 0, 1e-9, "FFWI at saturation clamps to zero")
}

func TestFosbergCalibration(t *testing.T) {
	// The index is calibrated so that near-zero fuel moisture in a 30 mph wind
	// approaches the extreme value 100. At T=110F/RH=0 the EMC is 0.032 (not
	// clamped), giving 99.77.
	got := FosbergIndex(110, 0, 30)
	closeTo(t, float64(got), 99.77, 0.1, "FFWI calibration near 100")
}

func TestFosbergMonotonic(t *testing.T) {
	if FosbergIndex(85, 15, 10) <= FosbergIndex(85, 40, 10) {
		t.Error("FFWI should rise as RH falls")
	}
	if FosbergIndex(85, 20, 20) <= FosbergIndex(85, 20, 10) {
		t.Error("FFWI should rise as wind rises")
	}
}

func TestFosbergFromObs(t *testing.T) {
	o := firewx.Obs{
		Temperature:      firewx.Some(firewx.Celsius(29.4)), // ~85F
		RelativeHumidity: firewx.Some(firewx.Percent(20)),
	}
	if FosbergFromObs(o).Valid() {
		t.Error("Fosberg should be absent without wind")
	}
	o.WindSpeed = firewx.Some(firewx.MetersPerSecond(6.7056)) // exactly 15 mph
	got, ok := FosbergFromObs(o).Get()
	if !ok {
		t.Fatal("Fosberg should be present with all inputs")
	}
	// The FromObs path must convert SI to the equation's units. 29.4C=84.92F,
	// 6.7056 m/s=15 mph; the value must track the pure function on those inputs.
	want := FosbergIndex(firewx.Celsius(29.4).Fahrenheit(), 20, firewx.MetersPerSecond(6.7056).MilesPerHour())
	closeTo(t, float64(got), float64(want), 1e-9, "FosbergFromObs conversion")
}

// --- Chandler -----------------------------------------------------------

func TestChandlerGolden(t *testing.T) {
	// T=25C, RH=40%. Identical to firebehavioR::fireIndex(temp=25, rh=40)$chandler
	// = (((110-1.373*40)-0.54*(10.20-25))*(124*10^(-0.0142*40)))/60 = 35.25.
	got := ChandlerIndex(25, 40)
	closeTo(t, float64(got), 35.25, 0.05, "CBI(25C,40%)")
}

func TestChandlerMonotonic(t *testing.T) {
	if ChandlerIndex(25, 15) <= ChandlerIndex(25, 60) {
		t.Error("CBI should rise as RH falls")
	}
	// The temperature term -0.54*(10.20-T) must raise CBI as temperature rises.
	if ChandlerIndex(35, 40) <= ChandlerIndex(15, 40) {
		t.Error("CBI should rise as temperature rises")
	}
}

func TestChandlerClassBoundaries(t *testing.T) {
	cases := []struct {
		v    Chandler
		want DangerClass
	}{
		{49, ClassLow}, {50, ClassModerate}, {74, ClassModerate},
		{75, ClassHigh}, {89, ClassHigh}, {90, ClassVeryHigh},
		{97.4, ClassVeryHigh}, {97.5, ClassExtreme},
	}
	for _, c := range cases {
		if got := c.v.Class(); got != c.want {
			t.Errorf("Chandler(%v).Class()=%v, want %v", c.v, got, c.want)
		}
	}
}

func TestChandlerFromObs(t *testing.T) {
	o := firewx.Obs{Temperature: firewx.Some(firewx.Celsius(25))}
	if ChandlerFromObs(o).Valid() {
		t.Error("Chandler should be absent without RH")
	}
	o.RelativeHumidity = firewx.Some(firewx.Percent(40))
	got, ok := ChandlerFromObs(o).Get()
	if !ok {
		t.Fatal("Chandler should be present with T and RH")
	}
	closeTo(t, float64(got), float64(ChandlerIndex(25, 40)), 1e-9, "ChandlerFromObs value")
}

// --- Angstrom -----------------------------------------------------------

func TestAngstromGolden(t *testing.T) {
	// Identical to firebehavioR::fireIndex(temp=25, rh=40)$angstrom
	// = 40/20 + (27-25)/10 = 2.2.
	got := AngstromIndex(25, 40)
	closeTo(t, float64(got), 2.2, 1e-9, "Angstrom(25C,40%)")
}

func TestAngstromInverted(t *testing.T) {
	// A lower index means greater danger, so hotter and drier lowers it.
	if AngstromIndex(35, 20) >= AngstromIndex(15, 80) {
		t.Error("Angstrom should fall as conditions worsen")
	}
}

func TestAngstromClassBoundaries(t *testing.T) {
	cases := []struct {
		v    Angstrom
		want DangerClass
	}{
		{4.1, ClassLow}, {4.0, ClassModerate}, {2.6, ClassModerate},
		{2.5, ClassHigh}, {2.1, ClassHigh}, {2.0, ClassVeryHigh}, {0.7, ClassVeryHigh},
	}
	for _, c := range cases {
		if got := c.v.Class(); got != c.want {
			t.Errorf("Angstrom(%v).Class()=%v, want %v", c.v, got, c.want)
		}
	}
}

func TestAngstromFromObs(t *testing.T) {
	o := firewx.Obs{RelativeHumidity: firewx.Some(firewx.Percent(40))}
	if AngstromFromObs(o).Valid() {
		t.Error("Angstrom should be absent without temperature")
	}
	o.Temperature = firewx.Some(firewx.Celsius(25))
	got, ok := AngstromFromObs(o).Get()
	if !ok {
		t.Fatal("Angstrom should be present with T and RH")
	}
	closeTo(t, float64(got), 2.2, 1e-9, "AngstromFromObs value")
}

// --- Hot-Dry-Windy ------------------------------------------------------

func TestHDWGolden(t *testing.T) {
	// T=25C, RH=40%, wind=5 m/s. VPD ~19.01 hPa, product ~95.04.
	got := HDWIndex(firewx.Hectopascals(19.00893), 5)
	closeTo(t, float64(got), 95.04, 0.05, "HDW pure")

	o := firewx.Obs{
		Temperature:      firewx.Some(firewx.Celsius(25)),
		RelativeHumidity: firewx.Some(firewx.Percent(40)),
		WindSpeed:        firewx.Some(firewx.MetersPerSecond(5)),
	}
	fromObs, ok := HDWFromObs(o).Get()
	if !ok {
		t.Fatal("HDW should be present")
	}
	closeTo(t, float64(fromObs), 95.04, 0.05, "HDW from obs")
}

func TestHDWMonotonicAndZero(t *testing.T) {
	if HDWIndex(0, 5) != 0 {
		t.Error("HDW must be zero when VPD is zero")
	}
	if HDWIndex(19, 10) <= HDWIndex(19, 5) {
		t.Error("HDW should rise with wind")
	}
	if HDWIndex(30, 5) <= HDWIndex(19, 5) {
		t.Error("HDW should rise with VPD")
	}
}

func TestHDWAbsentWithoutHumidity(t *testing.T) {
	o := firewx.Obs{
		Temperature: firewx.Some(firewx.Celsius(25)),
		WindSpeed:   firewx.Some(firewx.MetersPerSecond(5)),
	}
	if HDWFromObs(o).Valid() {
		t.Error("HDW should be absent without the humidity needed for VPD")
	}
}

// --- Nesterov -----------------------------------------------------------

func TestNesterovAccumulatesAndResets(t *testing.T) {
	s := NewNesterovState()
	s.Step(25, 10, 0) // 25*(25-10) = 375
	closeTo(t, float64(s.Index), 375, 1e-9, "Nesterov day 1")
	s.Step(30, 12, 1) // +30*(30-12) = 540, precip below reset
	closeTo(t, float64(s.Index), 915, 1e-9, "Nesterov day 2")
	if s.Index.Class() != ClassModerate {
		t.Errorf("Nesterov 915 should classify Moderate, got %v", s.Index.Class())
	}
	s.Step(28, 11, 5) // precip > 3mm resets to zero
	closeTo(t, float64(s.Index), 0, 1e-9, "Nesterov reset on rain")
}

func TestNesterovResetBoundary(t *testing.T) {
	// Precipitation of exactly 3 mm is at the limit, not above it, so it must
	// NOT reset. This pins the > versus >= choice.
	s := NewNesterovState()
	s.Step(25, 10, 0) // 375
	s.Step(20, 10, 3) // +20*(20-10)=200, no reset
	closeTo(t, float64(s.Index), 575, 1e-9, "Nesterov precip=3 does not reset")
}

func TestNesterovDailyDewAboveTemp(t *testing.T) {
	// When the dew point is at or above the temperature (near saturation) the
	// daily term is zero or negative. The term is not floored.
	if got := NesterovDaily(20, 20); got != 0 {
		t.Errorf("NesterovDaily(20,20)=%v, want 0", got)
	}
	if got := NesterovDaily(20, 25); got != -100 {
		t.Errorf("NesterovDaily(20,25)=%v, want -100", got)
	}
}

func TestNesterovClassBoundaries(t *testing.T) {
	cases := []struct {
		v    Nesterov
		want DangerClass
	}{
		{299, ClassLow}, {300, ClassModerate}, {1000, ClassHigh},
		{4000, ClassVeryHigh}, {10000, ClassExtreme},
	}
	for _, c := range cases {
		if got := c.v.Class(); got != c.want {
			t.Errorf("Nesterov(%v).Class()=%v, want %v", c.v, got, c.want)
		}
	}
}

func TestNesterovStepObs(t *testing.T) {
	// Success path: dew point is derived from temperature and humidity.
	o := firewx.Obs{
		Temperature:      firewx.Some(firewx.Celsius(25)),
		RelativeHumidity: firewx.Some(firewx.Percent(40)),
		Precipitation:    firewx.Some(firewx.Millimeters(0)),
	}
	dew := o.DewPoint().Must()
	s := NewNesterovState()
	if !s.StepObs(o) {
		t.Fatal("StepObs should apply with T, RH and precip present")
	}
	closeTo(t, float64(s.Index), float64(NesterovDaily(25, dew)), 1e-9, "StepObs derived term")

	// Missing temperature must not advance the state.
	s2 := NewNesterovState()
	if s2.StepObs(firewx.Obs{
		RelativeHumidity: firewx.Some(firewx.Percent(40)),
		Precipitation:    firewx.Some(firewx.Millimeters(0)),
	}) {
		t.Error("StepObs should report false without temperature")
	}
}

func TestNesterovStepObsAbsentPrecip(t *testing.T) {
	s := NewNesterovState()
	// Missing precipitation must not advance the state, since assuming zero rain
	// could carry the index through a wet spell it should have reset on.
	o := firewx.Obs{
		Temperature:      firewx.Some(firewx.Celsius(25)),
		RelativeHumidity: firewx.Some(firewx.Percent(40)),
	}
	if s.StepObs(o) {
		t.Error("StepObs should report false without precipitation")
	}
	if s.Index != 0 {
		t.Error("state must be untouched when an input is absent")
	}
}

// --- KBDI ---------------------------------------------------------------

func TestKBDIIncrementGolden(t *testing.T) {
	// From a saturated soil (Q=0), one day at 90F with 50 in mean annual
	// rainfall raises the index by about 24.92 hundredths of an inch. Recomputed
	// from the corrected Keetch-Byram (1968) equation.
	s := NewKBDIState(50)
	s.Step(90, 0)
	closeTo(t, float64(s.Index), 24.92, 0.05, "KBDI increment from 0")
}

func TestKBDIRainInterceptionOncePerSpell(t *testing.T) {
	// 40F clamps the evapotranspiration term to zero, isolating the rain
	// bookkeeping: the 0.20 in interception is charged once per wet spell.
	s := NewKBDIState(50)
	s.Index = 400

	s.Step(40, 0.5) // net 0.30 -> -30
	closeTo(t, float64(s.Index), 370, 1e-9, "KBDI after first wet day")
	s.Step(40, 0.5) // consecutive: interception already spent, net 0.50 -> -50
	closeTo(t, float64(s.Index), 320, 1e-9, "KBDI after second wet day")
	s.Step(40, 0) // dry day ends the spell, no drought added
	closeTo(t, float64(s.Index), 320, 1e-9, "KBDI unchanged on dry cool day")
	s.Step(40, 0.1) // sub-threshold rain wets nothing
	closeTo(t, float64(s.Index), 320, 1e-9, "KBDI unchanged on sub-threshold rain")
}

func TestKBDIStaysInRange(t *testing.T) {
	// Heavy rain floors at zero.
	s := NewKBDIState(50)
	s.Index = 100
	s.Step(40, 2.0) // net 1.8 -> -180, floored at 0
	closeTo(t, float64(s.Index), 0, 1e-9, "KBDI floored at zero")

	// At the ceiling the (800-Q) factor is zero, so a hot day adds nothing and
	// the index holds at 800. This exercises the upper bound directly.
	s.Index = KBDIMax
	s.Step(110, 0)
	closeTo(t, float64(s.Index), float64(KBDIMax), 1e-9, "KBDI holds at 800")

	// Sustained heat drives the index up toward, but never past, 800.
	s.Index = 790
	for range 60 {
		s.Step(110, 0)
	}
	if s.Index > KBDIMax {
		t.Errorf("KBDI exceeded max: %v", s.Index)
	}
	if s.Index <= 790 {
		t.Errorf("KBDI should climb under sustained heat, got %v", s.Index)
	}
}

func TestKBDIClassBoundaries(t *testing.T) {
	cases := []struct {
		v    KBDI
		want DangerClass
	}{
		{199, ClassLow}, {200, ClassModerate}, {399, ClassModerate},
		{400, ClassHigh}, {599, ClassHigh}, {600, ClassExtreme}, {800, ClassExtreme},
	}
	for _, c := range cases {
		if got := c.v.Class(); got != c.want {
			t.Errorf("KBDI(%v).Class()=%v, want %v", c.v, got, c.want)
		}
	}
}

func TestKBDIStepObs(t *testing.T) {
	// Success path: 90F, no rain, from a saturated soil.
	s := NewKBDIState(50)
	o := firewx.Obs{
		Temperature:   firewx.Some(firewx.Celsius(32.2222)), // 90F
		Precipitation: firewx.Some(firewx.Millimeters(0)),
	}
	if !s.StepObs(o) {
		t.Fatal("StepObs should apply with temperature and precip present")
	}
	closeTo(t, float64(s.Index), 24.92, 0.05, "KBDI StepObs increment")

	// Absent rain must not advance the state.
	s2 := NewKBDIState(50)
	s2.Index = 300
	if s2.StepObs(firewx.Obs{Temperature: firewx.Some(firewx.Celsius(30))}) {
		t.Error("StepObs should report false without precipitation")
	}
	if s2.Index != 300 {
		t.Error("state must be untouched when precipitation is absent")
	}
}

// --- DangerClass --------------------------------------------------------

func TestDangerClassString(t *testing.T) {
	cases := map[DangerClass]string{
		ClassLow: "low", ClassModerate: "moderate", ClassHigh: "high",
		ClassVeryHigh: "very high", ClassExtreme: "extreme", DangerClass(99): "unknown",
	}
	for c, want := range cases {
		if got := c.String(); got != want {
			t.Errorf("DangerClass(%d).String()=%q, want %q", int(c), got, want)
		}
	}
}

// --- State persistence --------------------------------------------------

func TestStateJSONRoundTrip(t *testing.T) {
	// Property: any state value survives a JSON round trip byte for byte, which
	// the drivers rely on for persistence between daily runs.
	nesterovs := []NesterovState{
		NewNesterovState(),
		{SchemaVersion: schemaVersion, Index: 375.5},
		{SchemaVersion: schemaVersion, Index: -12.25},
		{SchemaVersion: schemaVersion, Index: 12345.678},
	}
	for _, n := range nesterovs {
		var back NesterovState
		b, err := json.Marshal(n)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatal(err)
		}
		if back != n {
			t.Errorf("NesterovState round trip: got %+v, want %+v", back, n)
		}
	}

	kbdis := []KBDIState{
		NewKBDIState(50),
		{SchemaVersion: schemaVersion, Index: 412.5, MeanAnnualRain: 33.75, WetSpellRain: 0.15},
		{SchemaVersion: schemaVersion, Index: 800, MeanAnnualRain: 60.125, WetSpellRain: 1.2},
	}
	for _, k := range kbdis {
		var back KBDIState
		b, err := json.Marshal(k)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatal(err)
		}
		if back != k {
			t.Errorf("KBDIState round trip: got %+v, want %+v", back, k)
		}
	}
}

func TestStateSchemaVersionIsReadable(t *testing.T) {
	// A state written by an older binary carries a different schema version. The
	// field must decode faithfully so a caller can detect the mismatch.
	var n NesterovState
	if err := json.Unmarshal([]byte(`{"schema_version":0,"index":5}`), &n); err != nil {
		t.Fatal(err)
	}
	if n.SchemaVersion != 0 {
		t.Errorf("schema version should decode to 0, got %d", n.SchemaVersion)
	}
	if NewNesterovState().SchemaVersion != schemaVersion {
		t.Errorf("constructor should stamp schema version %d", schemaVersion)
	}
}
