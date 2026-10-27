package compat

import "testing"

func TestRoutePath(t *testing.T) {
	cases := map[string]string{
		"/download":             "/download",
		"/download/":            "/download",
		"//download":            "//download",
		"/DOWNLOAD":             "/DOWNLOAD",
		"/d%6Fwnload":           "/d%6Fwnload",
		"/auth/../download":     "/download",
		"/auth/%2E%2e/download": "/download",
		`/auth\steam`:           "/auth/steam",
		"/auth/steam/.":         "/auth/steam",
		"/auth/steam/..":        "/auth",
		"/..":                   "/",
		"/":                     "/",
	}
	for path, want := range cases {
		if got := RoutePath(path); got != want {
			t.Errorf("RoutePath(%q) = %q, want %q", path, got, want)
		}
	}
}

// The expected values are the TypeScript backend's Location headers in the parity golden.
func TestSerializeURL(t *testing.T) {
	const page = "https://mods.adon.hloth.dev/purchase-to-download/"
	cases := map[string]string{
		"https://mods.adon.hloth.dev":       "https://mods.adon.hloth.dev/",
		"HTTPS://Mods.Adon.Hloth.Dev:443/x": "https://mods.adon.hloth.dev/x",
		page + "a b":                        page + "a%20b",
		page + "a#frag":                     page + "a#frag",
		page + "a?x=1":                      page + "a?x=1",
		page + "../../evil":                 "https://mods.adon.hloth.dev/evil",
		page + "..":                         "https://mods.adon.hloth.dev/",
		page + ".":                          page,
		page + "%2e%2E":                     "https://mods.adon.hloth.dev/",
		page + ".%2E/x":                     "https://mods.adon.hloth.dev/x",
		page + `a\b`:                        page + "a/b",
		page + "//evil.example":             page + "//evil.example",
		page + "мод":                        page + "%D0%BC%D0%BE%D0%B4",
		page + "%":                          page + "%",
		page + "a\tb\nc\rd":                 page + "abcd",
		page + "a\x01b\x7fc\x00":            page + "a%01b%7Fc",
		page + "a  ":                        page + "a",
		page + " a":                         page + "%20a",
		page + "a^b{}`\"<>|~!$&'()*,;=:@[]": page + "a%5Eb%7B%7D%60%22%3C%3E|~!$&'()*,;=:@[]",
		page + "a?b'c d\"<>`{}":             page + "a?b%27c%20d%22%3C%3E`{}",
		page + "a#b'c d\"<>`{}|":            page + "a#b'c%20d%22%3C%3E%60{}|",
		page + `a?b\c#d\e`:                  page + `a?b\c#d\e`,
		page + "a?q#f?#g":                   page + "a?q#f?#g",
	}
	for raw, want := range cases {
		if got := SerializeURL(raw); got != want {
			t.Errorf("SerializeURL(%q) = %q, want %q", raw, got, want)
		}
	}
}
