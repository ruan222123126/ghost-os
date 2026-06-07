package orchestration

import (
	"context"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/orchestration/internal/adapters/promptpreview"
	"ghost-os/bridge/orchestration/internal/adapters/toolruntime"
	appconfig "ghost-os/bridge/orchestration/internal/app/config"
	appdownloads "ghost-os/bridge/orchestration/internal/app/downloads"
	appprompts "ghost-os/bridge/orchestration/internal/app/prompts"
	appskills "ghost-os/bridge/orchestration/internal/app/skills"
	apptools "ghost-os/bridge/orchestration/internal/app/tools"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	bridgeskills "ghost-os/bridge/skills"
)

const (
	systemPromptActionGet    = appprompts.SystemPromptActionGet
	systemPromptActionUpdate = appprompts.SystemPromptActionUpdate
	presetActionList         = appprompts.PresetActionList
	presetActionCreate       = appprompts.PresetActionCreate
	presetActionUpdate       = appprompts.PresetActionUpdate
	presetActionDelete       = appprompts.PresetActionDelete
	presetActionApply        = appprompts.PresetActionApply
)

type systemPromptResponse = appprompts.SystemResponse

type systemPromptToolDefinition = appprompts.ToolDefinition

type systemPromptPreview = appprompts.Preview

const screenControlToolID = apptools.ScreenControlToolID

func (s *bridgeService) promptService() appprompts.Service {
	return appprompts.Service{
		Store: s.configStore,
		PreviewLoader: promptpreview.Loader{
			Store:          s.configStore,
			RuntimeFactory: s.promptPreviewRuntimeFactory(),
		},
		Logger: serviceActionLogger{},
	}
}

type serviceActionLogger struct{}

func (serviceActionLogger) Log(traceID string, action string, status string, err error) {
	logAction(traceID, action, status, err)
}

func (s *bridgeService) promptPreviewRuntimeFactory() promptpreview.RuntimeFactory {
	if s == nil || s.runtimeFactory == nil {
		return nil
	}
	return promptpreview.RuntimeFactoryFunc(func(store bridgeconfig.Store) (promptpreview.RuntimeDependencies, error) {
		return s.runtimeFactory.Build(store)
	})
}

func (s *bridgeService) toolService() apptools.Service {
	adapter := toolruntime.Provider{
		Store:          s.configStore,
		RuntimeFactory: s.toolRuntimeFactory(),
	}
	return apptools.Service{
		Store:          s.configStore,
		SchemaProvider: adapter,
		ToolProvider:   adapter,
		Logger:         serviceActionLogger{},
	}
}

func (s *bridgeService) toolRuntimeFactory() toolruntime.RuntimeFactory {
	if s == nil || s.runtimeFactory == nil {
		return nil
	}
	return toolruntime.RuntimeFactoryFunc(func(store bridgeconfig.Store) (toolruntime.RuntimeDependencies, error) {
		return s.runtimeFactory.Build(store)
	})
}

func (s *bridgeService) executeSystemPromptGetAction(traceID string) (ServiceResult, error) {
	return s.promptService().GetSystemPrompt(traceID)
}

func (s *bridgeService) executeSystemPromptUpdateAction(
	req bridgeconfig.SystemPromptUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return s.promptService().UpdateSystemPrompt(req, traceID)
}

func (s *bridgeService) executePresetListAction(traceID string) (ServiceResult, error) {
	return s.promptService().ListPresets(traceID)
}

func (s *bridgeService) executePresetCreateAction(
	req bridgeconfig.PresetCreateRequest,
	traceID string,
) (ServiceResult, error) {
	return s.promptService().CreatePreset(req, traceID)
}

func (s *bridgeService) executePresetUpdateAction(
	presetID string,
	req bridgeconfig.PresetUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return s.promptService().UpdatePreset(presetID, req, traceID)
}

func (s *bridgeService) executePresetDeleteAction(
	presetID string,
	traceID string,
) (ServiceResult, error) {
	return s.promptService().DeletePreset(presetID, traceID)
}

func (s *bridgeService) executePresetApplyAction(
	presetID string,
	traceID string,
) (ServiceResult, error) {
	return s.promptService().ApplyPreset(presetID, traceID)
}

func (s *bridgeService) executeSkillListAction(traceID string) (any, int, error) {
	return s.skillService().List(traceID)
}

