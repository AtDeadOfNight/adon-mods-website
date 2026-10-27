package compat

import (
	"maps"
	"testing"
)

func TestParseQuery(t *testing.T) {
	cases := map[string]map[string]string{
		"":                             {},
		"file=a&file=b":                {"file": "a"},
		"file":                         {"file": ""},
		"file=":                        {"file": ""},
		"=x&&file=y":                   {"file": "y"},
		"a+b=c+d%20e":                  {"a b": "c d e"},
		"plus=%2B":                     {"plus": "+"},
		"fil%65=x":                     {"file": "x"},
		"bad=%E0%A4%A":                 {"bad": "%E0%A4%A"},
		"surrogate=%ED%A0%80":          {"surrogate": "%ED%A0%80"},
		"utf8=%D0%BC%D0%BE%D0%B4":      {"utf8": "мод"},
		"equals=a=b%3Dc":               {"equals": "a=b=c"},
		"sig=Kq3+/parity=#frag":        {"sig": "Kq3 /parity=#frag"},
		"file=a&fil%65=b":              {"file": "a"},
		"%zz=1":                        {"%zz": "1"},
		"openid.ns=x&openid%2Ens=y":    {"openid.ns": "x"},
		"k=%41%42&k2=%C3%A9%C3":        {"k": "AB", "k2": "%C3%A9%C3"},
		"lower=%c3%a9":                 {"lower": "é"},
		"nul=%00":                      {"nul": "\x00"},
		"space+key=1&space%20key=2&=3": {"space key": "1"},
	}
	for query, want := range cases {
		if got := ParseQuery(query); !maps.Equal(got, want) {
			t.Errorf("ParseQuery(%q) = %q, want %q", query, got, want)
		}
	}
}

func TestDecodeURIComponent(t *testing.T) {
	cases := map[string]struct {
		decoded string
		ok      bool
	}{
		"plain":        {"plain", true},
		"%41":          {"A", true},
		"%":            {"", false},
		"%4":           {"", false},
		"%G1":          {"", false},
		"%C0%80":       {"", false},
		"%F4%90%80%80": {"", false},
		"%E2%82%AC":    {"€", true},
		"%E2%82":       {"", false},
		"%E2%82x":      {"", false},
	}
	for encoded, want := range cases {
		decoded, ok := DecodeURIComponent(encoded)
		if decoded != want.decoded || ok != want.ok {
			t.Errorf("DecodeURIComponent(%q) = %q, %v; want %q, %v", encoded, decoded, ok, want.decoded, want.ok)
		}
	}
}

func TestEncoders(t *testing.T) {
	const wantURIComponent = "https%3A%2F%2Fx%2Fa%20b%3Fc%3Dd%26e~!*'()%C3%A9"
	if got := EncodeURIComponent("https://x/a b?c=d&e~!*'()é"); got != wantURIComponent {
		t.Errorf("EncodeURIComponent: got %s, want %s", got, wantURIComponent)
	}
	const wantForm = "file=a+b*-._%7E%21%27%28%29%C3%A9&z="
	if got := EncodeForm("file", "a b*-._~!'()é", "z", ""); got != wantForm {
		t.Errorf("EncodeForm: got %s, want %s", got, wantForm)
	}
}
