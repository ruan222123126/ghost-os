package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
)

func TestConfigStoreRuntimeConfigReturnsDeepClone(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	err := writeBridgeFileConfig(configPath, bridgeFileConfig{
		ActiveProvider: stringPointer("custom"),
		Providers: map[string]providerFileConfig{
			"custom": {
				Type:                     llm.ProviderCustom,
				BaseURL:                  "https://initial.example/v1",
				APIKey:                   stringPointer("initial-key"),
				ModelContextWindowTokens: map[string]int{"gpt-5.4": 8192},
			},
		},
	})
	if err != nil {
		t.Fatalf("writeBridgeFileConfig: %v", err)
	}

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	runtime := store.RuntimeConfig()
	runtime.ModelContextWindowTokens["gpt-5.4"] = 4096

	fresh := store.RuntimeConfig()
	if fresh.ModelContextWindowTokens["gpt-5.4"] != 8192 {
		t.Fatalf("unexpected model context window tokens: %+v", fresh.ModelContextWindowTokens)
	}
}

func TestConfigStoreListProvidersReturnsConfigReadError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	if err := os.WriteFile(configPath, []byte("providers = ["), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	providers, err := store.ListProviders()
	if err == nil {
		t.Fatal("expected ListProviders to return config read error")
	}
	if providers != nil {
		t.Fatalf("expected nil providers on error, got %+v", providers)
	}
	if !strings.Contains(err.Error(), "read config file") {
		t.Fatalf("unexpected error: %v", err)
	}
}
