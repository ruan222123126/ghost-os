package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	bridgeconfig "ghost-os/bridge/config"
	bridgerss "ghost-os/bridge/rss"
	"ghost-os/bridge/session"
	bridgeskills "ghost-os/bridge/skills"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

type AgentExecutorFunc = agentExecutorFunc
type AgentStreamExecutorFunc = agentStreamExecutorFunc
type SessionStore = session.Store
type StreamSink = streaming.Sink

type Service struct {
	inner *bridgeService
}

func NewService(store bridgeconfig.Store, sessionStore *SessionStore, executor AgentExecutorFunc) *Service {
	return &Service{inner: newBridgeService(store, sessionStore, executor)}
}

func NewServiceWithStreamExecutor(
	store bridgeconfig.Store,
	sessionStore *SessionStore,
	executor AgentExecutorFunc,
	streamExecutor AgentStreamExecutorFunc,
) *Service {
	return &Service{inner: newBridgeServiceWithStreamExecutor(store, sessionStore, executor, streamExecutor)}
}

func NewSessionStreamBroadcastSink(sink StreamSink, hub *SessionPushHub) StreamSink {
	return newSessionStreamBroadcastSink(sink, hub)
}

func NewSessionTurnRunnerAdapter(
	store bridgeconfig.Store,
	sessionStore *SessionStore,
	executor AgentExecutorFunc,
	streamExecutor AgentStreamExecutorFunc,
) SessionTurnRunner {
	return newSessionTurnRunnerAdapter(store, sessionStore, executor, streamExecutor)
}

func (s *Service) SessionPushHub() *SessionPushHub {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.sessionPushHub()
}

func (s *Service) ConfigStore() bridgeconfig.Store {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.configStore
}

func (s *Service) SessionStore() *SessionStore {
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

func (s *Service) SetRSSInbox(service *bridgerss.RSSInboxService) {
	s.SetRSSInboxService(service, nil)
}

func (s *Service) SetRSSInboxService(service *bridgerss.RSSInboxService, initErr error) {
	if s == nil || s.inner == nil {
		return
	}
	s.inner.setRSSHandler(service, initErr)
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

func (s *Service) DispatchAction(ctx context.Context, action string, params json.RawMessage, traceID string) (ServiceResult, error) {
	return s.inner.dispatchAction(ctx, action, params, traceID)
}

func (s *Service) ExecuteAgentAction(ctx context.Context, params AgentParams, traceID string) (ServiceResult, error) {
	return s.inner.executeAgentAction(ctx, params, traceID)
}

func ServiceErrorKindFromError(err error) ServiceErrorKind {
	return ServiceErrorKindOf(err)
}

func (s *Service) ExecuteProvidersGetAction(traceID string) (ServiceResult, error) {
	return s.inner.executeProvidersGetAction(traceID)
}

func (s *Service) ExecuteProviderCreateAction(req ProviderCreateRequest, traceID string) (ServiceResult, error) {
	return s.inner.executeProviderCreateAction(req, traceID)
}

func (s *Service) ExecuteProviderUpdateAction(name string, req ProviderUpdateRequest, traceID string) (ServiceResult, error) {
	return s.inner.executeProviderUpdateAction(name, req, traceID)
}

func (s *Service) ExecuteProviderDeleteAction(name string, traceID string) (ServiceResult, error) {
	return s.inner.executeProviderDeleteAction(name, traceID)
}

func (s *Service) ExecuteSetActiveProviderAction(req SetActiveProviderRequest, traceID string) (ServiceResult, error) {
	return s.inner.executeSetActiveProviderAction(req, traceID)
}

func (s *Service) ExecuteSessionsListAction(traceID string) (ServiceResult, error) {
	return s.inner.executeSessionsListAction(traceID)
}

func (s *Service) ExecuteSessionGetAction(params SessionGetParams, traceID string) (ServiceResult, error) {
	return s.inner.executeSessionGetAction(params, traceID)
}

func (s *Service) ExecuteSessionDeleteAction(params SessionIDParams, traceID string) (ServiceResult, error) {
	return s.inner.executeSessionDeleteAction(params, traceID)
}

func (s *Service) ExecuteHumanAnswerAndResumeAction(ctx context.Context, params HumanResponseParams, traceID string) (ServiceResult, error) {
	return s.inner.executeHumanAnswerAndResumeAction(ctx, params, traceID)
}

func (s *Service) ExecuteHumanAnswerAndResumeStreamAction(ctx context.Context, params HumanResponseParams, traceID string, sink streaming.Sink) (string, string, error) {
	return s.inner.executeHumanAnswerAndResumeStreamAction(ctx, params, traceID, sink)
}

func (s *Service) ExecuteAgentStreamAction(ctx context.Context, params AgentParams, traceID string, sink streaming.Sink) (string, string, error) {
	return s.inner.executeAgentStreamAction(ctx, params, traceID, sink)
}

func (s *Service) EnsureSessionNotInflight(sessionID string) error {
	return s.inner.ensureSessionNotInflight(sessionID)
}

func (s *Service) EnsureSessionActive(sessionID string) error {
	return s.inner.ensureSessionActive(sessionID)
}

func (s *Service) ExecuteRSSInboxPollUsecase(
	ctx context.Context,
	params bridgerss.InboxPollParams,
	taskID string,
	traceID string,
) (bridgerss.RSSInboxPollResult, int, error) {
	if s == nil || s.inner == nil || s.inner.rssActionHandler() == nil {
		return bridgerss.RSSInboxPollResult{}, http.StatusInternalServerError, fmt.Errorf("rss inbox service is not configured")
	}
	return s.inner.rssActionHandler().ExecuteInboxPollUsecase(ctx, params, taskID, traceID)
}

func (s *Service) ExecuteSkillListAction(traceID string) (ServiceResult, error) {
	return s.inner.executeSkillListActionResult(traceID)
}

func (s *Service) ExecuteSkillDeleteAction(params bridgeskills.SkillIDParams, traceID string) (ServiceResult, error) {
	return s.inner.executeSkillDeleteActionResult(params, traceID)
}

func (s *Service) ExecuteToolListAction(traceID string) (ServiceResult, error) {
	return s.inner.executeToolListActionResult(traceID)
}

func (s *Service) ExecuteToolUpdateAction(params ToolNameParams, req ToolUpdateRequest, traceID string) (ServiceResult, error) {
	return s.inner.executeToolUpdateActionResult(params, req, traceID)
}

func (s *Service) PendingQuestionSnapshot(sessionID string) (SessionPushEvent, bool) {
	return s.inner.pendingQuestionSnapshot(sessionID)
}

func RequireSessionID(id string) (string, int, error) {
	sessionID, err := requireSessionID(id)
	if err != nil {
		return "", legacyStatusFromServiceError(err), err
	}
	return sessionID, http.StatusOK, nil
}