func (s *bridgeService) executeSkillUpdateAction(
	params bridgeskills.SkillIDParams,
	req bridgeskills.SkillUpdateRequest,
	traceID string,
) (any, int, error) {
	return s.skillService().Update(params, req, traceID)
}

func (s *bridgeService) executeSkillDeleteAction(params bridgeskills.SkillIDParams, traceID string) (any, int, error) {
	return s.skillService().Delete(params, traceID)
}

func (s *bridgeService) executeSkillListActionResult(traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.executeSkillListAction(traceID))
}

func (s *bridgeService) executeSkillUpdateActionResult(
	params bridgeskills.SkillIDParams,
	req bridgeskills.SkillUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return bus.ResultFromStatus(s.executeSkillUpdateAction(params, req, traceID))
}

func (s *bridgeService) executeSkillDeleteActionResult(params bridgeskills.SkillIDParams, traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.executeSkillDeleteAction(params, traceID))
}

func (s *bridgeService) skillService() appskills.Service {
	return appskills.Service{Handler: s.skillHandler}
}

func (s *bridgeService) executeToolListAction(traceID string) (any, int, error) {
	return s.toolService().List(traceID)
}

func (s *bridgeService) executeToolUpdateAction(
	params toolNameParams,
	req toolUpdateRequest,
	traceID string,
) (any, int, error) {
	return s.toolService().Update(params, req, traceID)
}

func (s *bridgeService) executeFindIconTemplateUploadActionResult(
	req findIconTemplateUploadRequest,
	traceID string,
) (ServiceResult, error) {
	return s.toolService().FindIconTemplateUpload(req, traceID)
}

func (s *bridgeService) executeFindIconPreviewActionResult(
	ctx context.Context,
	req findIconPreviewRequest,
	traceID string,
) (ServiceResult, error) {
	return s.toolService().FindIconPreview(ctx, req, traceID)
}

func (s *Service) DownloadFindIconTemplate(templatePath string) (BinaryDownload, error) {
	return appdownloads.OpenFindIconTemplate(templatePath)
}

func (s *bridgeService) executeMousePositionActionResult(
	ctx context.Context,
	req mousePositionRequest,
	traceID string,
) (ServiceResult, error) {
	return s.toolService().MousePosition(ctx, req, traceID)
}

func resolveFindIconTemplateRoot() (string, error) {
	return apptools.ResolveFindIconTemplateRoot()
}

func (s *bridgeService) executeToolListActionResult(traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.executeToolListAction(traceID))
}

func (s *bridgeService) executeToolUpdateActionResult(
	params toolNameParams,
	req toolUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return bus.ResultFromStatus(s.executeToolUpdateAction(params, req, traceID))
}

// executeConfigGetAction 返回当前可编辑配置快照，不暴露敏感明文字段。
func (s *bridgeService) executeConfigGetAction(traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.configService().GetRuntime(traceID))
}

// executeConfigUpdateAction 按请求局部更新运行态配置，并返回更新后可编辑快照。
func (s *bridgeService) executeConfigUpdateAction(req configUpdateRequest, traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.configService().UpdateRuntime(req, traceID, s.afterRuntimeConfigUpdate))
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

func (s *bridgeService) configService() appconfig.Service {
	return appconfig.Service{
		Store:  s.configStore,
		Logger: serviceActionLogger{},
	}
}

func (s *bridgeService) afterRuntimeConfigUpdate() error {
	s.syncTaskSchedulerExecutionTimeout()
	return s.BootstrapSystemTasks()
}

func (s *bridgeService) executeProvidersGetAction(traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.configService().List(traceID))
}

func (s *bridgeService) executeProviderCreateAction(req providerCreateRequest, traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.configService().Create(req, traceID))
}

func (s *bridgeService) executeProviderUpdateAction(
	name string,
	req providerUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return bus.ResultFromStatus(s.configService().Update(name, req, traceID))
}

func (s *bridgeService) executeProviderDeleteAction(name string, traceID string) (ServiceResult, error) {
	return bus.ResultFromStatus(s.configService().Delete(name, traceID))
}

func (s *bridgeService) executeSetActiveProviderAction(
	req setActiveProviderRequest,
	traceID string,
) (ServiceResult, error) {
	return bus.ResultFromStatus(s.configService().SetActive(req, traceID))
}

func configResponseFromSnapshot(snapshot bridgeconfig.Snapshot) configResponse {
	return appconfig.ResponseFromSnapshot(snapshot)
}
