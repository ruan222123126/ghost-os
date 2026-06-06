package config

import (
	"net/http"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

type RuntimeUpdateHook func() error

func (s Service) GetRuntime(traceID string) (api.ConfigResponse, int, error) {
	response, err := s.publicRuntimeConfig()
	if err != nil {
		s.log(traceID, bus.ActionConfigGet, "error", err)
		return api.ConfigResponse{}, http.StatusInternalServerError, err
	}
	s.log(traceID, bus.ActionConfigGet, "success", nil)
	return response, http.StatusOK, nil
}

func (s Service) UpdateRuntime(
	req api.ConfigUpdateRequest,
	traceID string,
	afterUpdate RuntimeUpdateHook,
) (api.ConfigResponse, int, error) {
	s.log(traceID, bus.ActionConfigUpdate, "running", nil)
	if err := s.Store.Update(UpdateRequestToStoreRequest(req)); err != nil {
		s.log(traceID, bus.ActionConfigUpdate, "error", err)
		return api.ConfigResponse{}, http.StatusBadRequest, err
	}
	if afterUpdate != nil {
		if err := afterUpdate(); err != nil {
			s.log(traceID, bus.ActionConfigUpdate, "error", err)
			return api.ConfigResponse{}, http.StatusInternalServerError, err
		}
	}

	response, err := s.publicRuntimeConfig()
	if err != nil {
		s.log(traceID, bus.ActionConfigUpdate, "error", err)
		return api.ConfigResponse{}, http.StatusInternalServerError, err
	}
	s.log(traceID, bus.ActionConfigUpdate, "success", nil)
	return response, http.StatusOK, nil
}

func (s Service) publicRuntimeConfig() (api.ConfigResponse, error) {
	snapshot, err := s.Store.PublicSnapshot()
	if err != nil {
		return api.ConfigResponse{}, err
	}
	return ResponseFromSnapshot(snapshot), nil
}

func UpdateRequestToStoreRequest(req api.ConfigUpdateRequest) bridgeconfig.UpdateRequest {
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

func ResponseFromSnapshot(snapshot bridgeconfig.Snapshot) api.ConfigResponse {
	return api.ConfigResponse{
		Provider:                          snapshot.Provider,
		ProviderType:                      snapshot.ProviderType,
		BaseURL:                           snapshot.BaseURL,
		Model:                             snapshot.Model,
		ChatPath:                          snapshot.ChatPath,
		ProjectRoot:                       snapshot.ProjectRoot,
		MaxTurns:                          snapshot.MaxTurns,
		TaskExecutionTimeoutMs:            snapshot.TaskExecutionTimeoutMS,
		RelayDefaultStopPolicy:            snapshot.RelayDefaultStopPolicy,
		RelayDefaultMaxRounds:             snapshot.RelayDefaultMaxRounds,
		RelayDefaultExecutionTimeoutMs:    snapshot.RelayDefaultExecutionTimeoutMS,
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
		SessionTitleMode:                  snapshot.SessionTitleMode,
		WebSearchTavilyURL:                snapshot.WebSearchTavilyURL,
		WebSearchExaURL:                   snapshot.WebSearchExaURL,
		WebSearchTavilyAPIKeySet:          snapshot.WebSearchTavilyAPIKeySet,
		WebSearchExaAPIKeySet:             snapshot.WebSearchExaAPIKeySet,
	}
}
