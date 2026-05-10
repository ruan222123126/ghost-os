package config

import "ghost-os/bridge/llm"

func cloneRuntimeConfig(raw runtimeConfig) runtimeConfig {
	return runtimeConfig{
		ProviderName:                   raw.ProviderName,
		Provider:                       raw.Provider,
		APIKey:                         raw.APIKey,
		BaseURL:                        raw.BaseURL,
		Model:                          raw.Model,
		ChatPath:                       raw.ChatPath,
		ResponseOptions:                cloneResponseOptions(raw.ResponseOptions),
		CodexStatelessRetryEnabled:     raw.CodexStatelessRetryEnabled,
		NativePersistent:               raw.NativePersistent,
		ProjectRoot:                    raw.ProjectRoot,
		MaxTurns:                       raw.MaxTurns,
		TaskExecutionTimeoutMS:         raw.TaskExecutionTimeoutMS,
		RelayDefaultStopPolicy:         raw.RelayDefaultStopPolicy,
		RelayDefaultMaxRounds:          raw.RelayDefaultMaxRounds,
		RelayDefaultExecutionTimeoutMS: raw.RelayDefaultExecutionTimeoutMS,
		ModelSelectionEnabled:          raw.ModelSelectionEnabled,
		ContextWindowTokens:            raw.ContextWindowTokens,
		ResponseReserveTokens:          raw.ResponseReserveTokens,
		ModelContextWindowTokens:       cloneModelTokenOverrides(raw.ModelContextWindowTokens),
		ModelResponseReserveTokens:     cloneModelTokenOverrides(raw.ModelResponseReserveTokens),
		WebSearchTavilyURL:             raw.WebSearchTavilyURL,
		WebSearchExaURL:                raw.WebSearchExaURL,
		WebSearchTavilyAPIKey:          raw.WebSearchTavilyAPIKey,
		WebSearchExaAPIKey:             raw.WebSearchExaAPIKey,
		LLMCompletionRetryCount:        raw.LLMCompletionRetryCount,
		LLMCompletionRetryIntervalMS:   raw.LLMCompletionRetryIntervalMS,
		SessionHumanLogFullEnabled:     raw.SessionHumanLogFullEnabled,
		SessionSystemPromptVisible:     raw.SessionSystemPromptVisible,
		AssistantMarkdownEnabled:       raw.AssistantMarkdownEnabled,
		ToolCallCompactOutputEnabled:   raw.ToolCallCompactOutputEnabled,
		MemoryModeEnabled:              raw.MemoryModeEnabled,
		MicrocompactEnabled:            raw.MicrocompactEnabled,
		SessionTitleMode:               raw.SessionTitleMode,
	}
}

func cloneResponseOptions(raw llm.ResponseOptions) llm.ResponseOptions {
	out := llm.ResponseOptions{
		PromptCacheKey:       raw.PromptCacheKey,
		PromptCacheRetention: raw.PromptCacheRetention,
		SafetyIdentifier:     raw.SafetyIdentifier,
		Metadata:             cloneStringMap(raw.Metadata),
	}
	if raw.Store != nil {
		value := *raw.Store
		out.Store = &value
	}
	return out
}

func cloneStringSlice(raw []string) []string {
	if raw == nil {
		return nil
	}

	out := make([]string, len(raw))
	copy(out, raw)
	return out
}

func cloneStringMap(raw map[string]string) map[string]string {
	if raw == nil {
		return nil
	}

	out := make(map[string]string, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}
