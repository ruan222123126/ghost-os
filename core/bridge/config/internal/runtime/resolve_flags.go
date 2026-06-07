package runtime

import "ghost-os/bridge/config/internal/storage"

func resolveCodexRetryEnabled(fileCfg storage.FileConfig, fallback Snapshot) bool {
	if fileCfg.CodexStatelessRetryEnabled != nil {
		return *fileCfg.CodexStatelessRetryEnabled
	}
	return fallback.CodexStatelessRetryEnabled
}

func resolveModelSelectionEnabled(fileCfg storage.FileConfig, fallback Snapshot) bool {
	if fileCfg.ModelSelectionEnabled != nil {
		return *fileCfg.ModelSelectionEnabled
	}
	return fallback.ModelSelectionEnabled
}

func resolveSessionHumanLogFullEnabled(fileCfg storage.FileConfig, fallback Snapshot) bool {
	if fileCfg.SessionHumanLogFullEnabled != nil {
		return *fileCfg.SessionHumanLogFullEnabled
	}
	return fallback.SessionHumanLogFullEnabled
}

func resolveSessionSystemPromptVisible(fileCfg storage.FileConfig, fallback Snapshot) bool {
	if fileCfg.SessionSystemPromptVisible != nil {
		return *fileCfg.SessionSystemPromptVisible
	}
	return fallback.SessionSystemPromptVisible
}

func resolveAssistantMarkdownEnabled(fileCfg storage.FileConfig, fallback Snapshot) bool {
	if fileCfg.AssistantMarkdownEnabled != nil {
		return *fileCfg.AssistantMarkdownEnabled
	}
	return fallback.AssistantMarkdownEnabled
}

func resolveToolCallCompactOutputEnabled(fileCfg storage.FileConfig, fallback Snapshot) bool {
	if fileCfg.ToolCallCompactOutputEnabled != nil {
		return *fileCfg.ToolCallCompactOutputEnabled
	}
	return fallback.ToolCallCompactOutputEnabled
}

func resolveMemoryModeEnabled(fileCfg storage.FileConfig, fallback Snapshot) bool {
	if fileCfg.MemoryModeEnabled != nil {
		return *fileCfg.MemoryModeEnabled
	}
	return fallback.MemoryModeEnabled
}

func resolveMicrocompactEnabled(fileCfg storage.FileConfig, fallback Snapshot) bool {
	if fileCfg.MicrocompactEnabled != nil {
		return *fileCfg.MicrocompactEnabled
	}
	return fallback.MicrocompactEnabled
}
