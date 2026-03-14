package transport

import (
	"context"
	"encoding/json"

	"ghost-os/bridge/agent"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

const (
	defaultMaxRequestBodyBytes   = bridgeorchestration.DefaultMaxRequestBodyBytes
	busActionAgentSend           = bridgeorchestration.BusActionAgentSend
	busStatusSuccess             = bridgeorchestration.BusStatusSuccess
	busStatusError               = bridgeorchestration.BusStatusError
	busAssistantSessionEndSignal = bridgeorchestration.BusAssistantSessionEndSignal
	taskListScopeUser            = bridgeorchestration.TaskListScopeUser
	taskListScopeSystem          = bridgeorchestration.TaskListScopeSystem

	sessionPushAssistantMessage = bridgeorchestration.SessionPushAssistantMessage
	sessionPushAwaitingHuman    = bridgeorchestration.SessionPushAwaitingHuman
	sessionPushRunStarted       = bridgeorchestration.SessionPushRunStarted
	sessionPushCompletionDelta  = bridgeorchestration.SessionPushCompletionDelta
	sessionPushToolCallStarted  = bridgeorchestration.SessionPushToolCallStarted
	sessionPushToolCallFinished = bridgeorchestration.SessionPushToolCallFinished
	sessionPushError            = bridgeorchestration.SessionPushError
	sessionPushDone             = bridgeorchestration.SessionPushDone
)

type apiRequest = bridgeorchestration.APIRequest
type apiResponse = bridgeorchestration.APIResponse
type agentRequest = bridgeorchestration.AgentRequest
type agentParams = bridgeorchestration.AgentParams
type humanResponseParams = bridgeorchestration.HumanResponseParams
type sessionIDParams = bridgeorchestration.SessionIDParams
type configUpdateRequest = bridgeorchestration.ConfigUpdateRequest
type providerCreateRequest = bridgeorchestration.ProviderCreateRequest
type providerUpdateRequest = bridgeorchestration.ProviderUpdateRequest
type setActiveProviderRequest = bridgeorchestration.SetActiveProviderRequest
type taskCreateParams = bridgeorchestration.TaskCreateParams
type taskUpdateParams = bridgeorchestration.TaskUpdateParams
type taskIDParams = bridgeorchestration.TaskIDParams
type taskLogsParams = bridgeorchestration.TaskLogsParams
type rssInboxPollParams = bridgeorchestration.RSSInboxPollParams
type rssInboxListParams = bridgeorchestration.RSSInboxListParams
type rssInboxGetParams = bridgeorchestration.RSSInboxGetParams
type rssInboxGroupsParams = bridgeorchestration.RSSInboxGroupsParams
type rssBriefingParams = bridgeorchestration.RSSBriefingParams
type sessionPushEventType = bridgeorchestration.SessionPushEventType
type sessionPushEvent = bridgeorchestration.SessionPushEvent
type sessionPushHub = bridgeorchestration.SessionPushHub
type agentExecutorFunc = bridgeorchestration.AgentExecutorFunc
type agentStreamExecutorFunc = bridgeorchestration.AgentStreamExecutorFunc
type agentRuntimeDependencies = bridgeorchestration.RuntimeDependencies

func validateBusRequest(req apiRequest) error {
	return bridgeorchestration.ValidateBusRequest(req)
}

func newSessionPushHub() *sessionPushHub {
	return bridgeorchestration.NewSessionPushHub()
}

func newSessionStreamBroadcastSink(sink streaming.Sink, hub *sessionPushHub) streaming.Sink {
	return bridgeorchestration.NewSessionStreamBroadcastSink(sink, hub)
}

func newSessionTurnRunnerAdapter(
	store *ConfigStore,
	sessionStore *session.Store,
	executor agentExecutorFunc,
	streamExecutor agentStreamExecutorFunc,
) bridgeorchestration.SessionTurnRunner {
	return bridgeorchestration.NewSessionTurnRunnerAdapter(store, sessionStore, executor, streamExecutor)
}

func newRuntimeDependencies(
	cfg Config,
	client agent.Completer,
	registry *tools.Registry,
	systemPrompt string,
	cleanup func(),
) agentRuntimeDependencies {
	return bridgeorchestration.NewRuntimeDependencies(cfg, client, registry, systemPrompt, cleanup)
}

type bridgeService struct {
	inner        *bridgeorchestration.Service
	configStore  *ConfigStore
	sessionStore *session.Store
	runRegistry  *bridgeorchestration.RunRegistry
	sessionPush  *sessionPushHub
}

func newBridgeService(store *ConfigStore, sessionStore *session.Store, executor agentExecutorFunc) *bridgeService {
	return wrapBridgeService(bridgeorchestration.NewService(store, sessionStore, executor))
}

func newBridgeServiceWithStreamExecutor(
	store *ConfigStore,
	sessionStore *session.Store,
	executor agentExecutorFunc,
	streamExecutor agentStreamExecutorFunc,
) *bridgeService {
	return wrapBridgeService(bridgeorchestration.NewServiceWithStreamExecutor(store, sessionStore, executor, streamExecutor))
}

func wrapBridgeService(service *bridgeorchestration.Service) *bridgeService {
	if service == nil {
		return &bridgeService{}
	}
	return &bridgeService{
		inner:        service,
		configStore:  service.ConfigStore(),
		sessionStore: service.SessionStore(),
		runRegistry:  service.RunRegistry(),
		sessionPush:  service.SessionPushHub(),
	}
}

func (s *bridgeService) StartBackgroundRuntimes() error {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.StartBackgroundRuntimes()
}

func (s *bridgeService) BootstrapSystemTasks() error {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.BootstrapSystemTasks()
}

func (s *bridgeService) Close() {
	if s != nil && s.inner != nil {
		s.inner.Close()
	}
}

func (s *bridgeService) ConfigStore() *ConfigStore {
	if s == nil {
		return nil
	}
	return s.inner.ConfigStore()
}

func (s *bridgeService) SessionStore() *session.Store {
	if s == nil {
		return nil
	}
	return s.inner.SessionStore()
}

func (s *bridgeService) RunRegistry() *bridgeorchestration.RunRegistry {
	if s == nil {
		return nil
	}
	return s.inner.RunRegistry()
}

func (s *bridgeService) TaskToolManager() any {
	if s == nil {
		return nil
	}
	return s.inner.TaskToolManager()
}

func (s *bridgeService) SetAgentRunner(runner bridgeorchestration.SessionTurnRunner) {
	if s != nil && s.inner != nil {
		s.inner.SetAgentRunner(runner)
	}
}

func (s *bridgeService) SetRuntimeFactory(factory bridgeorchestration.AgentRuntimeFactory) {
	if s != nil && s.inner != nil {
		s.inner.SetRuntimeFactory(factory)
	}
}

func (s *bridgeService) SetRSSInbox(service *bridgeorchestration.RSSInboxService) {
	if s != nil && s.inner != nil {
		s.inner.SetRSSInbox(service)
	}
}

func (s *bridgeService) SetRSSInboxService(service *bridgeorchestration.RSSInboxService, initErr error) {
	if s != nil && s.inner != nil {
		s.inner.SetRSSInboxService(service, initErr)
	}
}

func (s *bridgeService) executeConfigGetAction(traceID string) (any, int, error) {
	return s.inner.ExecuteConfigGetAction(traceID)
}

func (s *bridgeService) executeConfigUpdateAction(req configUpdateRequest, traceID string) (any, int, error) {
	return s.inner.ExecuteConfigUpdateAction(req, traceID)
}

func (s *bridgeService) executeProvidersGetAction(traceID string) (any, int, error) {
	return s.inner.ExecuteProvidersGetAction(traceID)
}

func (s *bridgeService) executeProviderCreateAction(req providerCreateRequest, traceID string) (any, int, error) {
	return s.inner.ExecuteProviderCreateAction(req, traceID)
}

func (s *bridgeService) executeProviderUpdateAction(name string, req providerUpdateRequest, traceID string) (any, int, error) {
	return s.inner.ExecuteProviderUpdateAction(name, req, traceID)
}

func (s *bridgeService) executeProviderDeleteAction(name string, traceID string) (any, int, error) {
	return s.inner.ExecuteProviderDeleteAction(name, traceID)
}

func (s *bridgeService) executeSetActiveProviderAction(req setActiveProviderRequest, traceID string) (any, int, error) {
	return s.inner.ExecuteSetActiveProviderAction(req, traceID)
}

func (s *bridgeService) executeSessionsListAction(traceID string) (any, int, error) {
	return s.inner.ExecuteSessionsListAction(traceID)
}

func (s *bridgeService) executeSessionGetAction(params sessionIDParams, traceID string) (any, int, error) {
	return s.inner.ExecuteSessionGetAction(params, traceID)
}

func (s *bridgeService) executeSessionDeleteAction(params sessionIDParams, traceID string) (any, int, error) {
	return s.inner.ExecuteSessionDeleteAction(params, traceID)
}

func (s *bridgeService) executeHumanAnswerAndResumeAction(ctx context.Context, params humanResponseParams, traceID string) (any, int, error) {
	return s.inner.ExecuteHumanAnswerAndResumeAction(ctx, params, traceID)
}

func (s *bridgeService) executeHumanAnswerAndResumeStreamAction(ctx context.Context, params humanResponseParams, traceID string, sink streaming.Sink) (string, string, error) {
	return s.inner.ExecuteHumanAnswerAndResumeStreamAction(ctx, params, traceID, sink)
}

func (s *bridgeService) ensureSessionNotInflight(sessionID string) (int, error) {
	return s.inner.EnsureSessionNotInflight(sessionID)
}

func (s *bridgeService) ensureSessionActive(sessionID string) (int, error) {
	return s.inner.EnsureSessionActive(sessionID)
}

func (s *bridgeService) executeAgentStreamAction(ctx context.Context, params agentParams, traceID string, sink streaming.Sink) (string, string, error) {
	return s.inner.ExecuteAgentStreamAction(ctx, params, traceID, sink)
}

func (s *bridgeService) dispatchAction(ctx context.Context, action string, params json.RawMessage, traceID string) (any, int, error) {
	return s.inner.DispatchAction(ctx, action, params, traceID)
}

func (s *bridgeService) executeRSSBriefingGetAction(traceID string) (any, int, error) {
	return s.inner.ExecuteRSSBriefingGetAction(traceID)
}

func (s *bridgeService) executeRSSBriefingBuildAction(ctx context.Context, params rssBriefingParams, traceID string) (any, int, error) {
	return s.inner.ExecuteRSSBriefingBuildAction(ctx, params, traceID)
}

func (s *bridgeService) executeRSSInboxGroupsAction(params rssInboxGroupsParams, traceID string) (any, int, error) {
	return s.inner.ExecuteRSSInboxGroupsAction(params, traceID)
}

func (s *bridgeService) executeRSSInboxListAction(params rssInboxListParams, traceID string) (any, int, error) {
	return s.inner.ExecuteRSSInboxListAction(params, traceID)
}

func (s *bridgeService) executeRSSInboxPollAction(ctx context.Context, params rssInboxPollParams, traceID string) (any, int, error) {
	return s.inner.ExecuteRSSInboxPollAction(ctx, params, traceID)
}

func (s *bridgeService) executeRSSInboxPollUsecase(ctx context.Context, params rssInboxPollParams, taskID string, traceID string) (bridgeorchestration.RSSInboxPollResult, int, error) {
	return s.inner.ExecuteRSSInboxPollUsecase(ctx, params, taskID, traceID)
}

func (s *bridgeService) executeRSSInboxGetAction(params rssInboxGetParams, traceID string) (any, int, error) {
	return s.inner.ExecuteRSSInboxGetAction(params, traceID)
}

func (s *bridgeService) executeTaskListAction(scope string, traceID string) (any, int, error) {
	return s.inner.ExecuteTaskListAction(scope, traceID)
}

func (s *bridgeService) executeTaskCreateAction(params taskCreateParams, traceID string) (any, int, error) {
	return s.inner.ExecuteTaskCreateAction(params, traceID)
}

func (s *bridgeService) executeTaskLogsAction(params taskLogsParams, traceID string) (any, int, error) {
	return s.inner.ExecuteTaskLogsAction(params, traceID)
}

func (s *bridgeService) executeTaskRunNowAction(params taskIDParams, traceID string) (any, int, error) {
	return s.inner.ExecuteTaskRunNowAction(params, traceID)
}

func (s *bridgeService) executeTaskGetAction(params taskIDParams, traceID string) (any, int, error) {
	return s.inner.ExecuteTaskGetAction(params, traceID)
}

func (s *bridgeService) executeTaskUpdateAction(params taskUpdateParams, traceID string) (any, int, error) {
	return s.inner.ExecuteTaskUpdateAction(params, traceID)
}

func (s *bridgeService) executeTaskDeleteAction(params taskIDParams, traceID string) (any, int, error) {
	return s.inner.ExecuteTaskDeleteAction(params, traceID)
}

func (s *bridgeService) pendingQuestionSnapshot(sessionID string) (sessionPushEvent, bool) {
	return s.inner.PendingQuestionSnapshot(sessionID)
}

func requireSessionID(id string) (string, int, error) {
	return bridgeorchestration.RequireSessionID(id)
}
