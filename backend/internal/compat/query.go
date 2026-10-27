// Package compat reproduces the parsing and encoding rules of Bun, Elysia, the cookie package
// and the WHATWG URL standard that the old backend's answers depended on, odd corners included.
package compat

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

const alphanumeric = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// ParseQuery is Elysia's parseQueryFromURL: the first value of a key wins, '+' is a space even
// when escaped later, a malformed escape keeps its raw text, and an empty key is dropped.
func ParseQuery(query string) map[string]string {
	values := map[string]string{}
	for segment := range strings.SplitSeq(query, "&") {
		rawKey, rawValue, _ := strings.Cut(segment, "=")
		if rawKey == "" {
			continue
		}
		key := decodeQueryComponent(rawKey)
		if _, seen := values[key]; !seen {
			values[key] = decodeQueryComponent(rawValue)
		}
	}
	return values
}

func decodeQueryComponent(component string) string {
	return decodeOrKeep(strings.ReplaceAll(component, "+", " "))
}

func decodeOrKeep(encoded string) string {
	if decoded, ok := DecodeURIComponent(encoded); ok {
		return decoded
	}
	return encoded
}

// Each run of escapes must decode to valid UTF-8 on its own, as in JavaScript.
func DecodeURIComponent(encoded string) (string, bool) {
	if !strings.Contains(encoded, "%") {
		return encoded, true
	}
	var decoded, escaped []byte
	for i := 0; i < len(encoded); i++ {
		if encoded[i] != '%' {
			if !utf8.Valid(escaped) {
				return "", false
			}
			decoded = append(append(decoded, escaped...), encoded[i])
			escaped = escaped[:0]
			continue
		}
		if i+2 >= len(encoded) {
			return "", false
		}
		octet, err := strconv.ParseUint(encoded[i+1:i+3], 16, 8)
		if err != nil {
			return "", false
		}
		escaped = append(escaped, byte(octet))
		i += 2
	}
	if !utf8.Valid(escaped) {
		return "", false
	}
	return string(append(decoded, escaped...)), true
}

func EncodeURIComponent(text string) string {
	return percentEncode(text, alphanumeric+"-_.!~*'()", false)
}

// Same bytes as URLSearchParams.toString(); url.Values.Encode would sort the keys.
func EncodeForm(keysAndValues ...string) string {
	var form strings.Builder
	for i, text := range keysAndValues {
		switch {
		case i%2 == 1:
			form.WriteByte('=')
		case i > 0:
			form.WriteByte('&')
		}
		form.WriteString(percentEncode(text, alphanumeric+"*-._", true))
	}
	return form.String()
}

func percentEncode(text, unescaped string, spaceAsPlus bool) string {
	var encoded strings.Builder
	for i := range len(text) {
		switch c := text[i]; {
		case strings.IndexByte(unescaped, c) >= 0:
			encoded.WriteByte(c)
		case c == ' ' && spaceAsPlus:
			encoded.WriteByte('+')
		default:
			fmt.Fprintf(&encoded, "%%%02X", c)
		}
	}
	return encoded.String()
}
