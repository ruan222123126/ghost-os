package config

func snapshotFromRuntimeConfig(runtime runtimeConfig) Snapshot {
	runtime = normalizeRuntimeConfig(runtime)
	return Snapshot{
		Provider:                     activeProviderLabel(runtime),
		ProviderType:                 string(runtime.Provider),
		BaseURL:                      runtime.BaseURL,
		Model:                        runtime.Model,
		ChatPath:                     runtime.ChatPath,
		ProjectRoot:                  runtime.ProjectRoot,
		MaxTurns:                     runtime.MaxTurns,
		TaskExecutionTimeoutMS:       runtime.TaskExecutionTimeoutMS,
		LLMCompletionRetryCount:      runtime.LLMCompletionRetryCount,
		LLMCompletionRetryIntervalMS: runtime.LLMCompletionRetryIntervalMS,
		APIKeySet:                    runtime.APIKey != "",
		ModelSelectionEnabled:        runtime.ModelSelectionEnabled,
		SessionHumanLogFullEnabled:   runtime.SessionHumanLogFullEnabled,
		SessionSystemPromptVisible:   runtime.SessionSystemPromptVisible,
		AssistantMarkdownEnabled:     runtime.AssistantMarkdownEnabled,
		ToolCallCompactOutputEnabled: runtime.ToolCallCompactOutputEnabled,
		MemoryModeEnabled:            runtime.MemoryModeEnabled,
		MicrocompactEnabled:          runtime.MicrocompactEnabled,
		WebSearchTavilyURL:           runtime.WebSearchTavilyURL,
		WebSearchExaURL:              runtime.WebSearchExaURL,
		WebSearchTavilyAPIKeySet:     runtime.WebSearchTavilyAPIKey != "",
		WebSearchExaAPIKeySet:        runtime.WebSearchExaAPIKey != "",
	}
}
