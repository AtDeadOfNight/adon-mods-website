package compat

import (
	"net/http"
	"testing"
)

func TestCookie(t *testing.T) {
	cases := []struct {
		headers []string
		want    string
	}{
		{nil, ""},
		{[]string{"t"}, ""},
		{[]string{"t=a.b"}, "a.b"},
		{[]string{"x=1; t=first; t=second"}, "first"},
		{[]string{"  t = spaced\t;"}, "spaced"},
		{[]string{"junk; t=after-junk"}, "after-junk"},
		{[]string{"t; t=named-later"}, "named-later"},
		{[]string{"T=upper; t=lower"}, "lower"},
		{[]string{`t="quoted"`}, `"quoted"`},
		{[]string{"t=a=b"}, "a=b"},
		{[]string{"t=a%2Eb"}, "a.b"},
		{[]string{"t=a%252Eb"}, "a.b"},
		{[]string{"t=a%25252Eb"}, "a%2Eb"},
		{[]string{"t=%E0%A4%A"}, ""},
		{[]string{"t=%25E0%25A4%25A"}, ""},
		{[]string{"t=+x"}, "+x"},
		{[]string{"x=%E0%A4%A; t=fine"}, "fine"},
		{[]string{"a=1", "t=second-header"}, "second-header"},
		{[]string{"t=first-header", "t=second-header"}, "first-header"},
		{[]string{"t=", "t=second"}, ""},
	}
	for _, c := range cases {
		r := &http.Request{Header: http.Header{"Cookie": c.headers}}
		if got := Cookie(r, "t"); got != c.want {
			t.Errorf("Cookie(%q) = %q, want %q", c.headers, got, c.want)
		}
	}
}
