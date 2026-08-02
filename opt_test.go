package firewx

import (
	"encoding/json"
	"testing"
)

func TestOptJSONRoundTrip(t *testing.T) {
	type wrapper struct {
		A Opt[Celsius] `json:"a"`
		B Opt[Celsius] `json:"b"`
	}
	in := wrapper{A: Some(Celsius(21.5)), B: None[Celsius]()}

	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"a":21.5,"b":null}` {
		t.Fatalf("unexpected encoding: %s", b)
	}

	var out wrapper
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if v, ok := out.A.Get(); !ok || v != 21.5 {
		t.Errorf("A: got %v %v", v, ok)
	}
	if out.B.Valid() {
		t.Error("B should be absent after round trip")
	}
}
