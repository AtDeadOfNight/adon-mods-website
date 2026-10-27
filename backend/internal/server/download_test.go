package server

import (
	"bytes"
	"net/http"
	"slices"
	"testing"

	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/license"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/upstream/upstreamtest"
)

const invalidCookie = `{"ok":false,"error":"Invalid cookie. Please clear the website's cookies and sign in with Steam again."}`

func TestDownloadWithoutToken(t *testing.T) {
	a, _ := newTestApp(t, upstreamtest.Unused(t))
	cases := map[string]string{
		"/download?file=" + zipKey: testPublicURL + "/purchase-to-download/" + zipKey,
		"/download":                testPublicURL + "/purchase-to-download/undefined",
		"/download?file=a%20b%23c": testPublicURL + "/purchase-to-download/a%20b#c",
	}
	for target, location := range cases {
		for _, cookie := range []string{"", "adon-mods-token=", "adon-mods-token=%E0%A4%A", "other=1"} {
			recorder := serve(a, http.MethodGet, target, "Cookie", cookie)
			assertRedirect(t, recorder, location)
			if len(setCookies(recorder)) != 0 {
				t.Errorf("%s with %q: Set-Cookie %q", target, cookie, setCookies(recorder))
			}
		}
	}
}

func TestDownloadClearsAnUnusableToken(t *testing.T) {
	a, _ := newTestApp(t, upstreamtest.Unused(t))
	for _, cookie := range []string{
		"adon-mods-token=garbage",
		tokenCookie(signToken(`{"steamId":"` + ownerID + `","exp":1000000000}`)),
		tokenCookie(signToken(`{"steamId":123}`)),
		tokenCookie(validToken(ownerID) + "x"),
	} {
		recorder := serve(a, http.MethodGet, "/download?file="+zipKey, "Cookie", cookie)
		assertRedirect(t, recorder, testPublicURL+"/purchase-to-download/"+zipKey)
		if cookies := setCookies(recorder); !slices.Equal(cookies, []string{license.ClearCookie}) {
			t.Errorf("%s: Set-Cookie %q", cookie, cookies)
		}
	}
}

func TestDownloadAsksSteam(t *testing.T) {
	a, transport := newTestApp(t, steamAnswers(map[string]string{ownerID: ownsTheGame, nonOwnerID: otherGamesOnly}))
	putFile(t, a, zipName, []byte("zip bytes"))

	recorder := serve(a, http.MethodGet, "/download?file="+zipKey, "Cookie", tokenCookie(validToken(ownerID)))
	if recorder.Code != http.StatusOK || recorder.Body.String() != "zip bytes" {
		t.Fatalf("owner: %d %q", recorder.Code, recorder.Body.String())
	}
	if cookies := setCookies(recorder); !slices.Equal(cookies, []string{license.SetCookie(validToken(ownerID))}) {
		t.Fatalf("owner: Set-Cookie %q", cookies)
	}

	recorder = serve(a, http.MethodGet, "/download?file="+zipKey, "Cookie", tokenCookie(validToken(nonOwnerID)))
	assertRedirect(t, recorder, testPublicURL+"/purchase-to-download/"+zipKey)
	if cookies := setCookies(recorder); !slices.Equal(cookies, []string{license.ClearCookie}) {
		t.Fatalf("not owned: Set-Cookie %q", cookies)
	}

	recorder = serve(a, http.MethodGet, "/download?file="+zipKey, "Cookie", tokenCookie(validToken(steamFailsForID)))
	assertProblem(t, recorder, http.StatusBadRequest, invalidCookie)
	if len(setCookies(recorder)) != 0 {
		t.Fatalf("Steam failure: Set-Cookie %q", setCookies(recorder))
	}

	serve(a, http.MethodGet, "/download?file="+zipKey, "Cookie", tokenCookie(validToken(ownerID)))
	if calls := len(transport.Calls()); calls != 3 {
		t.Fatalf("%d GetOwnedGames calls, want 3: the owner is cached", calls)
	}
}

func TestDownloadAnswersAfterTheCheck(t *testing.T) {
	a, _ := newTestApp(t, steamAnswers(map[string]string{ownerID: ownsTheGame}))
	putFile(t, a, zipName, bytes.Repeat([]byte("0123456789"), 10))
	renewed := []string{license.SetCookie(validToken(ownerID))}
	cases := []struct {
		target string
		header []string
		status int
		body   string
	}{
		{"/download?file=nope", nil, http.StatusNotFound, `{"ok":false,"error":"File not found"}`},
		{"/download", nil, http.StatusNotFound, `{"ok":false,"error":"File not found"}`},
		{"/download?file=at-dead-of-night-ios-v1.0.0-t.me_adonru.ipa", nil, http.StatusInternalServerError, `{"ok":false,"error":"Internal server error"}`},
		{"/download?file=" + zipKey, []string{"Range", "bytes=-3"}, http.StatusPartialContent, "789"},
		{"/download?file=" + zipKey, []string{"Range", "bytes=100-"}, http.StatusRequestedRangeNotSatisfiable, `{"ok":false,"error":"Requested Range Not Satisfiable"}`},
	}
	for _, c := range cases {
		recorder := serve(a, http.MethodGet, c.target, append([]string{"Cookie", tokenCookie(validToken(ownerID))}, c.header...)...)
		if recorder.Code != c.status || recorder.Body.String() != c.body {
			t.Errorf("%s %v: %d %q, want %d %q", c.target, c.header, recorder.Code, recorder.Body.String(), c.status, c.body)
		}
		if !slices.Equal(setCookies(recorder), renewed) {
			t.Errorf("%s: Set-Cookie %q", c.target, setCookies(recorder))
		}
	}
}
