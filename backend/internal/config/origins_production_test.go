//go:build !parity

package config

import "testing"

func TestProductionBuildIgnoresTestOrigins(t *testing.T) {
	cfg, err := Load(testEnv(map[string]string{"TEST_STEAM_COMMUNITY_ORIGIN": "http://127.0.0.1:39540", "TEST_STEAM_API_ORIGIN": "http://127.0.0.1:39540"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Origins != (Origins{SteamCommunity: steamCommunityOrigin, SteamAPI: steamAPIOrigin}) {
		t.Fatalf("origins %+v", cfg.Origins)
	}
}
