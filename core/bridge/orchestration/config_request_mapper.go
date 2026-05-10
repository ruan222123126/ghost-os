package orchestration

import (
	"strings"

	bridgeconfig "ghost-os/bridge/config"
)

func toolUpdateRequestToStoreRequest(name string, req toolUpdateRequest) bridgeconfig.ToolUpdateRequest {
	return bridgeconfig.ToolUpdateRequest{
		Name:            strings.TrimSpace(name),
		Enabled:         req.Enabled,
		PromptOverride:  req.PromptOverride,
		SandboxMemoryMB: req.SandboxMemoryMB,
	}
}

func configUpdateRequestToStoreRequest(req configUpdateRequest) bridgeconfig.UpdateRequest {
	return bridgeconfig.UpdateRequest{
		Provider:                       req.Provider,
		APIKey:                         req.APIKey,
		BaseURL:                        req.BaseURL,
		Model:                          req.Model,
		ChatPath:                       req.ChatPath,
		ProjectRoot:                    req.ProjectRoot,
		MaxTurns:                       req.MaxTurns,
		TaskExecutionTimeoutMS:         req.TaskExecutionTimeoutMs,
		RelayDefaultStopPolicy:         req.RelayDefaultStopPolicy,
		RelayDefaultMaxRounds:          req.RelayDefaultMaxRounds,
		RelayDefaultExecutionTimeoutMS: req.RelayDefaultExecutionTimeoutMs,
		LLMCompletionRetryCount:        req.LlmCompletionRetryCount,
		LLMCompletionRetryIntervalMS:   req.LlmCompletionRetryIntervalMs,
		SessionHumanLogFullEnabled:     req.SessionHumanLogFullEnabled,
		SessionSystemPromptVisible:     req.SessionSystemPromptVisibleEnabled,
		AssistantMarkdownEnabled:       req.AssistantMarkdownEnabled,
		ToolCallCompactOutputEnabled:   req.ToolCallCompactOutputEnabled,
		MemoryModeEnabled:              req.MemoryModeEnabled,
		MicrocompactEnabled:            req.MicrocompactEnabled,
		SessionTitleMode:               req.SessionTitleMode,
		WebSearchTavilyURL:             req.WebSearchTavilyURL,
		WebSearchExaURL:                req.WebSearchExaURL,
		WebSearchTavilyAPIKey:          req.WebSearchTavilyAPIKey,
		WebSearchExaAPIKey:             req.WebSearchExaAPIKey,
		TraceID:                        req.TraceID,
	}
}

func cloneOptionalStringPointer(raw *string) *string {
	if raw == nil {
		return nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil
	}
	return &value
}

func stringValue(raw *string) string {
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(*raw)
}

func cloneStringMap(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}

func cloneModelTokenOverrides(raw map[string]int) map[string]int {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]int, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}
