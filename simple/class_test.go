package simple

import "testing"

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
