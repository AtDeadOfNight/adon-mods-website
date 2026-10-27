package steam

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"slices"
	"time"

	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/compat"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/config"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/license"
	"git.hloth.dev/at-dead-of-night/mods.adon.hloth.dev/backend/internal/upstream"
)

const atDeadOfNightAppID = 1450830

type Failure struct {
	Status    int
	Message   string
	MessageRu string
}

func (f *Failure) Error() string {
	return f.Message
}

var (
	ErrCallback = &Failure{
		Status:    http.StatusBadRequest,
		Message:   "Invalid authorization callback. Might be Steam auth issue. Try again in 10-30 minutes",
		MessageRu: "Некорректный коллбек авторизации. Возможно, ошибка на стороне Steam. Попробуйте через 10-30 минут.",
	}
	ErrState = &Failure{
		Status:    http.StatusUnauthorized,
		Message:   `Invalid state. You probably pressed "back" button in your browser.`,
		MessageRu: `Невалидное состояние авторизации. Скорее всего, вы нажали "назад" в браузере.`,
	}
	ErrID = &Failure{
		Status:    http.StatusBadRequest,
		Message:   "Failed to extract SteamID from response. This is a Steam-side issue with authorization. Please wait 10-30 minutes and try again.",
		MessageRu: "Не удалось извлечь SteamID из ответа. Это ошибка на стороне Steam. Пожалуйста, подождите 10-30 минут и попробуйте снова.",
	}
	ErrUnavailable = &Failure{
		Status:    http.StatusBadGateway,
		Message:   "Steam API error. This is a Steam-side issue with authorization. Please wait 10-30 minutes and try again.",
		MessageRu: "Ошибка API авторизации Steam. Это проблема на стороне Steam. Пожалуйста, подождите 10-30 минут и попробуйте снова.",
	}
	ErrOwnership = &Failure{
		Status:    http.StatusBadGateway,
		Message:   "Failed to verify At Dead of Night ownership. Might be Steam-side issue.",
		MessageRu: "Не удалось подтвердить наличие At Dead of Night в библиотеке. Возможно, это ошибка на стороне Steam.",
	}
)

type Client struct {
	cfg      config.Config
	http     *http.Client
	licenses *license.Cache[string]
}

func New(cfg config.Config, client *http.Client, now func() time.Time) *Client {
	return &Client{cfg: cfg, http: client, licenses: license.NewCache[string](now)}
}

// LoginURL is the old template string, where only return_to was encoded.
func (c *Client) LoginURL(file string) string {
	returnTo := c.cfg.SteamCallbackURL
	if file != "" {
		returnTo += "?" + compat.EncodeForm("file", file)
	}
	return compat.SerializeURL(openIDEndpoint +
		"?openid.ns=" + openIDNamespace +
		"&openid.mode=checkid_setup" +
		"&openid.return_to=" + compat.EncodeURIComponent(returnTo) +
		"&openid.realm=" + c.cfg.PublicURL +
		"&openid.identity=" + identifierSelect +
		"&openid.claimed_id=" + identifierSelect)
}

func (c *Client) Verify(ctx context.Context, query map[string]string) (string, error) {
	assertion, err := parseAssertion(query)
	if err != nil {
		return "", err
	}
	steamID, err := assertion.steamID(c.cfg.SteamCallbackURL)
	if err != nil {
		log.Printf("steam callback: %v", err)
		return "", err
	}
	if err := c.checkAuthentication(ctx, assertion); err != nil {
		return "", err
	}
	return steamID, nil
}

func (c *Client) checkAuthentication(ctx context.Context, assertion assertion) error {
	_, err := upstream.Fetch(ctx, c.http, upstream.Request{
		Name:    "steam check_authentication",
		Method:  http.MethodPost,
		URL:     c.cfg.Origins.SteamCommunity + openIDLoginPath,
		Form:    assertion.checkAuthenticationForm(),
		OKOnly:  true,
		Refusal: "assertion not confirmed",
	}, func(body []byte) (struct{}, bool) { return struct{}{}, confirmsAssertion(body) })
	switch {
	case errors.Is(err, upstream.ErrRefused):
		return ErrState
	case err != nil:
		return ErrUnavailable
	}
	return nil
}

func (c *Client) License(ctx context.Context, steamID string) license.Result {
	return c.licenses.Check(ctx, steamID, func() license.Result {
		return c.fetchLicense(ctx, steamID)
	})
}

type ownedGamesInput struct {
	SteamID                string `json:"steamid"`
	AppIDsFilter           []int  `json:"appids_filter"`
	IncludeAppInfo         bool   `json:"include_appinfo"`
	IncludePlayedFreeGames bool   `json:"include_played_free_games"`
}

func (c *Client) fetchLicense(ctx context.Context, steamID string) license.Result {
	input, err := json.Marshal(ownedGamesInput{SteamID: steamID, AppIDsFilter: []int{atDeadOfNightAppID}})
	if err != nil {
		return license.Failed
	}
	result, err := upstream.Fetch(ctx, c.http, upstream.Request{
		Name:   "steam GetOwnedGames",
		Method: http.MethodGet,
		URL:    c.cfg.Origins.SteamAPI + "/IPlayerService/GetOwnedGames/v1/?" + compat.EncodeForm("key", c.cfg.SteamAPIKey, "input_json", string(input)),
	}, ownership)
	if err != nil {
		return license.Failed
	}
	return result
}

// ownership reads GetOwnedGames like the old Zod schema: only a valid games list decides, and
// Steam's answer for an account without the game ({"response":{"game_count":0}}) is Unknown.
// The HTTP status was never looked at.
func ownership(body []byte) (license.Result, bool) {
	document, err := compat.DecodeJSON(body)
	if err != nil {
		return license.Failed, false
	}
	appIDs, ok := compat.Each(compat.Field(document, "response", "games"), "appid", compat.Number)
	switch {
	case !ok:
		return license.Unknown, true
	case slices.Contains(appIDs, atDeadOfNightAppID):
		return license.Owned, true
	}
	return license.NotOwned, true
}
