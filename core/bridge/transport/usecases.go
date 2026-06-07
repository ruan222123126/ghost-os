package transport

import (
	"context"
	"encoding/json"

	bridgeorchestration "ghost-os/bridge/orchestration"
)

type BusUsecase interface {
	Dispatch(ctx context.Context, action string, params json.RawMessage, traceID string) (bridgeorchestration.ServiceResult, error)
}

type AgentUsecase interface {
	Send(ctx context.Context, params bridgeorchestration.AgentParams, traceID string) (bridgeorchestration.ServiceResult, error)
	Answer(ctx context.Context, params bridgeorchestration.HumanResponseParams, traceID string) (bridgeorchestration.ServiceResult, error)
	AnswerStream(
		ctx context.Context,
		params bridgeorchestration.HumanResponseParams,
		traceID string,
		sink bridgeorchestration.StreamSink,
	) (string, string, error)
	PrepareStream(
		ctx context.Context,
		params bridgeorchestration.AgentParams,
		traceID string,
	) (bridgeorchestration.PreparedAgentStream, bridgeorchestration.ServiceResult, error)
}

type SessionUsecase interface {
	List(traceID string) (bridgeorchestration.ServiceResult, error)
	Search(query string, limit int, traceID string) (bridgeorchestration.ServiceResult, error)
	Sources(traceID string) (bridgeorchestration.ServiceResult, error)
	Get(params bridgeorchestration.SessionGetParams, traceID string) (bridgeorchestration.ServiceResult, error)
	Delete(params bridgeorchestration.SessionIDParams, traceID string) (bridgeorchestration.ServiceResult, error)
	ListPartitions(traceID string) (bridgeorchestration.ServiceResult, error)
	SavePartitions(
		req bridgeorchestration.SessionSidebarPartitionPutRequest,
		traceID string,
	) (bridgeorchestration.ServiceResult, error)
	OpenArtifactDownload(sessionID string, artifactID string) (bridgeorchestration.BinaryDownload, error)
}

type StreamUsecase interface {
	PendingQuestionSnapshot(sessionID string) (bridgeorchestration.SessionPushEvent, bool)
	SessionPushHub() *bridgeorchestration.SessionPushHub
}

type TaskUsecase interface {
	ListUserTasks(traceID string) (bridgeorchestration.ServiceResult, error)
	ListSystemTasks(traceID string) (bridgeorchestration.ServiceResult, error)
	ListOrchestrations(traceID string) (bridgeorchestration.ServiceResult, error)
	CreateUserTask(req bridgeorchestration.TaskCreateParams, traceID string) (bridgeorchestration.ServiceResult, error)
	CreateOrchestration(req bridgeorchestration.TaskCreateParams, traceID string) (bridgeorchestration.ServiceResult, error)
	GetUserTask(id string, traceID string) (bridgeorchestration.ServiceResult, error)
	UpdateUserTask(id string, req bridgeorchestration.TaskUpdateParams, traceID string) (bridgeorchestration.ServiceResult, error)
	DeleteUserTask(id string, traceID string) (bridgeorchestration.ServiceResult, error)
	RunUserTask(id string, startOnly *bool, traceID string) (bridgeorchestration.ServiceResult, error)
	StopUserTask(ctx context.Context, id string, req bridgeorchestration.TaskStopParams, traceID string) (bridgeorchestration.ServiceResult, error)
	UserTaskLogs(id string, limit *int, traceID string) (bridgeorchestration.ServiceResult, error)
	GetOrchestrationTask(id string, traceID string) (bridgeorchestration.ServiceResult, error)
	UpdateOrchestrationTask(id string, req bridgeorchestration.TaskUpdateParams, traceID string) (bridgeorchestration.ServiceResult, error)
	DeleteOrchestrationTask(id string, traceID string) (bridgeorchestration.ServiceResult, error)
	RunOrchestrationTask(id string, startOnly *bool, traceID string) (bridgeorchestration.ServiceResult, error)
	StopOrchestrationTask(ctx context.Context, id string, req bridgeorchestration.TaskStopParams, traceID string) (bridgeorchestration.ServiceResult, error)
	OrchestrationTaskLogs(id string, limit *int, traceID string) (bridgeorchestration.ServiceResult, error)
}

