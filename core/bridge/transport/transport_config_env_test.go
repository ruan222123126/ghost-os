package transport

import (
	"os"
	"path/filepath"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
)

func TestTransportConfigFromTomlConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	bindAddr := "0.0.0.0:9090"
	apiToken := "secret-token"
	configBody := `
active_provider = "openai"
bind_addr = "0.0.0.0:9090"
api_token = "secret-token"
cors_origins = ["http://localhost:5173"]

[providers.openai]
type = "openai"
base_url = "` + bridgeconfig.DefaultBaseURL + `"
api_key = "file-key"
`
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	if got, err := resolveBindAddr(8080); err != nil {
		t.Fatalf("resolve bind addr: %v", err)
	} else if got != bindAddr {
		t.Fatalf("unexpected bind addr: got %q want %q", got, bindAddr)
	}
	if auth, err := newAPITokenAuthFromEnv(); err != nil {
		t.Fatalf("resolve api token auth: %v", err)
	} else if auth.token != apiToken {
		t.Fatalf("unexpected api token: got %q want %q", auth.token, apiToken)
	}
	policy, err := newCORSPolicyFromEnv()
	if err != nil {
		t.Fatalf("resolve CORS policy: %v", err)
	}
	if !policy.allows("http://localhost:5173") {
		t.Fatalf("expected origin from config file to be allowed")
	}
	if policy.allows("https://evil.example") {
		t.Fatalf("unexpected allow for unconfigured origin")
	}
}

func TestTransportConfigParseErrorFailsClosed(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_API_TOKEN", "")
	t.Setenv("GHOST_CORS_ORIGINS", "")
	t.Setenv("GHOST_BIND_ADDR", "")

	if err := os.WriteFile(configPath, []byte("bind_addr = [\n"), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	if _, err := newServerOptionsFromEnv(8080); err == nil {
		t.Fatalf("expected server option resolution to fail when config parsing fails")
	}
	if _, err := resolveBindAddr(8080); err == nil {
		t.Fatalf("expected bind addr resolution to fail when config parsing fails")
	}
	if _, err := newAPITokenAuthFromEnv(); err == nil {
		t.Fatalf("expected api token auth resolution to fail when config parsing fails")
	}
	if _, err := newCORSPolicyFromEnv(); err == nil {
		t.Fatalf("expected CORS policy resolution to fail when config parsing fails")
	}
}
