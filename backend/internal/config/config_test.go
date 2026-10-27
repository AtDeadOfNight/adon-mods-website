package config

import (
	"maps"
	"strings"
	"testing"
)

func testEnv(overrides map[string]string) func(string) string {
	values := map[string]string{
		"STEAM_API_KEY": "test-steam-key",
		"JWT_SECRET":    "test-secret-for-parity",
		"PUBLIC_URL":    "https://mods.adon.hloth.dev",
		"DOWNLOADS_DIR": "/nonexistent",
	}
	maps.Copy(values, overrides)
	return func(name string) string { return values[name] }
}

func TestLoadNamesMissingVariables(t *testing.T) {
	_, err := Load(testEnv(map[string]string{"DOWNLOADS_DIR": "", "JWT_SECRET": "", "PUBLIC_URL": ""}))
	if err == nil || err.Error() != "missing environment variables: JWT_SECRET, PUBLIC_URL, DOWNLOADS_DIR" {
		t.Fatalf("error %v", err)
	}
}

func TestLoadRefusesPublicURLWithoutEchoingIt(t *testing.T) {
	for _, bad := range []string{
		"not a url",
		"ftp://mods.adon.hloth.dev",
		"https://mods.adon.hloth.dev?",
		"https://mods.adon.hloth.dev?x=1",
		"https://mods.adon.hloth.dev#x",
		"https://user@mods.adon.hloth.dev",
		"https:///x",
		"https://mods.adon.hloth.dev/a b",
		`https://mods.adon.hloth.dev/a\b`,
	} {
		_, err := Load(testEnv(map[string]string{"PUBLIC_URL": bad, "STEAM_API_KEY": "secret-key-value"}))
		if err == nil || !strings.Contains(err.Error(), "PUBLIC_URL") || strings.Contains(err.Error(), bad) || strings.Contains(err.Error(), "secret-key-value") {
			t.Errorf("PUBLIC_URL %q: error %v must name the variable, not its value", bad, err)
		}
	}
}

func TestLoadRefusesPublicURLEndingWithSlash(t *testing.T) {
	for _, bad := range []string{"https://mods.adon.hloth.dev/", "https://mods.adon.hloth.dev//", "http://localhost:3001/app/"} {
		_, err := Load(testEnv(map[string]string{"PUBLIC_URL": bad}))
		if err == nil || err.Error() != "PUBLIC_URL must not end with a slash: paths are joined to it" {
			t.Errorf("PUBLIC_URL %q: error %v", bad, err)
		}
	}
}

func TestLoadRefusesInvalidPort(t *testing.T) {
	for _, port := range []string{"70000", "-1", "http"} {
		if _, err := Load(testEnv(map[string]string{"PORT": port})); err == nil {
			t.Errorf("PORT %q was accepted", port)
		}
	}
}

func TestLoadDefaultsAndDerivedURLs(t *testing.T) {
	cfg, err := Load(testEnv(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "127.0.0.1:3001" || cfg.SteamCallbackURL != "https://mods.adon.hloth.dev/auth/steam/callback" || cfg.PublicURL != "https://mods.adon.hloth.dev" {
		t.Fatalf("config %+v", cfg)
	}
	cfg, err = Load(testEnv(map[string]string{"HOST": "::1", "PORT": "3002", "PUBLIC_URL": "http://localhost:3001/app"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "[::1]:3002" || cfg.SteamCallbackURL != "http://localhost:3001/app/auth/steam/callback" {
		t.Fatalf("config %+v", cfg)
	}
}
