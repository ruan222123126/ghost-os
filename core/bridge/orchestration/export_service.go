package orchestration

import (
	"context"
	"encoding/json"

	bridgeconfig "ghost-os/bridge/config"
	agentadapter "ghost-os/bridge/orchestration/internal/adapters/agent"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/session"
	bridgeskills "ghost-os/bridge/skills"
	"ghost-os/bridge/streaming"
)

type AgentExecutorFunc = agentExecutorFunc
type AgentStreamExecutorFunc = agentStreamExecutorFunc
type SessionStore = session.Store
type StreamSink = streaming.Sink
type ServiceOutcome = bus.ServiceOutcome
type ServiceErrorKind = bus.ServiceErrorKind
type ServiceResult = bus.ServiceResult
type SystemPromptUpdateRequest = bridgeconfig.SystemPromptUpdateRequest
type PresetCreateRequest = bridgeconfig.PresetCreateRequest
type PresetUpdateRequest = bridgeconfig.PresetUpdateRequest
type SkillIDParams = bridgeskills.SkillIDParams
type SkillUpdateRequest = bridgeskills.SkillUpdateRequest
type SessionAppendRequest = sessionAppendRequest
type SessionAppendResponse = sessionAppendResponse

const (
	ServiceOutcomeSuccess  = bus.ServiceOutcomeSuccess
	ServiceOutcomeCreated  = bus.ServiceOutcomeCreated
	ServiceOutcomeAccepted = bus.ServiceOutcomeAccepted
)

const (
	ServiceErrorInvalidInput = bus.ServiceErrorInvalidInput
	ServiceErrorNotFound     = bus.ServiceErrorNotFound
	ServiceErrorConflict     = bus.ServiceErrorConflict
	ServiceErrorUnavailable  = bus.ServiceErrorUnavailable
	ServiceErrorInternal     = bus.ServiceErrorInternal
)

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
	return internaltrace.NewSessionStreamBroadcastSink(sink, hub)
}

func NewSessionTurnRunnerAdapter(
	store bridgeconfig.Store,
	sessionStore *SessionStore,
	executor AgentExecutorFunc,
	streamExecutor AgentStreamExecutorFunc,
) SessionTurnRunner {
	return agentadapter.NewExecutorRunner(
		store,
		sessionStore,
		agentadapter.ExecutorFunc(executor),
		agentadapter.StreamExecutorFunc(streamExecutor),
	)
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

func ServiceErrorKindOf(err error) ServiceErrorKind        { return bus.ErrorKindOf(err) }
func ServiceErrorKindFromError(err error) ServiceErrorKind { return ServiceErrorKindOf(err) }
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

func (s *Service) ExecuteSessionsSearchAction(query string, limit int, traceID string) (ServiceResult, error) {
	return s.inner.executeSessionsSearchQueryAction(query, limit, traceID)
}

func (s *Service) ExecuteSessionSourcesAction(traceID string) (ServiceResult, error) {
	return s.inner.executeSessionSourcesAction(traceID)
}

func (s *Service) ExecuteSessionGetAction(params SessionGetParams, traceID string) (ServiceResult, error) {
	return s.inner.executeSessionGetAction(params, traceID)
}

func (s *Service) ExecuteSessionAppendAction(req SessionAppendRequest, traceID string) (ServiceResult, error) {
	return s.inner.executeSessionAppendAction(req, traceID)
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

func (s *Service) PrepareAgentStreamAction(ctx context.Context, params AgentParams, traceID string) (PreparedAgentStream, ServiceResult, error) {
	return s.inner.prepareAgentStreamAction(ctx, params, traceID)
}

func (s *Service) PrepareExternalAgentStreamAction(
	params ExternalAgentRequest,
	traceID string,
	forceStart bool,
) (PreparedAgentStream, ServiceResult, error) {
	return s.inner.prepareExternalAgentStreamAction(params, traceID, forceStart)
}

func (s *Service) ExecuteExternalAgentStopAction(ctx context.Context, params ExternalAgentStopParams, traceID string) (ServiceResult, error) {
	return s.inner.executeExternalAgentStopAction(ctx, params, traceID)
}

func (s *Service) ExecuteExternalAgentApproveAction(ctx context.Context, params ExternalAgentApprovalParams, traceID string) (ServiceResult, error) {
	return s.inner.executeExternalAgentApproveAction(ctx, params, traceID)
}

func (s *Service) ExecuteSkillListAction(traceID string) (ServiceResult, error) {
	return s.inner.executeSkillListActionResult(traceID)
}

func (s *Service) ExecuteSkillUpdateAction(
	params bridgeskills.SkillIDParams,
	req bridgeskills.SkillUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return s.inner.executeSkillUpdateActionResult(params, req, traceID)
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

func (s *Service) ExecuteSystemPromptGetAction(traceID string) (ServiceResult, error) {
	return s.inner.executeSystemPromptGetAction(traceID)
}

func (s *Service) ExecuteSystemPromptUpdateAction(req bridgeconfig.SystemPromptUpdateRequest, traceID string) (ServiceResult, error) {
	return s.inner.executeSystemPromptUpdateAction(req, traceID)
}

func (s *Service) ExecutePresetListAction(traceID string) (ServiceResult, error) {
	return s.inner.executePresetListAction(traceID)
}

func (s *Service) ExecutePresetCreateAction(req bridgeconfig.PresetCreateRequest, traceID string) (ServiceResult, error) {
	return s.inner.executePresetCreateAction(req, traceID)
}

func (s *Service) ExecutePresetUpdateAction(presetID string, req bridgeconfig.PresetUpdateRequest, traceID string) (ServiceResult, error) {
	return s.inner.executePresetUpdateAction(presetID, req, traceID)
}

func (s *Service) ExecutePresetDeleteAction(presetID string, traceID string) (ServiceResult, error) {
	return s.inner.executePresetDeleteAction(presetID, traceID)
}

func (s *Service) ExecutePresetApplyAction(presetID string, traceID string) (ServiceResult, error) {
	return s.inner.executePresetApplyAction(presetID, traceID)
}

func (s *Service) ExecuteFindIconTemplateUploadAction(req FindIconTemplateUploadRequest, traceID string) (ServiceResult, error) {
	return s.inner.executeFindIconTemplateUploadActionResult(req, traceID)
}

func (s *Service) ExecuteFindIconPreviewAction(
	ctx context.Context,
	req FindIconPreviewRequest,
	traceID string,
) (ServiceResult, error) {
	return s.inner.executeFindIconPreviewActionResult(ctx, req, traceID)
}

func (s *Service) ExecuteMousePositionAction(
	ctx context.Context,
	req MousePositionRequest,
	traceID string,
) (ServiceResult, error) {
	return s.inner.executeMousePositionActionResult(ctx, req, traceID)
}

func (s *Service) PendingQuestionSnapshot(sessionID string) (SessionPushEvent, bool) {
	return s.inner.pendingQuestionSnapshot(sessionID)
}
