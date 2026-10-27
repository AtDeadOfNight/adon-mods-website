package server

import (
	"net/http"
	"strings"
	"testing"

	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/upstream/upstreamtest"
)

const notFound = `{"ok":false,"error":"Not found"}`

func TestRouting(t *testing.T) {
	a, _ := newTestApp(t, upstreamtest.Unused(t))
	const steamLogin = "https://steamcommunity.com/openid/login?"
	for _, target := range []string{"/auth/steam", "/auth/steam/", `/auth\steam`, "/download/../auth/steam", "/auth/steam/.", "http://t/download/../auth/steam"} {
		recorder := serve(a, http.MethodGet, target)
		if recorder.Code != http.StatusFound || !strings.HasPrefix(recorder.Header().Get("Location"), steamLogin) {
			t.Errorf("GET %s: %d %q, want 302 to %s…", target, recorder.Code, recorder.Header().Get("Location"), steamLogin)
		}
	}
	if recorder := serve(a, http.MethodHead, "/auth/steam"); recorder.Code != http.StatusFound {
		t.Errorf("HEAD: %d", recorder.Code)
	}
	for _, target := range []string{"/", "/nope", "/logout", "/auth", "/auth/callback", "/api/download", "/DOWNLOAD", "//download", "/d%6Fwnload", "/auth/steam/callback/x"} {
		assertProblem(t, serve(a, http.MethodGet, target), http.StatusNotFound, notFound)
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions, "FOO", "get"} {
		assertProblem(t, serve(a, method, "/auth/steam"), http.StatusNotFound, notFound)
	}
}

func TestSteamLoginCarriesTheFile(t *testing.T) {
	a, _ := newTestApp(t, upstreamtest.Unused(t))
	cases := map[string]string{
		"/auth/steam":       "",
		"/auth/steam?file=": "",
		"/auth/steam?file=AtDeadOfNight.app&file=x": "AtDeadOfNight.app",
		"/auth/steam?file=a+b":                      "a b",
	}
	for target, file := range cases {
		assertRedirect(t, serve(a, http.MethodGet, target, "Cookie", "adon-mods-token=garbage"), a.steam.LoginURL(file))
	}
}
