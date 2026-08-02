package firewx

import "testing"

func TestAdjustWindHeight(t *testing.T) {
	cases := []struct {
		name     string
		v        MetersPerSecond
		from, to Meters
		z0       Roughness
		want     float64 // exact expected value, used when greaterThanV is false
		// greaterThanV requires only that the corrected speed exceeds v, which is
		// what a correction up from a lower, sheltered sensor must produce.
		greaterThanV bool
	}{
		{"correcting upward increases speed", 4.0, 3.0, HeightFWI, RoughnessSuburban, 0, true},
		{"identity when heights match", 4.0, 10, 10, RoughnessOpenGrass, 4.0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := float64(AdjustWindHeight(tc.v, tc.from, tc.to, tc.z0))
			if tc.greaterThanV {
				if got <= float64(tc.v) {
					t.Errorf("%s: got %v, want > %v", tc.name, got, tc.v)
				}
				return
			}
			if got != tc.want {
				t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