type ConfigUsecase interface {
	Get(traceID string) (bridgeorchestration.ServiceResult, error)
	Update(req bridgeorchestration.ConfigUpdateRequest, traceID string) (bridgeorchestration.ServiceResult, error)
	ListProviders(traceID string) (bridgeorchestration.ServiceResult, error)
	CreateProvider(req bridgeorchestration.ProviderCreateRequest, traceID string) (bridgeorchestration.ServiceResult, error)
	UpdateProvider(
		name string,
		req bridgeorchestration.ProviderUpdateRequest,
		traceID string,
	) (bridgeorchestration.ServiceResult, error)
	DeleteProvider(name string, traceID string) (bridgeorchestration.ServiceResult, error)
	SetActiveProvider(req bridgeorchestration.SetActiveProviderRequest, traceID string) (bridgeorchestration.ServiceResult, error)
	GetSystemPrompts(traceID string) (bridgeorchestration.ServiceResult, error)
	UpdateSystemPrompts(
		req bridgeorchestration.SystemPromptUpdateRequest,
		traceID string,
	) (bridgeorchestration.ServiceResult, error)
}

type PresetUsecase interface {
	List(traceID string) (bridgeorchestration.ServiceResult, error)
	Create(req bridgeorchestration.PresetCreateRequest, traceID string) (bridgeorchestration.ServiceResult, error)
	Update(presetID string, req bridgeorchestration.PresetUpdateRequest, traceID string) (bridgeorchestration.ServiceResult, error)
	Delete(presetID string, traceID string) (bridgeorchestration.ServiceResult, error)
	Apply(presetID string, traceID string) (bridgeorchestration.ServiceResult, error)
}

type SkillUsecase interface {
	List(traceID string) (bridgeorchestration.ServiceResult, error)
	Update(
		params bridgeorchestration.SkillIDParams,
		req bridgeorchestration.SkillUpdateRequest,
		traceID string,
	) (bridgeorchestration.ServiceResult, error)
	Delete(params bridgeorchestration.SkillIDParams, traceID string) (bridgeorchestration.ServiceResult, error)
}

type ToolUsecase interface {
	List(traceID string) (bridgeorchestration.ServiceResult, error)
	Update(
		params bridgeorchestration.ToolNameParams,
		req bridgeorchestration.ToolUpdateRequest,
		traceID string,
	) (bridgeorchestration.ServiceResult, error)
	UploadFindIconTemplate(
		req bridgeorchestration.FindIconTemplateUploadRequest,
		traceID string,
	) (bridgeorchestration.ServiceResult, error)
	PreviewFindIcon(
		ctx context.Context,
		req bridgeorchestration.FindIconPreviewRequest,
		traceID string,
	) (bridgeorchestration.ServiceResult, error)
	DownloadFindIconTemplate(templatePath string) (bridgeorchestration.BinaryDownload, error)
	MousePosition(
		ctx context.Context,
		req bridgeorchestration.MousePositionRequest,
		traceID string,
	) (bridgeorchestration.ServiceResult, error)
}

type transportUsecases struct {
	bus      BusUsecase
	agent    AgentUsecase
	sessions SessionUsecase
	streams  StreamUsecase
	tasks    TaskUsecase
	config   ConfigUsecase
	presets  PresetUsecase
	skills   SkillUsecase
	tools    ToolUsecase
}

func newTransportUsecases(service *bridgeorchestration.Service) transportUsecases {
	return transportUsecases{
		bus:      orchestrationBusAdapter{service: service},
		agent:    orchestrationAgentAdapter{service: service},
		sessions: orchestrationSessionAdapter{service: service},
		streams:  orchestrationStreamAdapter{service: service},
		tasks:    orchestrationTaskUsecase{service: service},
		config:   orchestrationConfigAdapter{service: service},
		presets:  orchestrationPresetAdapter{service: service},
		skills:   orchestrationSkillAdapter{service: service},
		tools:    orchestrationToolAdapter{service: service},
	}
}

type orchestrationBusAdapter struct {
	service *bridgeorchestration.Service
}

func (a orchestrationBusAdapter) Dispatch(
	ctx context.Context,
	action string,
	params json.RawMessage,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.DispatchAction(ctx, action, params, traceID)
}

type orchestrationAgentAdapter struct {
	service *bridgeorchestration.Service
}

func (a orchestrationAgentAdapter) Send(
	ctx context.Context,
	params bridgeorchestration.AgentParams,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteAgentAction(ctx, params, traceID)
}

func (a orchestrationAgentAdapter) Answer(
	ctx context.Context,
	params bridgeorchestration.HumanResponseParams,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteHumanAnswerAndResumeAction(ctx, params, traceID)
}

func (a orchestrationAgentAdapter) AnswerStream(
	ctx context.Context,
	params bridgeorchestration.HumanResponseParams,
	traceID string,
	sink bridgeorchestration.StreamSink,
) (string, string, error) {
	return a.service.ExecuteHumanAnswerAndResumeStreamAction(ctx, params, traceID, sink)
}

