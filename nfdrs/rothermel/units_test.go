package rothermel

import "testing"

// TestFuelLoadConversions checks the round trip between pounds per square foot
// and tons per acre. One ton per acre is 2000/43560 lb/ft2.
func TestFuelLoadConversions(t *testing.T) {
	cases := []struct {
		tonsPerAcre float64
		load        FuelLoad
	}{
		{1.0, FuelLoad(2000.0 / 43560.0)},
		{2.0, FuelLoad(0.0918273645546373)},
		{0.0, 0.0},
	}
	for _, tc := range cases {
		got := FuelLoadFromTonsPerAcre(tc.tonsPerAcre)
		closeTo(t, float64(got), float64(tc.load), 1e-12, "FuelLoadFromTonsPerAcre")
		closeTo(t, got.TonsPerAcre(), tc.tonsPerAcre, 1e-12, "TonsPerAcre round trip")
	}
}

// TestChainsPerHour checks the rate of spread conversion. One chain is 66 ft, so
// one chain per hour is 1.1 ft/min.
func TestChainsPerHour(t *testing.T) {
	// 1.1 ft/min is 1 chain per hour.
	closeTo(t, FeetPerMinute(1.1).ChainsPerHour(), 1.0, 1e-12, "1.1 ft/min to ch/h")
	// 11 ft/min is 10 chains per hour.
	closeTo(t, FeetPerMinute(11.0).ChainsPerHour(), 10.0, 1e-12, "11 ft/min to ch/h")
	closeTo(t, FeetPerMinute(0).ChainsPerHour(), 0.0, 1e-12, "zero")
}
