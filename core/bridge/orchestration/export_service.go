package orchestration

import (
	"context"
	"encoding/json"

	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

type AgentExecutorFunc = agentExecutorFunc
type AgentStreamExecutorFunc = agentStreamExecutorFunc

type Service struct {
	inner *bridgeService
}

func NewService(store *ConfigStore, sessionStore *session.Store, executor AgentExecutorFunc) *Service {
	return &Service{inner: newBridgeService(store, sessionStore, executor)}
}

func NewServiceWithStreamExecutor(
	store *ConfigStore,
	sessionStore *session.Store,
	executor AgentExecutorFunc,
	streamExecutor AgentStreamExecutorFunc,
) *Service {
	return &Service{inner: newBridgeServiceWithStreamExecutor(store, sessionStore, executor, streamExecutor)}
}

func NewSessionStreamBroadcastSink(sink streaming.Sink, hub *SessionPushHub) streaming.Sink {
	return newSessionStreamBroadcastSink(sink, hub)
}

func NewSessionTurnRunnerAdapter(
	store *ConfigStore,
	sessionStore *session.Store,
	executor AgentExecutorFunc,
	streamExecutor AgentStreamExecutorFunc,
) SessionTurnRunner {
	return newSessionTurnRunnerAdapter(store, sessionStore, executor, streamExecutor)
}

func (s *Service) SessionPushHub() *SessionPushHub {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.sessionPush
}

func (s *Service) ConfigStore() *ConfigStore {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.configStore
}

func (s *Service) SessionStore() *session.Store {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.sessionStore
}

func (s *Service) RunRegistry() *RunRegistry {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.runRegistry
}

func (s *Service) TaskToolManager() tools.TaskManager {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.taskToolManager()
}

func (s *Service) SetAgentRunner(runner SessionTurnRunner) {
	if s != nil && s.inner != nil {
		s.inner.agentRunner = runner
	}
}

func (s *Service) SetRuntimeFactory(factory AgentRuntimeFactory) {
	if s != nil && s.inner != nil && factory != nil {
		s.inner.runtimeFactory = factory
		if runner, ok := s.inner.agentRunner.(*SessionAgentRunner); ok && runner != nil {
			runner.runtimeFactory = factory
		}
	}
}

func (s *Service) SetRSSInbox(service *RSSInboxService) {
	if s != nil && s.inner != nil {
		s.inner.rssInbox = service
		if service == nil {
			s.inner.feedStore = nil
			return
		}
		s.inner.feedStore = service.FeedStore()
	}
}

func (s *Service) SetRSSInboxService(service *RSSInboxService, initErr error) {
	if s == nil || s.inner == nil {
		return
	}
	s.inner.rssInitErr = initErr
	s.SetRSSInbox(service)
}

func (s *Service) StartBackgroundRuntimes() error {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.StartBackgroundRuntimes()
}

func (s *Service) BootstrapSystemTasks() error {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.BootstrapSystemTasks()
}

func (s *Service) Close() {
	if s != nil && s.inner != nil {
		s.inner.Close()
	}
}

func (s *Service) DispatchAction(ctx context.Context, action string, params json.RawMessage, traceID string) (any, int, error) {
	return s.inner.dispatchAction(ctx, action, params, traceID)
}

func (s *Service) ExecuteConfigGetAction(traceID string) (any, int, error) {
	return s.inner.executeConfigGetAction(traceID)
}

func (s *Service) ExecuteConfigUpdateAction(req ConfigUpdateRequest, traceID string) (any, int, error) {
	return s.inner.executeConfigUpdateAction(req, traceID)
}

func (s *Service) ExecuteProvidersGetAction(traceID string) (any, int, error) {
	return s.inner.executeProvidersGetAction(traceID)
}

func (s *Service) ExecuteProviderCreateAction(req ProviderCreateRequest, traceID string) (any, int, error) {
	return s.inner.executeProviderCreateAction(req, traceID)
}

func (s *Service) ExecuteProviderUpdateAction(name string, req ProviderUpdateRequest, traceID string) (any, int, error) {
	return s.inner.executeProviderUpdateAction(name, req, traceID)
}

func (s *Service) ExecuteProviderDeleteAction(name string, traceID string) (any, int, error) {
	return s.inner.executeProviderDeleteAction(name, traceID)
}

func (s *Service) ExecuteSetActiveProviderAction(req SetActiveProviderRequest, traceID string) (any, int, error) {
	return s.inner.executeSetActiveProviderAction(req, traceID)
}

func (s *Service) ExecuteSessionsListAction(traceID string) (any, int, error) {
	return s.inner.executeSessionsListAction(traceID)
}

func (s *Service) ExecuteSessionGetAction(params SessionGetParams, traceID string) (any, int, error) {
	return s.inner.executeSessionGetAction(params, traceID)
}

func (s *Service) ExecuteSessionDeleteAction(params SessionIDParams, traceID string) (any, int, error) {
	return s.inner.executeSessionDeleteAction(params, traceID)
}

func (s *Service) ExecuteHumanAnswerAndResumeAction(ctx context.Context, params HumanResponseParams, traceID string) (any, int, error) {
	return s.inner.executeHumanAnswerAndResumeAction(ctx, params, traceID)
}

func (s *Service) ExecuteHumanAnswerAndResumeStreamAction(ctx context.Context, params HumanResponseParams, traceID string, sink streaming.Sink) (string, string, error) {
	return s.inner.executeHumanAnswerAndResumeStreamAction(ctx, params, traceID, sink)
}

func (s *Service) ExecuteAgentStreamAction(ctx context.Context, params AgentParams, traceID string, sink streaming.Sink) (string, string, error) {
	return s.inner.executeAgentStreamAction(ctx, params, traceID, sink)
}

func (s *Service) EnsureSessionNotInflight(sessionID string) (int, error) {
	return s.inner.ensureSessionNotInflight(sessionID)
}

func (s *Service) EnsureSessionActive(sessionID string) (int, error) {
	return s.inner.ensureSessionActive(sessionID)
}

func (s *Service) ExecuteRSSBriefingGetAction(traceID string) (any, int, error) {
	return s.inner.executeRSSBriefingGetAction(traceID)
}

func (s *Service) ExecuteRSSBriefingBuildAction(ctx context.Context, params RSSBriefingParams, traceID string) (any, int, error) {
	return s.inner.executeRSSBriefingBuildAction(ctx, params, traceID)
}

func (s *Service) ExecuteRSSInboxGroupsAction(params RSSInboxGroupsParams, traceID string) (any, int, error) {
	return s.inner.executeRSSInboxGroupsAction(params, traceID)
}

func (s *Service) ExecuteRSSInboxListAction(params RSSInboxListParams, traceID string) (any, int, error) {
	return s.inner.executeRSSInboxListAction(params, traceID)
}

func (s *Service) ExecuteRSSInboxPollAction(ctx context.Context, params RSSInboxPollParams, traceID string) (any, int, error) {
	return s.inner.executeRSSInboxPollAction(ctx, params, traceID)
}

func (s *Service) ExecuteRSSInboxPollUsecase(
	ctx context.Context,
	params RSSInboxPollParams,
	taskID string,
	traceID string,
) (RSSInboxPollResult, int, error) {
	return s.inner.executeRSSInboxPollUsecase(ctx, params, taskID, traceID)
}

func (s *Service) ExecuteRSSInboxGetAction(params RSSInboxGetParams, traceID string) (any, int, error) {
	return s.inner.executeRSSInboxGetAction(params, traceID)
}

func (s *Service) ExecuteTaskListAction(scope string, traceID string) (any, int, error) {
	return s.inner.executeTaskListAction(scope, traceID)
}

func (s *Service) ExecuteTaskCreateAction(params TaskCreateParams, traceID string) (any, int, error) {
	return s.inner.executeTaskCreateAction(params, traceID)
}

func (s *Service) ExecuteTaskLogsAction(params TaskLogsParams, traceID string) (any, int, error) {
	return s.inner.executeTaskLogsAction(params, traceID)
}

func (s *Service) ExecuteTaskRunNowAction(params TaskIDParams, traceID string) (any, int, error) {
	return s.inner.executeTaskRunNowAction(params, traceID)
}

func (s *Service) ExecuteTaskGetAction(params TaskIDParams, traceID string) (any, int, error) {
	return s.inner.executeTaskGetAction(params, traceID)
}

func (s *Service) ExecuteTaskUpdateAction(params TaskUpdateParams, traceID string) (any, int, error) {
	return s.inner.executeTaskUpdateAction(params, traceID)
}

func (s *Service) ExecuteTaskDeleteAction(params TaskIDParams, traceID string) (any, int, error) {
	return s.inner.executeTaskDeleteAction(params, traceID)
}

func (s *Service) PendingQuestionSnapshot(sessionID string) (SessionPushEvent, bool) {
	return s.inner.pendingQuestionSnapshot(sessionID)
}

func RequireSessionID(id string) (string, int, error) {
	return requireSessionID(id)
}
