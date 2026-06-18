package config

import (
	"strings"
	"testing"
)

func TestResolveMobileWebRTCConfigUsesEnvOverrides(t *testing.T) {
	t.Parallel()

	cfg, err := resolveMobileWebRTCConfig(bridgeFileConfig{}, envSnapshot{
		"GHOST_MOBILE_WEBRTC_ENABLED":        "true",
		"GHOST_SIGNALING_URL":                "wss://signal.example/ws",
		"GHOST_SIGNALING_TOKEN":              "token",
		"GHOST_MOBILE_PC_ID":                 "pc-1",
		"GHOST_ICE_SERVERS_JSON":             `[{"urls":["turn:turn.example:3478"],"username":"u","credential":"p"}]`,
		"GHOST_MOBILE_CREDENTIAL_STORE_PATH": t.TempDir() + "/devices.json",
	})
	if err != nil {
		t.Fatalf("resolveMobileWebRTCConfig: %v", err)
	}
	if !cfg.Enabled || cfg.SignalingURL != "wss://signal.example/ws" || cfg.PCID != "pc-1" {
		t.Fatalf("unexpected mobile config: %+v", cfg)
	}
	if len(cfg.ICEServers) != 1 || cfg.ICEServers[0].URLs[0] != "turn:turn.example:3478" {
		t.Fatalf("unexpected ice servers: %+v", cfg.ICEServers)
	}
}

func TestResolveMobileWebRTCConfigRequiresSignalingWhenEnabled(t *testing.T) {
	t.Parallel()

	_, err := resolveMobileWebRTCConfig(bridgeFileConfig{}, envSnapshot{
		"GHOST_MOBILE_WEBRTC_ENABLED": "true",
	})
	if err == nil || !strings.Contains(err.Error(), "signaling_url is required") {
		t.Fatalf("expected signaling_url error, got %v", err)
	}
}

func TestResolveMobileWebRTCConfigRejectsInvalidICEJSON(t *testing.T) {
	t.Parallel()

	_, err := resolveMobileWebRTCConfig(bridgeFileConfig{}, envSnapshot{
		"GHOST_ICE_SERVERS_JSON": "{",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid GHOST_ICE_SERVERS_JSON") {
		t.Fatalf("expected invalid ice json error, got %v", err)
	}
}

func TestResolveMobileWebRTCConfigAcceptsSingleICEURL(t *testing.T) {
	t.Parallel()

	cfg, err := resolveMobileWebRTCConfig(bridgeFileConfig{}, envSnapshot{
		"GHOST_ICE_SERVERS_JSON": `[{"urls":"stun:stun.example:3478"}]`,
	})
	if err != nil {
		t.Fatalf("resolveMobileWebRTCConfig: %v", err)
	}
	if len(cfg.ICEServers) != 1 || cfg.ICEServers[0].URLs[0] != "stun:stun.example:3478" {
		t.Fatalf("unexpected ice servers: %+v", cfg.ICEServers)
	}
}
