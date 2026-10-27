package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/compat"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/config"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/upstream/upstreamtest"
)

const (
	testSecret        = "test-secret-for-parity"
	testPublicURL     = "https://mods.adon.hloth.dev"
	testCallbackURL   = testPublicURL + "/auth/steam/callback"
	testUpstream      = "http://upstream.test"
	ownerID           = "76561197960010001"
	nonOwnerID        = "76561197960030001"
	steamFailsForID   = "76561197960070001"
	openIDValidAnswer = "ns:http://specs.openid.net/auth/2.0\nis_valid:true\n"
	ownsTheGame       = `{"response":{"game_count":1,"games":[{"appid":1450830,"playtime_forever":42}]}}`
	otherGamesOnly    = `{"response":{"game_count":1,"games":[{"appid":730}]}}`
	zipKey            = "AtDeadOfNight.app"
	zipName           = "At Dead Of Night.zip"
)

var testNow = time.Unix(1793105599, 0)

func testConfig(t *testing.T) config.Config {
	t.Helper()
	env := map[string]string{
		"STEAM_API_KEY": "test-steam-key",
		"JWT_SECRET":    testSecret,
		"PUBLIC_URL":    testPublicURL,
		"DOWNLOADS_DIR": t.TempDir(),
	}
	cfg, err := config.Load(func(name string) string { return env[name] })
	if err != nil {
		t.Fatal(err)
	}
	cfg.Origins = config.Origins{SteamCommunity: testUpstream, SteamAPI: testUpstream}
	return cfg
}

func fixedClock() time.Time {
	return testNow
}

func newTestApp(t *testing.T, respond upstreamtest.Respond) (*App, *upstreamtest.Transport) {
	t.Helper()
	client, transport := upstreamtest.NewClient(respond)
	return New(testConfig(t), client, fixedClock), transport
}

// steamAnswers confirms every assertion; GetOwnedGames fails for a Steam ID it has no answer for.
func steamAnswers(ownedGamesByID map[string]string) upstreamtest.Respond {
	return func(call upstreamtest.Call) (int, string, error) {
		if call.Path == "/openid/login" {
			return http.StatusOK, openIDValidAnswer, nil
		}
		for id, body := range ownedGamesByID {
			if strings.Contains(call.RawQuery, id) {
				return http.StatusOK, body, nil
			}
		}
		return http.StatusInternalServerError, "<html>", nil
	}
}

func putFile(t *testing.T, a *App, name string, content []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(a.cfg.DownloadsDir, name), content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func serve(handler http.Handler, method, target string, headers ...string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, nil)
	for i := 0; i+1 < len(headers); i += 2 {
		request.Header.Add(headers[i], headers[i+1])
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

// signToken signs like jsonwebtoken, apart from the license package under test.
func signToken(payload string) string {
	input := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." + base64.RawURLEncoding.EncodeToString([]byte(payload))
	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write([]byte(input))
	return input + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func validToken(steamID string) string {
	return signToken(`{"steamId":"` + steamID + `","iat":1700000000,"exp":4102444800}`)
}

func tokenCookie(token string) string {
	return "adon-mods-token=" + token
}

func steamCallbackTarget(steamID, file string) string {
	claimedID := "https://steamcommunity.com/openid/id/" + steamID
	returnTo, target := testCallbackURL, "/auth/steam/callback?"
	if file != "" {
		fileQuery := compat.EncodeForm("file", file)
		returnTo += "?" + fileQuery
		target += fileQuery + "&"
	}
	return target + compat.EncodeForm(
		"openid.ns", "http://specs.openid.net/auth/2.0",
		"openid.mode", "id_res",
		"openid.op_endpoint", "https://steamcommunity.com/openid/login",
		"openid.claimed_id", claimedID,
		"openid.identity", claimedID,
		"openid.return_to", returnTo,
		"openid.response_nonce", "2026-10-27T12:00:00Zparity",
		"openid.assoc_handle", "1234567890",
		"openid.signed", "signed,op_endpoint,claimed_id,identity,return_to,response_nonce,assoc_handle",
		"openid.sig", "Kq3+/parity=",
	)
}

func assertRedirect(t *testing.T, recorder *httptest.ResponseRecorder, location string) {
	t.Helper()
	if recorder.Code != http.StatusFound {
		t.Fatalf("status %d, want 302 (body %q)", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Location"); got != location {
		t.Fatalf("Location %q, want %q", got, location)
	}
	if recorder.Body.Len() != 0 || recorder.Header().Get("Content-Length") != "0" || recorder.Header().Get("Content-Type") != "" {
		t.Fatalf("redirect has a body: %q (%v)", recorder.Body.String(), recorder.Header())
	}
}

func assertProblem(t *testing.T, recorder *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if recorder.Code != status || recorder.Body.String() != body {
		t.Fatalf("got %d %q, want %d %q", recorder.Code, recorder.Body.String(), status, body)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type %q", got)
	}
}

func setCookies(recorder *httptest.ResponseRecorder) []string {
	return recorder.Header().Values("Set-Cookie")
}
