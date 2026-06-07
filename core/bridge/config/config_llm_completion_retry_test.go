package config

import (
	"path/filepath"
	"testing"
)

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
