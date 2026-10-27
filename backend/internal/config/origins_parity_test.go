//go:build parity

package config

import "testing"

func TestParityBuildReadsTestOrigins(t *testing.T) {
	cfg, err := Load(testEnv(map[string]string{"TEST_STEAM_COMMUNITY_ORIGIN": "http://127.0.0.1:39540"}))
	if err != nil {
		t.Fatal(err)
	}
	want := Origins{SteamCommunity: "http://127.0.0.1:39540", SteamAPI: steamAPIOrigin}
	if cfg.Origins != want {
		t.Fatalf("origins %+v, want %+v", cfg.Origins, want)
	}
}
