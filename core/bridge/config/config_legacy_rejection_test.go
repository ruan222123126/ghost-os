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

func TestLoadBridgeFileConfigRejectsRemovedGraphQLFields(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	raw := strings.Join([]string{
		`graphql_tool_runtime_enabled = true`,
		`graphql_text_sanitize_enabled = false`,
		`graphql_default_source = "crm"`,
		`[[graphql_sources]]`,
		`name = "crm"`,
		`endpoint = "https://legacy.example/graphql"`,
		`schema_path = "/schemas/legacy.json"`,
		`[[graphql_mutation_policies]]`,
		`name = "approve"`,
		`source = "crm"`,
		`domain = "orders"`,
		`root_mutation = "approveOrder"`,
	}, "\n")
	if err := os.WriteFile(configPath, []byte(raw), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, _, err := loadBridgeFileConfig()
	if err == nil {
		t.Fatal("expected removed graphql config error")
	}
	for _, field := range []string{
		"graphql_tool_runtime_enabled",
		"graphql_text_sanitize_enabled",
		"graphql_default_source",
		"graphql_sources",
		"graphql_mutation_policies",
	} {
		if !strings.Contains(err.Error(), field) {
			t.Fatalf("expected error to mention %q, got %v", field, err)
		}
	}
}

func TestRuntimeConfigFromEnvRejectsRemovedGraphQLEnvVars(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_GRAPHQL_TOOL_RUNTIME_ENABLED", "true")
	t.Setenv("GHOST_GRAPHQL_TEXT_SANITIZE_ENABLED", "false")

	if _, err := runtimeConfigFromEnv(); err == nil {
		t.Fatal("expected removed graphql env config error")
	} else if !strings.Contains(err.Error(), "GHOST_GRAPHQL_TOOL_RUNTIME_ENABLED") ||
		!strings.Contains(err.Error(), "GHOST_GRAPHQL_TEXT_SANITIZE_ENABLED") {
		t.Fatalf("unexpected error: %v", err)
	}
}
