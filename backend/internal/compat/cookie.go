package compat

import (
	"net/http"
	"strings"
)

// Cookie is what Elysia's cookie jar held: Bun joins every Cookie header with "; ", the cookie
// package keeps the first pair with a name and decodes it when it can, and Elysia decodes once
// more. A value that fails the second decode was null, which the old code treated like no cookie.
func Cookie(r *http.Request, name string) string {
	header := strings.Join(r.Header.Values("Cookie"), "; ")
	for pair := range strings.SplitSeq(header, ";") {
		key, raw, ok := strings.Cut(pair, "=")
		if !ok || strings.Trim(key, " \t") != name {
			continue
		}
		value, _ := DecodeURIComponent(decodeOrKeep(strings.Trim(raw, " \t")))
		return value
	}
	return ""
}
