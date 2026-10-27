package steam

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"testing"
	"time"

	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/config"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/license"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/upstream/upstreamtest"
)

func testConfig() config.Config {
	return config.Config{
		SteamAPIKey:      "test-steam-key",
		PublicURL:        "https://mods.adon.hloth.dev",
		SteamCallbackURL: testCallback,
		Origins:          config.Origins{SteamCommunity: "https://steamcommunity.test", SteamAPI: "https://api.steampowered.test"},
	}
}

// The expected values are the TypeScript backend's Location headers in the parity golden.
func TestLoginURL(t *testing.T) {
	client := New(testConfig(), nil, time.Now)
	const prefix = "https://steamcommunity.com/openid/login?openid.ns=http://specs.openid.net/auth/2.0&openid.mode=checkid_setup&openid.return_to="
	const suffix = "&openid.realm=https://mods.adon.hloth.dev&openid.identity=http://specs.openid.net/auth/2.0/identifier_select&openid.claimed_id=http://specs.openid.net/auth/2.0/identifier_select"
	cases := map[string]string{
		"":                  "https%3A%2F%2Fmods.adon.hloth.dev%2Fauth%2Fsteam%2Fcallback",
		"AtDeadOfNight.app": "https%3A%2F%2Fmods.adon.hloth.dev%2Fauth%2Fsteam%2Fcallback%3Ffile%3DAtDeadOfNight.app",
		"*-._~!'()":         "https%3A%2F%2Fmods.adon.hloth.dev%2Fauth%2Fsteam%2Fcallback%3Ffile%3D*-._%257E%2521%2527%2528%2529",
		"a&b=c/?#%":         "https%3A%2F%2Fmods.adon.hloth.dev%2Fauth%2Fsteam%2Fcallback%3Ffile%3Da%2526b%253Dc%252F%253F%2523%2525",
		"мод":               "https%3A%2F%2Fmods.adon.hloth.dev%2Fauth%2Fsteam%2Fcallback%3Ffile%3D%25D0%25BC%25D0%25BE%25D0%25B4",
	}
	for file, returnTo := range cases {
		if got := client.LoginURL(file); got != prefix+returnTo+suffix {
			t.Errorf("LoginURL(%q)\ngot  %s\nwant %s", file, got, prefix+returnTo+suffix)
		}
	}
}

func TestVerify(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
		err    error
		want   error
	}{
		"confirmed":           {200, "ns:http://specs.openid.net/auth/2.0\nis_valid:true\n", nil, nil},
		"refused":             {200, "ns:http://specs.openid.net/auth/2.0\nis_valid:false\n", nil, ErrState},
		"confirmed, HTTP 500": {500, "is_valid:true\n", nil, ErrState},
		"unreachable":         {0, "", syscall.ECONNREFUSED, ErrUnavailable},
	}
	for name, c := range cases {
		httpClient, transport := upstreamtest.NewClient(func(upstreamtest.Call) (int, string, error) { return c.status, c.body, c.err })
		id, err := New(testConfig(), httpClient, time.Now).Verify(t.Context(), validQuery())
		if !errors.Is(err, c.want) || (err == nil) != (id == testID) {
			t.Errorf("%s: %q, %v", name, id, err)
		}
		calls := transport.Calls()
		if len(calls) != 1 || calls[0].Method != http.MethodPost || calls[0].Path != "/openid/login" || calls[0].ContentType != "application/x-www-form-urlencoded" {
			t.Fatalf("%s: calls %+v", name, calls)
		}
		form, _ := url.ParseQuery(calls[0].Body)
		if form.Get("openid.mode") != "check_authentication" || len(form) != 10 {
			t.Errorf("%s: form %v", name, form)
		}
	}

	httpClient, transport := upstreamtest.NewClient(upstreamtest.Unused(t))
	query := validQuery()
	query["openid.return_to"] = "https://evil.example/auth/steam/callback"
	if _, err := New(testConfig(), httpClient, time.Now).Verify(t.Context(), query); !errors.Is(err, ErrState) || len(transport.Calls()) != 0 {
		t.Fatalf("a foreign assertion: %v after %d calls", err, len(transport.Calls()))
	}
}

func TestOwnership(t *testing.T) {
	cases := map[string]struct {
		result license.Result
		ok     bool
	}{
		`{"response":{"game_count":1,"games":[{"appid":1450830,"playtime_forever":42}]}}`: {license.Owned, true},
		`{"response":{"games":[{"appid":730},{"appid":1450830.0}]}}`:                      {license.Owned, true},
		"\xef\xbb\xbf{\"response\":{\"games\":[{\"appid\":1450830e0}]}}":                  {license.Owned, true},
		`{"response":{"games":[{"appid":730}]}}`:                                          {license.NotOwned, true},
		`{"response":{"games":[]}}`:                                                       {license.NotOwned, true},
		`{"response":{"games":[{"appid":1e400}]}}`:                                        {license.NotOwned, true},
		`{"response":{"games":[{"appid":1450830,"appid":730}]}}`:                          {license.NotOwned, true},
		`{"response":{"game_count":0}}`:                                                   {license.Unknown, true},
		`{"response":{}}`:                                                                 {license.Unknown, true},
		`{"response":{"games":[{"appid":"1450830"}]}}`:                                    {license.Unknown, true},
		`{"response":{"games":[{"appid":1450830},null]}}`:                                 {license.Unknown, true},
		`{"response":[]}`:             {license.Unknown, true},
		`[{"response":{"games":[]}}]`: {license.Unknown, true},
		`null`:                        {license.Unknown, true},
		`<html>`:                      {license.Failed, false},
		`{"response":{}}x`:            {license.Failed, false},
		``:                            {license.Failed, false},
	}
	for body, want := range cases {
		if result, ok := ownership([]byte(body)); result != want.result || ok != want.ok {
			t.Errorf("ownership(%s) = %v, %v; want %v, %v", body, result, ok, want.result, want.ok)
		}
	}
}

func TestLicenseAsksOnceAndCaches(t *testing.T) {
	const (
		notOwnerID      = "76561197960030001"
		steamFailsForID = "76561197960070001"
	)
	httpClient, transport := upstreamtest.NewClient(func(call upstreamtest.Call) (int, string, error) {
		if strings.Contains(call.RawQuery, notOwnerID) {
			return 200, `{"response":{"games":[{"appid":730}]}}`, nil
		}
		return 500, "<html>", nil
	})
	client := New(testConfig(), httpClient, time.Now)
	for range 2 {
		if result := client.License(t.Context(), notOwnerID); result != license.NotOwned {
			t.Fatalf("result %v, want %v", result, license.NotOwned)
		}
	}
	for range 2 {
		if result := client.License(t.Context(), steamFailsForID); result != license.Failed {
			t.Fatalf("result %v, want %v", result, license.Failed)
		}
	}
	calls := transport.Calls()
	if len(calls) != 3 {
		t.Fatalf("%d calls, want one for the cached account and two for the failing one", len(calls))
	}
	wantQuery := "key=test-steam-key&input_json=" + url.QueryEscape(`{"steamid":"`+notOwnerID+`","appids_filter":[1450830],"include_appinfo":false,"include_played_free_games":false}`)
	if calls[0].Method != http.MethodGet || calls[0].Path != "/IPlayerService/GetOwnedGames/v1/" || calls[0].RawQuery != wantQuery {
		t.Fatalf("call %+v, want GET /IPlayerService/GetOwnedGames/v1/?%s", calls[0], wantQuery)
	}
}
