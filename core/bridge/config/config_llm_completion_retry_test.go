package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConfigLoadsDefaultLLMCompletionRetrySettings(t *testing.T) {
	cfg, err := resolveConfig(bridgeFileConfig{}, envSnapshot{"GHOST_PROVIDER": "custom"})
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.LLMCompletionRetryCount != defaultLLMCompletionRetryCount {
		t.Fatalf(
			"unexpected llm_completion_retry_count: got %d want %d",
			cfg.LLMCompletionRetryCount,
			defaultLLMCompletionRetryCount,
		)
	}
	if cfg.LLMCompletionRetryIntervalMS != defaultLLMCompletionRetryIntervalMS {
		t.Fatalf(
			"unexpected llm_completion_retry_interval_ms: got %d want %d",
			cfg.LLMCompletionRetryIntervalMS,
			defaultLLMCompletionRetryIntervalMS,
		)
	}
}

func TestResolveConfigAllowsZeroLLMCompletionRetrySettings(t *testing.T) {
	cfg, err := resolveConfig(bridgeFileConfig{
		LLMCompletionRetryCount:      intPtr(0),
		LLMCompletionRetryIntervalMS: intPtr(0),
	}, envSnapshot{
		"GHOST_PROVIDER":                         "custom",
		"GHOST_LLM_COMPLETION_RETRY_COUNT":       "5",
		"GHOST_LLM_COMPLETION_RETRY_INTERVAL_MS": "500",
	})
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if cfg.LLMCompletionRetryCount != 0 {
		t.Fatalf("expected llm_completion_retry_count to stay 0, got %d", cfg.LLMCompletionRetryCount)
	}
	if cfg.LLMCompletionRetryIntervalMS != 0 {
		t.Fatalf("expected llm_completion_retry_interval_ms to stay 0, got %d", cfg.LLMCompletionRetryIntervalMS)
	}
}

func TestResolveConfigRejectsInvalidLLMCompletionRetrySettings(t *testing.T) {
	cases := []struct {
		name    string
		fileCfg bridgeFileConfig
		env     envSnapshot
		want    string
	}{
		{
			name: "env negative retry count",
			env: envSnapshot{
				"GHOST_PROVIDER":                   "custom",
				"GHOST_LLM_COMPLETION_RETRY_COUNT": "-1",
			},
			want: "invalid GHOST_LLM_COMPLETION_RETRY_COUNT",
		},
		{
			name: "env invalid retry interval",
			env: envSnapshot{
				"GHOST_PROVIDER":                         "custom",
				"GHOST_LLM_COMPLETION_RETRY_INTERVAL_MS": "fast",
			},
			want: "invalid GHOST_LLM_COMPLETION_RETRY_INTERVAL_MS",
		},
		{
			name: "file negative retry interval",
			fileCfg: bridgeFileConfig{
				LLMCompletionRetryIntervalMS: intPtr(-1),
			},
			env:  envSnapshot{"GHOST_PROVIDER": "custom"},
			want: "invalid llm_completion_retry_interval_ms",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := resolveConfig(tc.fileCfg, tc.env)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestConfigStoreUpdatePersistsZeroLLMCompletionRetrySettings(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	retryCount := 0
	retryIntervalMS := 0
	if err := store.Update(configUpdateRequest{
		LLMCompletionRetryCount:      &retryCount,
		LLMCompletionRetryIntervalMS: &retryIntervalMS,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}

	runtime := store.RuntimeConfig()
	if runtime.LLMCompletionRetryCount != 0 {
		t.Fatalf("unexpected runtime llm_completion_retry_count: %d", runtime.LLMCompletionRetryCount)
	}
	if runtime.LLMCompletionRetryIntervalMS != 0 {
		t.Fatalf("unexpected runtime llm_completion_retry_interval_ms: %d", runtime.LLMCompletionRetryIntervalMS)
	}
	if store.Snapshot().LLMCompletionRetryCount != 0 {
		t.Fatalf("unexpected snapshot llm_completion_retry_count: %d", store.Snapshot().LLMCompletionRetryCount)
	}
	if store.Snapshot().LLMCompletionRetryIntervalMS != 0 {
		t.Fatalf(
			"unexpected snapshot llm_completion_retry_interval_ms: %d",
			store.Snapshot().LLMCompletionRetryIntervalMS,
		)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.LLMCompletionRetryCount == nil || *fileCfg.LLMCompletionRetryCount != 0 {
		t.Fatalf("unexpected persisted llm_completion_retry_count: %#v", fileCfg.LLMCompletionRetryCount)
	}
	if fileCfg.LLMCompletionRetryIntervalMS == nil || *fileCfg.LLMCompletionRetryIntervalMS != 0 {
		t.Fatalf(
			"unexpected persisted llm_completion_retry_interval_ms: %#v",
			fileCfg.LLMCompletionRetryIntervalMS,
		)
	}
}

func TestConfigStoreSnapshotDoesNotMaterializeEnvLLMCompletionRetrySettings(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_LLM_COMPLETION_RETRY_COUNT", "4")
	t.Setenv("GHOST_LLM_COMPLETION_RETRY_INTERVAL_MS", "320")

	store, err := newStoreFromEnv()
	if err != nil {
		t.Fatalf("newStoreFromEnv: %v", err)
	}

	snapshot := store.Snapshot()
	if snapshot.LLMCompletionRetryCount != 4 {
		t.Fatalf("unexpected snapshot llm_completion_retry_count: %d", snapshot.LLMCompletionRetryCount)
	}
	if snapshot.LLMCompletionRetryIntervalMS != 320 {
		t.Fatalf(
			"unexpected snapshot llm_completion_retry_interval_ms: %d",
			snapshot.LLMCompletionRetryIntervalMS,
		)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.LLMCompletionRetryCount != nil {
		t.Fatalf("snapshot should not persist llm_completion_retry_count, got %#v", fileCfg.LLMCompletionRetryCount)
	}
	if fileCfg.LLMCompletionRetryIntervalMS != nil {
		t.Fatalf(
			"snapshot should not persist llm_completion_retry_interval_ms, got %#v",
			fileCfg.LLMCompletionRetryIntervalMS,
		)
	}
}
