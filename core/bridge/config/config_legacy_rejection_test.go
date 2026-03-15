package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBridgeFileConfigRejectsLegacyProviderFields(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	raw := strings.Join([]string{
		`provider = "openai"`,
		`api_key = "sk-legacy"`,
		`base_url = "https://api.openai.com/v1"`,
		`model_provider = "backup"`,
		`[[model_providers]]`,
		`name = "backup"`,
		`type = "custom"`,
		`base_url = "https://backup.example/v1"`,
	}, "\n")
	if err := os.WriteFile(configPath, []byte(raw), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, _, err := loadBridgeFileConfig()
	if err == nil {
		t.Fatal("expected legacy provider config error")
	}
	for _, field := range []string{"provider", "api_key", "base_url", "model_provider", "model_providers"} {
		if !strings.Contains(err.Error(), field) {
			t.Fatalf("expected error to mention %q, got %v", field, err)
		}
	}
}

func TestLoadBridgeFileConfigRejectsLegacyGraphQLFields(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	raw := strings.Join([]string{
		`graphql_enabled = true`,
		`graphql_endpoint = "https://legacy.example/graphql"`,
		`graphql_api_key = "graphql-legacy-key"`,
		`graphql_schema_path = "/schemas/legacy.json"`,
		`graphql_timeout_ms = 5000`,
		`graphql_max_response_bytes = 8192`,
		`graphql_headers = { "X-Tenant" = "tenant-1" }`,
	}, "\n")
	if err := os.WriteFile(configPath, []byte(raw), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, _, err := loadBridgeFileConfig()
	if err == nil {
		t.Fatal("expected legacy graphql config error")
	}
	for _, field := range []string{
		"graphql_enabled",
		"graphql_endpoint",
		"graphql_api_key",
		"graphql_schema_path",
		"graphql_timeout_ms",
		"graphql_max_response_bytes",
		"graphql_headers",
	} {
		if !strings.Contains(err.Error(), field) {
			t.Fatalf("expected error to mention %q, got %v", field, err)
		}
	}
}

func TestRuntimeConfigFromEnvRejectsLegacyGraphQLEnvVars(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_GRAPHQL_ENABLED", "true")
	t.Setenv("GHOST_GRAPHQL_ENDPOINT", "https://legacy.example/graphql")

	if _, err := runtimeConfigFromEnv(); err == nil {
		t.Fatal("expected legacy graphql env config error")
	} else if !strings.Contains(err.Error(), "GHOST_GRAPHQL_ENABLED") || !strings.Contains(err.Error(), "GHOST_GRAPHQL_ENDPOINT") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func findGraphQLSource(t *testing.T, sources []GraphQLSourceConfig, name string) GraphQLSourceConfig {
	t.Helper()
	for _, source := range sources {
		if source.Name == name {
			return source
		}
	}
	t.Fatalf("graphql source %q was not found in %+v", name, sources)
	return GraphQLSourceConfig{}
}
