// Runtime config snapshot/update use cases exposed to the transport layer.

package orchestration

import (
	"strings"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	appconfig "ghost-os/bridge/orchestration/internal/app/config"
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

func (s *bridgeService) configProviderService() appconfig.Service {
	return appconfig.Service{
		Store:  s.configStore,
		Logger: serviceActionLogger{},
	}
}

func (s *bridgeService) executeProvidersGetAction(traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.configProviderService().List(traceID))
}

func (s *bridgeService) executeProviderCreateAction(req providerCreateRequest, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.configProviderService().Create(req, traceID))
}

func (s *bridgeService) executeProviderUpdateAction(
	name string,
	req providerUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return adaptLegacyResult(s.configProviderService().Update(name, req, traceID))
}

func (s *bridgeService) executeProviderDeleteAction(name string, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.configProviderService().Delete(name, traceID))
}

func (s *bridgeService) executeSetActiveProviderAction(
	req setActiveProviderRequest,
	traceID string,
) (ServiceResult, error) {
	return adaptLegacyResult(s.configProviderService().SetActive(req, traceID))
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
	return appconfig.CloneOptionalStringPointer(raw)
}

func stringValue(raw *string) string {
	return appconfig.StringValue(raw)
}

func cloneModelTokenOverrides(raw map[string]int) map[string]int {
	return appconfig.CloneModelTokenOverrides(raw)
}
