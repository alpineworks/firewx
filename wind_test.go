package firewx

import "testing"

func TestAdjustWindHeight(t *testing.T) {
	// A 3 m backyard sensor over suburban roughness, corrected up to the
	// 10 m exposure the FWI system assumes, must report a higher speed.
	got := AdjustWindHeight(4.0, 3.0, HeightFWI, RoughnessSuburban)
	if got <= 4.0 {
		t.Errorf("correcting upward should increase speed, got %v", got)
	}
	// Identity when heights match.
	if got := AdjustWindHeight(4.0, 10, 10, RoughnessOpenGrass); got != 4.0 {
		t.Errorf("identity case: got %v", got)
	}
}
