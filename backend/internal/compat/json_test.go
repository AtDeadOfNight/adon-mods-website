package compat

import (
	"encoding/json"
	"math"
	"testing"
)

func TestDecodeJSON(t *testing.T) {
	for _, valid := range []string{"\xef\xbb\xbf{}", " \n{} \n", `[1]`, `null`, `{"a":"\ud800"}`} {
		if _, err := DecodeJSON([]byte(valid)); err != nil {
			t.Errorf("%q: %v", valid, err)
		}
	}
	for _, invalid := range []string{"", "{}x", "{} {}", "\xef\xbb\xbf\xef\xbb\xbf{}", "<html>"} {
		if _, err := DecodeJSON([]byte(invalid)); err == nil {
			t.Errorf("%q was accepted", invalid)
		}
	}
}

func TestFieldAndEach(t *testing.T) {
	document, err := DecodeJSON([]byte(`{"response":{"games":[{"appid":1},{"appid":2.0}],"x":[1]}}`))
	if err != nil {
		t.Fatal(err)
	}
	ids, ok := Each(Field(document, "response", "games"), "appid", Number)
	if !ok || len(ids) != 2 || ids[1] != 2 {
		t.Fatalf("ids %v, %v", ids, ok)
	}
	if _, ok := Each(Field(document, "response", "x"), "appid", Number); ok {
		t.Fatal("a list of numbers is not a list of objects")
	}
	if Field(document, "response", "games", "appid") != nil || Field(nil, "a") != nil {
		t.Fatal("Field went through a non-object")
	}
}

func TestNumber(t *testing.T) {
	cases := map[string]float64{"1": 1, "1e400": math.Inf(1), "-1e400": math.Inf(-1), "1e-400": 0, "1450830.0": 1450830}
	for text, want := range cases {
		if got, ok := Number(json.Number(text)); !ok || got != want {
			t.Errorf("Number(%s) = %v, %v; want %v, true", text, got, ok, want)
		}
	}
	for _, value := range []any{"1", true, nil, json.Number("x")} {
		if _, ok := Number(value); ok {
			t.Errorf("Number(%#v) was accepted", value)
		}
	}
}
