// Runtime config snapshot/update use cases exposed to the transport layer.

package orchestration

import (
	"strings"
	"time"

	bridgeconfig "ghost-os/bridge/config"
)

// executeConfigGetAction 返回当前可编辑配置快照，不暴露敏感明文字段。
func (s *bridgeService) executeConfigGetAction(traceID string) (ServiceResult, error) {
	response, err := s.publicConfigResponse()
	if err != nil {
		logAction(traceID, busActionConfigGet, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}
	logAction(traceID, busActionConfigGet, "success", nil)
	return serviceResultSuccess(response), nil
}

// executeConfigUpdateAction 按请求局部更新运行态配置，并返回更新后可编辑快照。
func (s *bridgeService) executeConfigUpdateAction(req configUpdateRequest, traceID string) (ServiceResult, error) {
	logAction(traceID, busActionConfigUpdate, "running", nil)
	if err := s.configStore.Update(configUpdateRequestToStoreRequest(req)); err != nil {
		logAction(traceID, busActionConfigUpdate, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	s.syncTaskSchedulerExecutionTimeout()
	if err := s.BootstrapSystemTasks(); err != nil {
		logAction(traceID, busActionConfigUpdate, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}

	response, err := s.publicConfigResponse()
	if err != nil {
		logAction(traceID, busActionConfigUpdate, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}
	logAction(traceID, busActionConfigUpdate, "success", nil)
	return serviceResultSuccess(response), nil
}

func (s *bridgeService) syncTaskSchedulerExecutionTimeout() {
	if s == nil {
		return
	}
	scheduler := s.taskScheduler()
	if scheduler == nil {
		return
	}
	cfg, err := s.configStore.Config()
	if err != nil {
		return
	}
	scheduler.SetExecutionTimeout(time.Duration(cfg.Task.ExecutionTimeoutMS) * time.Millisecond)
}

func (s *bridgeService) publicConfigResponse() (configResponse, error) {
	snapshot, err := s.configStore.PublicSnapshot()
	if err != nil {
		return configResponse{}, err
	}
	return configResponseFromSnapshot(snapshot), nil
}

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
