package config

import (
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
)

func TestResolveUsesProvidedEnvSnapshot(t *testing.T) {
	processPromptsDir := filepath.Join(t.TempDir(), "process-prompts")
	snapshotPromptsDir := filepath.Join(t.TempDir(), "snapshot-prompts")
	t.Setenv("GHOST_API_KEY", "process-key")
	t.Setenv("GHOST_SESSIONS_PATH", "/process/sessions")
	t.Setenv("GHOST_PROMPTS_DIR", processPromptsDir)
	t.Setenv("GHOST_NATIVE_BINARY_PATH", "/process/native")

	env := envSnapshot{
		"GHOST_PROVIDER":           "openai",
		"GHOST_API_KEY":            "snapshot-key",
		"GHOST_SESSIONS_PATH":      "/snapshot/sessions",
		"GHOST_PROMPTS_DIR":        snapshotPromptsDir,
		"GHOST_NATIVE_BINARY_PATH": "/snapshot/native",
	}

	cfg, err := resolveConfig(bridgeFileConfig{}, env)
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.Provider.APIKey != "snapshot-key" {
		t.Fatalf("unexpected api key: got %q want %q", cfg.Provider.APIKey, "snapshot-key")
	}
	if cfg.SessionsPath != "/snapshot/sessions" {
		t.Fatalf("unexpected sessions path: got %q want %q", cfg.SessionsPath, "/snapshot/sessions")
	}
	if cfg.PromptsDir != snapshotPromptsDir {
		t.Fatalf("unexpected prompts dir: got %q want %q", cfg.PromptsDir, snapshotPromptsDir)
	}
	if cfg.NativeBinaryPath != "/snapshot/native" {
		t.Fatalf("unexpected native binary path: got %q want %q", cfg.NativeBinaryPath, "/snapshot/native")
	}
}

