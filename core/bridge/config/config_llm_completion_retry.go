package config

import "fmt"

func resolveRuntimeLLMCompletionRetryCount(
	fileCfg bridgeFileConfig,
	fallback runtimeConfig,
) (int, error) {
	return resolveRuntimeNonNegativeInt(
		fileCfg.LLMCompletionRetryCount,
		"llm_completion_retry_count",
		fallback.LLMCompletionRetryCount,
	)
}

func resolveRuntimeLLMCompletionRetryIntervalMS(
	fileCfg bridgeFileConfig,
	fallback runtimeConfig,
) (int, error) {
	return resolveRuntimeNonNegativeInt(
		fileCfg.LLMCompletionRetryIntervalMS,
		"llm_completion_retry_interval_ms",
		fallback.LLMCompletionRetryIntervalMS,
	)
}

func resolveRuntimeNonNegativeInt(raw *int, fieldName string, fallback int) (int, error) {
	if raw == nil {
		return fallback, nil
	}
	if *raw < 0 {
		return 0, fmt.Errorf("invalid %s: must be >= 0, got %d", fieldName, *raw)
	}
	return *raw, nil
}