func (a orchestrationAgentAdapter) PrepareStream(
	ctx context.Context,
	params bridgeorchestration.AgentParams,
	traceID string,
) (bridgeorchestration.PreparedAgentStream, bridgeorchestration.ServiceResult, error) {
	return a.service.PrepareAgentStreamAction(ctx, params, traceID)
}

type orchestrationSessionAdapter struct {
	service *bridgeorchestration.Service
}

func (a orchestrationSessionAdapter) List(traceID string) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteSessionsListAction(traceID)
}

func (a orchestrationSessionAdapter) Search(
	query string,
	limit int,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteSessionsSearchAction(query, limit, traceID)
}

func (a orchestrationSessionAdapter) Sources(traceID string) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteSessionSourcesAction(traceID)
}

func (a orchestrationSessionAdapter) Get(
	params bridgeorchestration.SessionGetParams,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteSessionGetAction(params, traceID)
}

func (a orchestrationSessionAdapter) Delete(
	params bridgeorchestration.SessionIDParams,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteSessionDeleteAction(params, traceID)
}

func (a orchestrationSessionAdapter) ListPartitions(traceID string) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteSessionSidebarPartitionsGetAction(traceID)
}

func (a orchestrationSessionAdapter) SavePartitions(
	req bridgeorchestration.SessionSidebarPartitionPutRequest,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteSessionSidebarPartitionsPutAction(req, traceID)
}

func (a orchestrationSessionAdapter) OpenArtifactDownload(
	sessionID string,
	artifactID string,
) (bridgeorchestration.BinaryDownload, error) {
	return a.service.OpenSessionArtifactDownload(sessionID, artifactID)
}

type orchestrationStreamAdapter struct {
	service *bridgeorchestration.Service
}

func (a orchestrationStreamAdapter) PendingQuestionSnapshot(
	sessionID string,
) (bridgeorchestration.SessionPushEvent, bool) {
	return a.service.PendingQuestionSnapshot(sessionID)
}

func (a orchestrationStreamAdapter) SessionPushHub() *bridgeorchestration.SessionPushHub {
	return a.service.SessionPushHub()
}

type orchestrationConfigAdapter struct {
	service *bridgeorchestration.Service
}

func (a orchestrationConfigAdapter) Get(traceID string) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteConfigGetAction(traceID)
}

func (a orchestrationConfigAdapter) Update(
	req bridgeorchestration.ConfigUpdateRequest,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteConfigUpdateAction(req, traceID)
}

func (a orchestrationConfigAdapter) ListProviders(traceID string) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteProvidersGetAction(traceID)
}

func (a orchestrationConfigAdapter) CreateProvider(
	req bridgeorchestration.ProviderCreateRequest,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteProviderCreateAction(req, traceID)
}

func (a orchestrationConfigAdapter) UpdateProvider(
	name string,
	req bridgeorchestration.ProviderUpdateRequest,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteProviderUpdateAction(name, req, traceID)
}

func (a orchestrationConfigAdapter) DeleteProvider(name string, traceID string) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteProviderDeleteAction(name, traceID)
}

func (a orchestrationConfigAdapter) SetActiveProvider(
	req bridgeorchestration.SetActiveProviderRequest,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteSetActiveProviderAction(req, traceID)
}

func (a orchestrationConfigAdapter) GetSystemPrompts(traceID string) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteSystemPromptGetAction(traceID)
}

func (a orchestrationConfigAdapter) UpdateSystemPrompts(
	req bridgeorchestration.SystemPromptUpdateRequest,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteSystemPromptUpdateAction(req, traceID)
}

type orchestrationPresetAdapter struct {
	service *bridgeorchestration.Service
}

func (a orchestrationPresetAdapter) List(traceID string) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecutePresetListAction(traceID)
}

func (a orchestrationPresetAdapter) Create(
	req bridgeorchestration.PresetCreateRequest,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecutePresetCreateAction(req, traceID)
}

func (a orchestrationPresetAdapter) Update(
	presetID string,
	req bridgeorchestration.PresetUpdateRequest,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecutePresetUpdateAction(presetID, req, traceID)
}

func (a orchestrationPresetAdapter) Delete(presetID string, traceID string) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecutePresetDeleteAction(presetID, traceID)
}

func (a orchestrationPresetAdapter) Apply(presetID string, traceID string) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecutePresetApplyAction(presetID, traceID)
}

type orchestrationSkillAdapter struct {
	service *bridgeorchestration.Service
}

func (a orchestrationSkillAdapter) List(traceID string) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteSkillListAction(traceID)
}