func TestResolveNativeBinaryOverrideBeatsFileConfig(t *testing.T) {
	cfg, err := resolveConfig(
		bridgeFileConfig{
			NativeBinaryPath: stringPointer("/file/native"),
		},
		envSnapshot{
			"GHOST_PROVIDER":                    "custom",
			"GHOST_NATIVE_BINARY_PATH_OVERRIDE": "/override/native",
			"GHOST_NATIVE_BINARY_PATH":          "/env/native",
		},
	)
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.NativeBinaryPath != "/override/native" {
		t.Fatalf("unexpected native binary path: got %q want %q", cfg.NativeBinaryPath, "/override/native")
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

func TestResolveConfigFailsFastOnInvalidEnvValues(t *testing.T) {
	cases := []struct {
		name string
		env  envSnapshot
		want string
	}{
		{
			name: "runtime fallback bool",
			env:  envSnapshot{"GHOST_PROVIDER": "custom", "GHOST_TOOL_ALLOWLIST_ONLY": "maybe"},
			want: "invalid GHOST_TOOL_ALLOWLIST_ONLY",
		},
		{
			name: "rss bool",
			env:  envSnapshot{"GHOST_PROVIDER": "custom", "GHOST_RSS_POLL_ENABLED": "maybe"},
			want: "invalid GHOST_RSS_POLL_ENABLED",
		},
		{
			name: "positive int",
			env:  envSnapshot{"GHOST_PROVIDER": "custom", "GHOST_MAX_TURNS": "0"},
			want: "invalid GHOST_MAX_TURNS",
		},
		{
			name: "float range",
			env:  envSnapshot{"GHOST_PROVIDER": "custom", "GHOST_TOOL_SELECTOR_CONFIDENCE": "1.5"},
			want: "invalid GHOST_TOOL_SELECTOR_CONFIDENCE",
		},
		{
			name: "duration",
			env:  envSnapshot{"GHOST_PROVIDER": "custom", "GHOST_RSS_POLL_INTERVAL": "later"},
			want: "invalid GHOST_RSS_POLL_INTERVAL",
		},
		{
			name: "response store bool",
			env:  envSnapshot{"GHOST_PROVIDER": "custom", "GHOST_RESPONSE_STORE": "maybe"},
			want: "invalid GHOST_RESPONSE_STORE",
		},
		{
			name: "codex stateless retry bool",
			env:  envSnapshot{"GHOST_PROVIDER": "custom", "GHOST_CODEX_STATELESS_RETRY_ENABLED": "maybe"},
			want: "invalid GHOST_CODEX_STATELESS_RETRY_ENABLED",
		},
		{
			name: "session human log full bool",
			env:  envSnapshot{"GHOST_PROVIDER": "custom", "GHOST_SESSION_HUMAN_LOG_FULL_ENABLED": "maybe"},
			want: "invalid GHOST_SESSION_HUMAN_LOG_FULL_ENABLED",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveConfig(bridgeFileConfig{}, tc.env)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestResolveConfigLoadsSessionHumanLogModeFromEnvAndFile(t *testing.T) {
	cfg, err := resolveConfig(
		bridgeFileConfig{
			SessionHumanLogFullEnabled: boolPtr(true),
		},
		envSnapshot{
			"GHOST_PROVIDER":                       "custom",
			"GHOST_SESSION_HUMAN_LOG_FULL_ENABLED": "false",
			"GHOST_CODEX_STATELESS_RETRY_ENABLED":  "false",
		},
	)
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if !cfg.SessionHumanLogFullEnabled {
		t.Fatal("expected session_human_log_full_enabled to be true from file override")
	}
}

func TestResolveConfigLoadsSessionDisplayModesFromDefaultsAndFile(t *testing.T) {
	defaultCfg, err := resolveConfig(
		bridgeFileConfig{},
		envSnapshot{
			"GHOST_PROVIDER": "custom",
		},
	)
	if err != nil {
		t.Fatalf("resolveConfig default: %v", err)
	}
	if !defaultCfg.SessionSystemPromptVisible {
		t.Fatal("expected session_system_prompt_visible_enabled to default to true")
	}
	if !defaultCfg.AssistantMarkdownEnabled {
		t.Fatal("expected assistant_markdown_enabled to default to true")
	}
	if defaultCfg.ToolCallCompactOutputEnabled {
		t.Fatal("expected tool_call_compact_output_enabled to default to false")
	}
	if defaultCfg.MemoryModeEnabled {
		t.Fatal("expected memory_mode_enabled to default to false")
	}
	if defaultCfg.MicrocompactEnabled {
		t.Fatal("expected microcompact_enabled to default to false")
	}
	if defaultCfg.MaxTurns != defaultMaxTurns {
		t.Fatalf("expected max_turns to default to %d, got %d", defaultMaxTurns, defaultCfg.MaxTurns)
	}

	fileOverrideCfg, err := resolveConfig(
		bridgeFileConfig{
			MaxTurns:                     intPtr(7),
			SessionSystemPromptVisible:   boolPtr(false),
			AssistantMarkdownEnabled:     boolPtr(false),
			ToolCallCompactOutputEnabled: boolPtr(true),
			MemoryModeEnabled:            boolPtr(true),
			MicrocompactEnabled:          boolPtr(true),
		},
		envSnapshot{
			"GHOST_PROVIDER": "custom",
		},
	)
	if err != nil {
		t.Fatalf("resolveConfig file override: %v", err)
	}
	if fileOverrideCfg.SessionSystemPromptVisible {
		t.Fatal("expected session_system_prompt_visible_enabled to be false from file override")
	}
	if fileOverrideCfg.AssistantMarkdownEnabled {
		t.Fatal("expected assistant_markdown_enabled to be false from file override")
	}
	if !fileOverrideCfg.ToolCallCompactOutputEnabled {
		t.Fatal("expected tool_call_compact_output_enabled to be true from file override")
	}
	if !fileOverrideCfg.MemoryModeEnabled {
		t.Fatal("expected memory_mode_enabled to be true from file override")
	}
	if !fileOverrideCfg.MicrocompactEnabled {
		t.Fatal("expected microcompact_enabled to be true from file override")
	}
	if fileOverrideCfg.MaxTurns != 7 {
		t.Fatalf("expected max_turns to be 7 from file override, got %d", fileOverrideCfg.MaxTurns)
	}
}

func TestResolveConfigFailsFastOnInvalidFileValues(t *testing.T) {
	cases := []struct {
		name    string
		fileCfg bridgeFileConfig
		want    string
	}{
		{
			name:    "max turns",
			fileCfg: bridgeFileConfig{MaxTurns: intPtr(0)},
			want:    "invalid max_turns",
		},
		{
			name:    "selector confidence",
			fileCfg: bridgeFileConfig{ToolSelectorConfidence: floatPtr(1.1)},
			want:    "invalid tool_selector_confidence",
		},
		{
			name:    "rss poll interval",
			fileCfg: bridgeFileConfig{RSSPollInterval: stringPointer("later")},
			want:    "invalid rss_poll_interval",
		},
	}

	env := envSnapshot{"GHOST_PROVIDER": "custom"}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveConfig(tc.fileCfg, env)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestResolveConfigLoadsResponseOptionsFromEnvAndFile(t *testing.T) {
	fileStore := true
	cfg, err := resolveConfig(
		bridgeFileConfig{
			ResponsePromptCacheKey:       stringPointer("file-cache-key"),
			ResponsePromptCacheRetention: stringPointer("file-retention"),
			ResponseSafetyIdentifier:     stringPointer("file-safe"),
			ResponseStore:                &fileStore,
			ResponseMetadata: map[string]string{
				"source": "file",
			},
			CodexStatelessRetryEnabled: boolPtr(true),
		},
		envSnapshot{
			"GHOST_PROVIDER":                     "custom",
			"GHOST_PROMPT_CACHE_KEY":             "env-cache-key",
			"GHOST_PROMPT_CACHE_RETENTION":       "env-retention",
			"GHOST_SAFETY_IDENTIFIER":            "env-safe",
			"GHOST_RESPONSE_STORE":               "false",
			"GHOST_RESPONSE_METADATA_env_source": "env",
		},
	)
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.ResponseOptions.PromptCacheKey != "file-cache-key" {
		t.Fatalf("unexpected prompt_cache_key: got %q want %q", cfg.ResponseOptions.PromptCacheKey, "file-cache-key")
	}
	if cfg.ResponseOptions.PromptCacheRetention != "file-retention" {
		t.Fatalf(
			"unexpected prompt_cache_retention: got %q want %q",
			cfg.ResponseOptions.PromptCacheRetention,
			"file-retention",
		)
	}
	if cfg.ResponseOptions.SafetyIdentifier != "file-safe" {
		t.Fatalf("unexpected safety_identifier: got %q want %q", cfg.ResponseOptions.SafetyIdentifier, "file-safe")
	}
	if cfg.ResponseOptions.Store == nil || !*cfg.ResponseOptions.Store {
		t.Fatalf("unexpected response store setting: %+v", cfg.ResponseOptions.Store)
	}
	if cfg.ResponseOptions.Metadata["source"] != "file" {
		t.Fatalf("unexpected response metadata: %+v", cfg.ResponseOptions.Metadata)
	}
	if !cfg.CodexStatelessRetryEnabled {
		t.Fatalf("expected codex stateless retry enabled, got false")
	}
}

func TestResolveConfigLoadsResponseMetadataFromEnvPrefix(t *testing.T) {
	cfg, err := resolveConfig(
		bridgeFileConfig{},
		envSnapshot{
			"GHOST_PROVIDER":                     "custom",
			"GHOST_RESPONSE_METADATA_channel":    "bridge",
			"GHOST_RESPONSE_METADATA_request_id": "abc-123",
		},
	)
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.ResponseOptions.Metadata["channel"] != "bridge" {
		t.Fatalf("unexpected response metadata: %+v", cfg.ResponseOptions.Metadata)
	}
	if cfg.ResponseOptions.Metadata["request_id"] != "abc-123" {
		t.Fatalf("unexpected response metadata: %+v", cfg.ResponseOptions.Metadata)
	}
}

func intPtr(value int) *int {
	return &value
}

func floatPtr(value float64) *float64 {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}
