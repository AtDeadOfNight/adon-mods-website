package compat

import (
	"fmt"
	"strings"
)

// RoutePath resolves the request path like the WHATWG URL parser in front of Elysia did: a
// backslash is a slash, dot segments go, nothing is percent-decoded. Elysia matched both
// spellings of a trailing slash, so one is dropped.
func RoutePath(path string) string {
	segments := pathSegments(strings.TrimPrefix(strings.ReplaceAll(path, `\`, "/"), "/"), "/")
	normalized := "/" + strings.Join(segments, "/")
	if len(normalized) > 1 {
		normalized = strings.TrimSuffix(normalized, "/")
	}
	return normalized
}

func pathSegments(path, separators string) []string {
	var segments []string
	for {
		end := strings.IndexAny(path, separators)
		last := end < 0
		segment := path
		if !last {
			segment, path = path[:end], path[end+1:]
		}
		switch strings.ToLower(segment) {
		case "..", ".%2e", "%2e.", "%2e%2e":
			if len(segments) > 0 {
				segments = segments[:len(segments)-1]
			}
			if last {
				segments = append(segments, "")
			}
		case ".", "%2e":
			if last {
				segments = append(segments, "")
			}
		default:
			segments = append(segments, segment)
		}
		if last {
			return segments
		}
	}
}

const (
	fragmentEscapes = " \"<>`"
	queryEscapes    = " \"#<>'"
	pathEscapes     = " \"#<>?^`{}"
)

// SerializeURL is what Response.redirect sent for an absolute http(s) URL with a plain host: the
// WHATWG parser drops tabs and newlines and trims C0 controls and spaces, resolves dot segments,
// reads a backslash in the path as a slash, and percent-encodes each part with its own set.
func SerializeURL(raw string) string {
	raw = strings.TrimFunc(raw, func(r rune) bool { return r <= ' ' })
	raw = strings.NewReplacer("\t", "", "\n", "", "\r", "").Replace(raw)
	scheme, rest, _ := strings.Cut(raw, "://")
	scheme = strings.ToLower(scheme)
	authorityEnd := strings.IndexAny(rest, `/\?#`)
	if authorityEnd < 0 {
		authorityEnd = len(rest)
	}
	authority, rest := strings.ToLower(rest[:authorityEnd]), rest[authorityEnd:]
	defaultPort := map[string]string{"http": ":80", "https": ":443"}[scheme]
	authority = strings.TrimSuffix(authority, defaultPort)

	path, query, fragment, hasQuery, hasFragment := splitURL(rest)
	var serialized strings.Builder
	serialized.WriteString(scheme + "://" + authority)
	for _, segment := range pathSegments(path[min(1, len(path)):], `/\`) {
		serialized.WriteString("/" + escapeBytes(segment, pathEscapes))
	}
	if hasQuery {
		serialized.WriteString("?" + escapeBytes(query, queryEscapes))
	}
	if hasFragment {
		serialized.WriteString("#" + escapeBytes(fragment, fragmentEscapes))
	}
	return serialized.String()
}

func splitURL(rest string) (path, query, fragment string, hasQuery, hasFragment bool) {
	beforeFragment, fragment, hasFragment := strings.Cut(rest, "#")
	path, query, hasQuery = strings.Cut(beforeFragment, "?")
	return path, query, fragment, hasQuery, hasFragment
}

func escapeBytes(text, escapes string) string {
	var escaped strings.Builder
	for i := range len(text) {
		c := text[i]
		if c < 0x20 || c > 0x7E || strings.IndexByte(escapes, c) >= 0 {
			fmt.Fprintf(&escaped, "%%%02X", c)
			continue
		}
		escaped.WriteByte(c)
	}
	return escaped.String()
}
