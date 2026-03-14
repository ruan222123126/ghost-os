package transport

import (
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
