package config

func resolveRuntimeCodexRetryEnabled(fileCfg bridgeFileConfig, fallback runtimeConfig) bool {
	if fileCfg.CodexStatelessRetryEnabled != nil {
		return *fileCfg.CodexStatelessRetryEnabled
	}
	return fallback.CodexStatelessRetryEnabled
}

func resolveRuntimeAllowlistOnly(fileCfg bridgeFileConfig, fallback runtimeConfig) bool {
	if fileCfg.ToolAllowlistOnly != nil {
		return *fileCfg.ToolAllowlistOnly
	}
	return !fallback.ModelSelectionEnabled
}

func resolveRuntimeSessionHumanLogFullEnabled(fileCfg bridgeFileConfig, fallback runtimeConfig) bool {
	if fileCfg.SessionHumanLogFullEnabled != nil {
		return *fileCfg.SessionHumanLogFullEnabled
	}
	return fallback.SessionHumanLogFullEnabled
}

func resolveRuntimeAssistantMarkdownEnabled(fileCfg bridgeFileConfig, fallback runtimeConfig) bool {
	if fileCfg.AssistantMarkdownEnabled != nil {
		return *fileCfg.AssistantMarkdownEnabled
	}
	return fallback.AssistantMarkdownEnabled
}

func resolveRuntimeMemoryModeEnabled(fileCfg bridgeFileConfig, fallback runtimeConfig) bool {
	if fileCfg.MemoryModeEnabled != nil {
		return *fileCfg.MemoryModeEnabled
	}
	return fallback.MemoryModeEnabled
}
