package firewx

import (
	"encoding/json"
	"testing"
)

func TestOptJSONRoundTrip(t *testing.T) {
	cases := []struct {
		name     string
		opt      Opt[Celsius]
		wantJSON string
	}{
		{"present encodes as the bare value", Some(Celsius(21.5)), `21.5`},
		{"absent encodes as null", None[Celsius](), `null`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.opt)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tc.wantJSON {
				t.Fatalf("encoding: got %s, want %s", b, tc.wantJSON)
			}
			var back Opt[Celsius]
			if err := json.Unmarshal(b, &back); err != nil {
				t.Fatal(err)
			}
			if back != tc.opt {
				t.Errorf("round trip: got %+v, want %+v", back, tc.opt)
			}
		})
	}
}
