package server

import (
	"net/http"
	"slices"
	"strings"
	"syscall"
	"testing"

	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/license"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/steam"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/upstream/upstreamtest"
)

func failureBody(failure *steam.Failure) string {
	return `{"ok":false,"error":"` + strings.ReplaceAll(failure.Message, `"`, `\"`) + `","errorRu":"` + strings.ReplaceAll(failure.MessageRu, `"`, `\"`) + `"}`
}

func TestSteamCallbackOwner(t *testing.T) {
	a, transport := newTestApp(t, steamAnswers(map[string]string{ownerID: ownsTheGame}))
	recorder := serve(a, http.MethodGet, steamCallbackTarget(ownerID, zipKey))
	assertRedirect(t, recorder, testPublicURL+"/download-page/"+zipKey)
	want := license.SetCookie(license.Issue(ownerID, testNow, []byte(testSecret)))
	if cookies := setCookies(recorder); !slices.Equal(cookies, []string{want}) {
		t.Fatalf("Set-Cookie %q, want %q", cookies, want)
	}
	assertRedirect(t, serve(a, http.MethodGet, steamCallbackTarget(ownerID, "")), testPublicURL+"/")
	assertRedirect(t, serve(a, http.MethodGet, steamCallbackTarget(ownerID, "../x y")), testPublicURL+"/x%20y")
	if calls := transport.Calls(); len(calls) != 4 {
		t.Fatalf("%d upstream calls, want 3 check_authentication and 1 GetOwnedGames", len(calls))
	}
}

func TestSteamCallbackRedirectsByOwnership(t *testing.T) {
	cases := map[string]struct {
		answer   string
		location string
	}{
		"other games only": {otherGamesOnly, testPublicURL + "/error/game-not-owned"},
		"game_count 0":     {`{"response":{"game_count":0}}`, testPublicURL + "/error/game-owned-unknown"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			a, _ := newTestApp(t, steamAnswers(map[string]string{nonOwnerID: c.answer}))
			recorder := serve(a, http.MethodGet, steamCallbackTarget(nonOwnerID, zipKey))
			assertRedirect(t, recorder, c.location)
			if len(setCookies(recorder)) != 0 {
				t.Errorf("Set-Cookie %q", setCookies(recorder))
			}
		})
	}
}

func TestSteamCallbackFailures(t *testing.T) {
	refusing := func(upstreamtest.Call) (int, string, error) { return http.StatusOK, "is_valid:false\n", nil }
	unreachable := func(upstreamtest.Call) (int, string, error) { return 0, "", syscall.ECONNREFUSED }
	cases := []struct {
		name    string
		respond upstreamtest.Respond
		target  string
		status  int
		failure *steam.Failure
	}{
		{"ownership unreadable", steamAnswers(nil), steamCallbackTarget(ownerID, zipKey), http.StatusBadGateway, steam.ErrOwnership},
		{"login canceled, no Steam call", upstreamtest.Unused(t), "/auth/steam/callback?openid.mode=cancel", http.StatusBadRequest, steam.ErrCallback},
		{"account id not 17 digits, no Steam call", upstreamtest.Unused(t), steamCallbackTarget("123", zipKey), http.StatusBadRequest, steam.ErrID},
		{"assertion refused by Steam", refusing, steamCallbackTarget(ownerID, zipKey), http.StatusUnauthorized, steam.ErrState},
		{"Steam unreachable", unreachable, steamCallbackTarget(ownerID, zipKey), http.StatusBadGateway, steam.ErrUnavailable},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, _ := newTestApp(t, c.respond)
			assertProblem(t, serve(a, http.MethodGet, c.target), c.status, failureBody(c.failure))
		})
	}
}
