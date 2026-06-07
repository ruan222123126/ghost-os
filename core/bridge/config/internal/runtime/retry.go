package runtime

import (
	"fmt"

	"ghost-os/bridge/config/internal/storage"
)

func resolveLLMCompletionRetryCount(fileCfg storage.FileConfig, fallback Snapshot) (int, error) {
	return resolveNonNegativeInt(
		fileCfg.LLMCompletionRetryCount,
		"llm_completion_retry_count",
		fallback.LLMCompletionRetryCount,
	)
}

func resolveLLMCompletionRetryIntervalMS(fileCfg storage.FileConfig, fallback Snapshot) (int, error) {
	return resolveNonNegativeInt(
		fileCfg.LLMCompletionRetryIntervalMS,
		"llm_completion_retry_interval_ms",
		fallback.LLMCompletionRetryIntervalMS,
	)
}

func resolveNonNegativeInt(raw *int, fieldName string, fallback int) (int, error) {
	if raw == nil {
		return fallback, nil
	}
	if *raw < 0 {
		return 0, fmt.Errorf("invalid %s: must be >= 0, got %d", fieldName, *raw)
	}
	return *raw, nil
}
