package orchestration

import bridgeconfig "ghost-os/bridge/config"

func configResponseFromSnapshot(snapshot bridgeconfig.Snapshot) configResponse {
	return configResponse{
		Provider:                          snapshot.Provider,
		ProviderType:                      snapshot.ProviderType,
		BaseURL:                           snapshot.BaseURL,
		Model:                             snapshot.Model,
		ChatPath:                          snapshot.ChatPath,
		ProjectRoot:                       snapshot.ProjectRoot,
		MaxTurns:                          snapshot.MaxTurns,
		TaskExecutionTimeoutMs:            snapshot.TaskExecutionTimeoutMS,
		LlmCompletionRetryCount:           snapshot.LLMCompletionRetryCount,
		LlmCompletionRetryIntervalMs:      snapshot.LLMCompletionRetryIntervalMS,
		APIKeySet:                         snapshot.APIKeySet,
		ModelSelectionEnabled:             snapshot.ModelSelectionEnabled,
		SessionHumanLogFullEnabled:        snapshot.SessionHumanLogFullEnabled,
		SessionSystemPromptVisibleEnabled: snapshot.SessionSystemPromptVisible,
		AssistantMarkdownEnabled:          snapshot.AssistantMarkdownEnabled,
		ToolCallCompactOutputEnabled:      snapshot.ToolCallCompactOutputEnabled,
		MemoryModeEnabled:                 snapshot.MemoryModeEnabled,
		MicrocompactEnabled:               snapshot.MicrocompactEnabled,
		WebSearchTavilyURL:                snapshot.WebSearchTavilyURL,
		WebSearchExaURL:                   snapshot.WebSearchExaURL,
		WebSearchTavilyAPIKeySet:          snapshot.WebSearchTavilyAPIKeySet,
		WebSearchExaAPIKeySet:             snapshot.WebSearchExaAPIKeySet,
	}
}
