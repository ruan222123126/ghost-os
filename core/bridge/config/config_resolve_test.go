package config

import (
	"testing"

	"ghost-os/bridge/llm"
)

func TestResolveUsesProvidedEnvSnapshot(t *testing.T) {
	t.Setenv("GHOST_API_KEY", "process-key")
	t.Setenv("GHOST_SESSIONS_PATH", "/process/sessions")
	t.Setenv("GHOST_PROMPTS_DIR", "/process/prompts")
	t.Setenv("GHOST_NATIVE_BINARY_PATH", "/process/native")

	env := Env{
		"GHOST_PROVIDER":           "openai",
		"GHOST_API_KEY":            "snapshot-key",
		"GHOST_SESSIONS_PATH":      "/snapshot/sessions",
		"GHOST_PROMPTS_DIR":        "/snapshot/prompts",
		"GHOST_NATIVE_BINARY_PATH": "/snapshot/native",
	}

	cfg, err := Resolve(bridgeFileConfig{}, env)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if cfg.Provider.APIKey != "snapshot-key" {
		t.Fatalf("unexpected api key: got %q want %q", cfg.Provider.APIKey, "snapshot-key")
	}
	if cfg.SessionsPath != "/snapshot/sessions" {
		t.Fatalf("unexpected sessions path: got %q want %q", cfg.SessionsPath, "/snapshot/sessions")
	}
	if cfg.PromptsDir != "/snapshot/prompts" {
		t.Fatalf("unexpected prompts dir: got %q want %q", cfg.PromptsDir, "/snapshot/prompts")
	}
	if cfg.NativeBinaryPath != "/snapshot/native" {
		t.Fatalf("unexpected native binary path: got %q want %q", cfg.NativeBinaryPath, "/snapshot/native")
	}
}

func TestResolveRuntimeConfigWithFallbackKeepsSnapshotModelSelection(t *testing.T) {
	t.Setenv("GHOST_TOOL_ALLOWLIST_ONLY", "false")

	runtime, err := resolveRuntimeConfigWithFallback(bridgeFileConfig{}, runtimeConfig{
		ProviderName:          "openai",
		Provider:              llm.ProviderOpenAI,
		APIKey:                "snapshot-key",
		BaseURL:               defaultBaseURL,
		Model:                 defaultModel,
		ModelSelectionEnabled: false,
	})
	if err != nil {
		t.Fatalf("resolveRuntimeConfigWithFallback: %v", err)
	}
	if runtime.ModelSelectionEnabled {
		t.Fatalf("expected model selection to stay disabled, got %+v", runtime)
	}
}
