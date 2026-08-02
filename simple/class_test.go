package simple

import "testing"

func TestDangerClassString(t *testing.T) {
	cases := []struct {
		c    DangerClass
		want string
	}{
		{ClassLow, "low"},
		{ClassModerate, "moderate"},
		{ClassHigh, "high"},
		{ClassVeryHigh, "very high"},
		{ClassExtreme, "extreme"},
		{DangerClass(99), "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := tc.c.String(); got != tc.want {
				t.Errorf("DangerClass(%d).String()=%q, want %q", int(tc.c), got, tc.want)
			}
		})
	}
}
