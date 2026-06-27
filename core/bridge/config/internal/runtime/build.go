package runtime

import (
	"ghost-os/bridge/config/internal/providers"
	"ghost-os/bridge/config/internal/storage"
)

func buildWithProviders(input buildInput, records []providers.Record) Snapshot {
	active := providers.ResolveActive(records, storage.StringValue(input.FileCfg.ActiveProvider), providerRuntimeSnapshot(input.Fallback))
	return Normalize(Snapshot{
		ProviderName:                   active.Name,
		Provider:                       active.Type.Normalized(),
		APIKey:                         resolveAPIKey(active, input.Fallback.APIKey),
		BaseURL:                        resolveBaseURL(active.BaseURL, input.Fallback.BaseURL),
		Model:                          resolveModel(input.FileCfg, input.Fallback),
		ChatPath:                       resolveChatPath(input.FileCfg, input.Fallback),
		ResponseOptions:                input.Settings.ResponseOptions,
		CodexStatelessRetryEnabled:     input.Settings.CodexStatelessRetryEnabled,
		NativePersistent:               resolveNativePersistent(input.FileCfg, input.Fallback),
		ProjectRoot:                    resolveProjectRoot(input.FileCfg, input.Fallback),
		MaxTurns:                       input.Settings.MaxTurns,
		TaskExecutionTimeoutMS:         input.Settings.TaskExecutionTimeoutMS,
		RelayDefaultStopPolicy:         input.Settings.RelayDefaultStopPolicy,
		RelayDefaultMaxRounds:          input.Settings.RelayDefaultMaxRounds,
		RelayDefaultExecutionTimeoutMS: input.Settings.RelayDefaultExecutionTimeoutMS,
		ExternalCodexPermissionMode:    input.Settings.ExternalCodexPermissionMode,
		LLMCompletionRetryCount:        input.Settings.LLMCompletionRetryCount,
		LLMCompletionRetryIntervalMS:   input.Settings.LLMCompletionRetryIntervalMS,
		ModelSelectionEnabled:          input.Settings.ModelSelectionEnabled,
		ContextWindowTokens:            active.ContextWindowTokens,
		ResponseReserveTokens:          active.ResponseReserveTokens,
		ModelContextWindowTokens:       cloneModelTokenOverrides(active.ModelContextWindowTokens),
		ModelResponseReserveTokens:     cloneModelTokenOverrides(active.ModelResponseReserveTokens),
		WebSearchTavilyURL:             input.Settings.WebSearch.TavilyURL,
		WebSearchExaURL:                input.Settings.WebSearch.ExaURL,
		WebSearchTavilyAPIKey:          input.Settings.WebSearch.TavilyAPIKey,
		WebSearchExaAPIKey:             input.Settings.WebSearch.ExaAPIKey,
		SessionHumanLogFullEnabled:     input.Settings.SessionHumanLogFullEnabled,
		SessionSystemPromptVisible:     input.Settings.SessionSystemPromptVisible,
		AssistantMarkdownEnabled:       input.Settings.AssistantMarkdownEnabled,
		ToolCallCompactOutputEnabled:   input.Settings.ToolCallCompactOutputEnabled,
		MemoryModeEnabled:              input.Settings.MemoryModeEnabled,
		MicrocompactEnabled:            input.Settings.MicrocompactEnabled,
		SessionTitleMode:               input.Settings.SessionTitleMode,
	})
}

func buildWithoutProviders(input buildInput) Snapshot {
	providerName := resolveProviderName(input.Fallback)
	return Normalize(Snapshot{
		ProviderName:                   providerName,
		Provider:                       providers.InferType(providerName, input.Fallback.BaseURL, resolveModel(input.FileCfg, input.Fallback)),
		APIKey:                         input.Fallback.APIKey,
		BaseURL:                        input.Fallback.BaseURL,
		Model:                          resolveModel(input.FileCfg, input.Fallback),
		ChatPath:                       resolveChatPath(input.FileCfg, input.Fallback),
		ResponseOptions:                input.Settings.ResponseOptions,
		CodexStatelessRetryEnabled:     input.Settings.CodexStatelessRetryEnabled,
		NativePersistent:               resolveNativePersistent(input.FileCfg, input.Fallback),
		ProjectRoot:                    resolveProjectRoot(input.FileCfg, input.Fallback),
		MaxTurns:                       input.Settings.MaxTurns,
		TaskExecutionTimeoutMS:         input.Settings.TaskExecutionTimeoutMS,
		RelayDefaultStopPolicy:         input.Settings.RelayDefaultStopPolicy,
		RelayDefaultMaxRounds:          input.Settings.RelayDefaultMaxRounds,
		RelayDefaultExecutionTimeoutMS: input.Settings.RelayDefaultExecutionTimeoutMS,
		ExternalCodexPermissionMode:    input.Settings.ExternalCodexPermissionMode,
		LLMCompletionRetryCount:        input.Settings.LLMCompletionRetryCount,
		LLMCompletionRetryIntervalMS:   input.Settings.LLMCompletionRetryIntervalMS,
		ModelSelectionEnabled:          input.Settings.ModelSelectionEnabled,
		ContextWindowTokens:            input.Fallback.ContextWindowTokens,
		ResponseReserveTokens:          input.Fallback.ResponseReserveTokens,
		ModelContextWindowTokens:       cloneModelTokenOverrides(input.Fallback.ModelContextWindowTokens),
		ModelResponseReserveTokens:     cloneModelTokenOverrides(input.Fallback.ModelResponseReserveTokens),
		WebSearchTavilyURL:             input.Settings.WebSearch.TavilyURL,
		WebSearchExaURL:                input.Settings.WebSearch.ExaURL,
		WebSearchTavilyAPIKey:          input.Settings.WebSearch.TavilyAPIKey,
		WebSearchExaAPIKey:             input.Settings.WebSearch.ExaAPIKey,
		SessionHumanLogFullEnabled:     input.Settings.SessionHumanLogFullEnabled,
		SessionSystemPromptVisible:     input.Settings.SessionSystemPromptVisible,
		AssistantMarkdownEnabled:       input.Settings.AssistantMarkdownEnabled,
		ToolCallCompactOutputEnabled:   input.Settings.ToolCallCompactOutputEnabled,
		MemoryModeEnabled:              input.Settings.MemoryModeEnabled,
		MicrocompactEnabled:            input.Settings.MicrocompactEnabled,
		SessionTitleMode:               input.Settings.SessionTitleMode,
	})
}