func (a orchestrationSkillAdapter) Update(
	params bridgeorchestration.SkillIDParams,
	req bridgeorchestration.SkillUpdateRequest,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteSkillUpdateAction(params, req, traceID)
}

func (a orchestrationSkillAdapter) Delete(
	params bridgeorchestration.SkillIDParams,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteSkillDeleteAction(params, traceID)
}

type orchestrationToolAdapter struct {
	service *bridgeorchestration.Service
}

func (a orchestrationToolAdapter) List(traceID string) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteToolListAction(traceID)
}

func (a orchestrationToolAdapter) Update(
	params bridgeorchestration.ToolNameParams,
	req bridgeorchestration.ToolUpdateRequest,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteToolUpdateAction(params, req, traceID)
}

func (a orchestrationToolAdapter) UploadFindIconTemplate(
	req bridgeorchestration.FindIconTemplateUploadRequest,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteFindIconTemplateUploadAction(req, traceID)
}

func (a orchestrationToolAdapter) PreviewFindIcon(
	ctx context.Context,
	req bridgeorchestration.FindIconPreviewRequest,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteFindIconPreviewAction(ctx, req, traceID)
}

func (a orchestrationToolAdapter) DownloadFindIconTemplate(
	templatePath string,
) (bridgeorchestration.BinaryDownload, error) {
	return a.service.DownloadFindIconTemplate(templatePath)
}

func (a orchestrationToolAdapter) MousePosition(
	ctx context.Context,
	req bridgeorchestration.MousePositionRequest,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return a.service.ExecuteMousePositionAction(ctx, req, traceID)
}

type orchestrationTaskUsecase struct {
	service *bridgeorchestration.Service
}

func (u orchestrationTaskUsecase) ListUserTasks(traceID string) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteUserTaskListAction(traceID)
}

func (u orchestrationTaskUsecase) ListSystemTasks(traceID string) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteSystemTaskListAction(traceID)
}

func (u orchestrationTaskUsecase) ListOrchestrations(traceID string) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteOrchestrationListAction(traceID)
}

func (u orchestrationTaskUsecase) CreateUserTask(
	req bridgeorchestration.TaskCreateParams,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteUserTaskCreateAction(req, traceID)
}

func (u orchestrationTaskUsecase) CreateOrchestration(
	req bridgeorchestration.TaskCreateParams,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteOrchestrationCreateAction(req, traceID)
}

func (u orchestrationTaskUsecase) GetUserTask(
	id string,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteUserTaskGetAction(id, traceID)
}

func (u orchestrationTaskUsecase) UpdateUserTask(
	id string,
	req bridgeorchestration.TaskUpdateParams,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteUserTaskUpdateAction(id, req, traceID)
}

func (u orchestrationTaskUsecase) DeleteUserTask(
	id string,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteUserTaskDeleteAction(id, traceID)
}

func (u orchestrationTaskUsecase) RunUserTask(
	id string,
	startOnly *bool,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteUserTaskRunAction(id, startOnly, traceID)
}

func (u orchestrationTaskUsecase) StopUserTask(
	ctx context.Context,
	id string,
	req bridgeorchestration.TaskStopParams,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteUserTaskStopAction(ctx, id, req, traceID)
}

func (u orchestrationTaskUsecase) UserTaskLogs(
	id string,
	limit *int,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteUserTaskLogsAction(id, limit, traceID)
}

func (u orchestrationTaskUsecase) GetOrchestrationTask(
	id string,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteOrchestrationTaskGetAction(id, traceID)
}

func (u orchestrationTaskUsecase) UpdateOrchestrationTask(
	id string,
	req bridgeorchestration.TaskUpdateParams,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteOrchestrationTaskUpdateAction(id, req, traceID)
}

func (u orchestrationTaskUsecase) DeleteOrchestrationTask(
	id string,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteOrchestrationTaskDeleteAction(id, traceID)
}

func (u orchestrationTaskUsecase) RunOrchestrationTask(
	id string,
	startOnly *bool,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteOrchestrationTaskRunAction(id, startOnly, traceID)
}

func (u orchestrationTaskUsecase) StopOrchestrationTask(
	ctx context.Context,
	id string,
	req bridgeorchestration.TaskStopParams,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteOrchestrationTaskStopAction(ctx, id, req, traceID)
}

func (u orchestrationTaskUsecase) OrchestrationTaskLogs(
	id string,
	limit *int,
	traceID string,
) (bridgeorchestration.ServiceResult, error) {
	return u.service.ExecuteOrchestrationTaskLogsAction(id, limit, traceID)
}
