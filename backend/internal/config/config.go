package config

import (
	"cmp"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

type Config struct {
	Addr         string
	SteamAPIKey  string
	JWTSecret    []byte
	DownloadsDir string
	Origins      Origins

	// PublicURL is joined to paths as written, like the old backend's host string.
	PublicURL        string
	SteamCallbackURL string
}

type Origins struct {
	SteamCommunity string
	SteamAPI       string
}

const (
	steamCommunityOrigin = "https://steamcommunity.com"
	steamAPIOrigin       = "https://api.steampowered.com"
)

func Load(getenv func(string) string) (Config, error) {
	var missing []string
	require := func(name string) string {
		value := getenv(name)
		if value == "" {
			missing = append(missing, name)
		}
		return value
	}
	steamAPIKey := require("STEAM_API_KEY")
	jwtSecret := require("JWT_SECRET")
	publicURL := require("PUBLIC_URL")
	downloadsDir := require("DOWNLOADS_DIR")
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing environment variables: %s", strings.Join(missing, ", "))
	}

	port := cmp.Or(getenv("PORT"), "3001")
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return Config{}, errors.New("PORT is not a TCP port number")
	}
	if !isPlainBaseURL(publicURL) {
		return Config{}, errors.New("PUBLIC_URL is not an http or https URL without a query or fragment")
	}
	return Config{
		Addr:             net.JoinHostPort(cmp.Or(getenv("HOST"), "127.0.0.1"), port),
		SteamAPIKey:      steamAPIKey,
		JWTSecret:        []byte(jwtSecret),
		DownloadsDir:     downloadsDir,
		Origins:          loadOrigins(getenv),
		PublicURL:        publicURL,
		SteamCallbackURL: publicURL + "/auth/steam/callback",
	}, nil
}

func isPlainBaseURL(raw string) bool {
	if strings.ContainsAny(raw, "#\\\t\r\n ") {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	isHTTP := parsed.Scheme == "http" || parsed.Scheme == "https"
	return isHTTP && parsed.Host != "" && parsed.User == nil && !parsed.ForceQuery && parsed.RawQuery == ""
}
