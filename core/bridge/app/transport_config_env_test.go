package app

import (
	"os"
	"path/filepath"
	"testing"

	"ghost-os/bridge/llm"
)

func TestTransportConfigFromTomlConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	providerName := "openai"
	bindAddr := "0.0.0.0:9090"
	apiToken := "secret-token"
	if err := writeBridgeFileConfig(configPathFromEnv(), bridgeFileConfig{
		ActiveProvider: &providerName,
		Providers: map[string]providerFileConfig{
			"openai": {
				Type:    llm.ProviderOpenAI,
				BaseURL: defaultBaseURL,
				APIKey:  optionalStringPointer("file-key"),
			},
		},
		BindAddr:    &bindAddr,
		APIToken:    &apiToken,
		CORSOrigins: []string{"http://localhost:5173"},
	}); err != nil {
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
