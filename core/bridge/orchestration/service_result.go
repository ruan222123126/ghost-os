package orchestration

import (
	"context"
	"fmt"
	"net/http"

	apptools "ghost-os/bridge/orchestration/internal/app/tools"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	sharedtext "ghost-os/bridge/orchestration/internal/shared/text"
	bridgeskills "ghost-os/bridge/skills"
	bridgetools "ghost-os/bridge/tools"
)

type ServiceOutcome = bus.ServiceOutcome

const (
	ServiceOutcomeSuccess  = bus.ServiceOutcomeSuccess
	ServiceOutcomeCreated  = bus.ServiceOutcomeCreated
	ServiceOutcomeAccepted = bus.ServiceOutcomeAccepted
)

type ServiceErrorKind = bus.ServiceErrorKind

const (
	ServiceErrorInvalidInput = bus.ServiceErrorInvalidInput
	ServiceErrorNotFound     = bus.ServiceErrorNotFound
	ServiceErrorConflict     = bus.ServiceErrorConflict
	ServiceErrorUnavailable  = bus.ServiceErrorUnavailable
	ServiceErrorInternal     = bus.ServiceErrorInternal
)

type ServiceResult = bus.ServiceResult

func wrapServiceError(kind ServiceErrorKind, err error) error {
	return bus.WrapError(kind, err)
}

func ServiceErrorKindOf(err error) ServiceErrorKind {
	return bus.ErrorKindOf(err)
}

func serviceResultSuccess(payload any) ServiceResult {
	return bus.ResultSuccess(payload)
}

func serviceResultCreated(payload any) ServiceResult {
	return bus.ResultCreated(payload)
}

func serviceResultAccepted(payload any) ServiceResult {
	return bus.ResultAccepted(payload)
}

func serviceResultFromStatus(payload any, statusCode int, err error) (ServiceResult, error) {
	if err != nil {
		return ServiceResult{}, wrapServiceError(serviceErrorKindFromStatus(statusCode), err)
	}
	return serviceResultFromStatusSuccess(payload, statusCode), nil
}

func serviceResultFromStatusSuccess(payload any, statusCode int) ServiceResult {
	switch serviceOutcomeFromStatus(statusCode) {
	case ServiceOutcomeCreated:
		return serviceResultCreated(payload)
	case ServiceOutcomeAccepted:
		return serviceResultAccepted(payload)
	default:
		return serviceResultSuccess(payload)
	}
}

func serviceOutcomeFromStatus(statusCode int) ServiceOutcome {
	switch statusCode {
	case http.StatusCreated:
		return ServiceOutcomeCreated
	case http.StatusAccepted:
		return ServiceOutcomeAccepted
	default:
		return ServiceOutcomeSuccess
	}
}

func serviceErrorKindFromStatus(statusCode int) ServiceErrorKind {
	switch statusCode {
	case http.StatusBadRequest:
		return ServiceErrorInvalidInput
	case http.StatusNotFound:
		return ServiceErrorNotFound
	case http.StatusConflict:
		return ServiceErrorConflict
	case http.StatusServiceUnavailable:
		return ServiceErrorUnavailable
	default:
		return ServiceErrorInternal
	}
}

func serviceResultFromLegacySuccess(payload any, statusCode int) ServiceResult {
	return serviceResultFromStatusSuccess(payload, statusCode)
}

func serviceErrorKindFromLegacyStatus(statusCode int) ServiceErrorKind {
	return serviceErrorKindFromStatus(statusCode)
}

func legacyStatusFromServiceErrorKind(kind ServiceErrorKind) int {
	switch kind {
	case ServiceErrorInvalidInput:
		return http.StatusBadRequest
	case ServiceErrorNotFound:
		return http.StatusNotFound
	case ServiceErrorConflict:
		return http.StatusConflict
	case ServiceErrorUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func legacyStatusFromServiceError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	return legacyStatusFromServiceErrorKind(ServiceErrorKindOf(err))
}

func truncateRunes(value string, limit int) string {
	return sharedtext.TruncateRunes(value, limit)
}

func adaptLegacyResult(payload any, statusCode int, err error) (ServiceResult, error) {
	return serviceResultFromStatus(payload, statusCode, err)
}

const screenControlToolID = apptools.ScreenControlToolID

func (s *bridgeService) toolService() apptools.Service {
	adapter := runtimeToolsAdapter{service: s}
	return apptools.Service{
		Store:          s.configStore,
		SchemaProvider: adapter,
		ToolProvider:   adapter,
		Logger:         serviceActionLogger{},
	}
}

type runtimeToolsAdapter struct {
	service *bridgeService
}

func (a runtimeToolsAdapter) ListToolInputSchemas() (map[string]map[string]any, error) {
	if a.service == nil || a.service.runtimeFactory == nil {
		return nil, nil
	}
	deps, err := a.service.runtimeFactory.Build(a.service.configStore)
	if err != nil {
		return nil, err
	}
	defer deps.Close()
	if deps.registry == nil {
		return map[string]map[string]any{}, nil
	}
	return apptools.CollectSchemas(deps.registry.ToolDefs())
}

func (a runtimeToolsAdapter) Tool(name string) (bridgetools.Tool, func(), error) {
	if a.service == nil || a.service.runtimeFactory == nil {
		return nil, func() {}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("runtime factory is not configured"))
	}
	deps, err := a.service.runtimeFactory.Build(a.service.configStore)
	if err != nil {
		return nil, func() {}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("build runtime dependencies: %w", err))
	}
	if deps.registry == nil {
		deps.Close()
		return nil, func() {}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("tool registry is not configured"))
	}
	tool := deps.registry.Get(name)
	if tool == nil {
		deps.Close()
		return nil, func() {}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("tool %q is not available", name))
	}
	return tool, deps.Close, nil
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

func (s *bridgeService) executeTaskCreateActionResult(params taskCreateParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskCreateAction(params, traceID))
}

func (s *bridgeService) executeTaskListActionResult(scope string, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskListAction(scope, traceID))
}

func (s *bridgeService) executeTaskGetActionResult(params taskIDParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskGetAction(params, traceID))
}

func (s *bridgeService) executeTaskUpdateActionResult(params taskUpdateParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskUpdateAction(params, traceID))
}

func (s *bridgeService) executeTaskRunNowActionResult(params taskIDParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskRunNowAction(params, traceID))
}

func (s *bridgeService) executeTaskStopActionResult(
	ctx context.Context,
	params taskStopParams,
	traceID string,
) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskStopAction(ctx, params, traceID))
}

func (s *bridgeService) executeTaskLogsActionResult(params taskLogsParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskLogsAction(params, traceID))
}

func (s *bridgeService) executeTaskDeleteActionResult(params taskIDParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeTaskDeleteAction(params, traceID))
}

func (s *bridgeService) executeSkillListActionResult(traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeSkillListAction(traceID))
}

func (s *bridgeService) executeSkillUpdateActionResult(
	params bridgeskills.SkillIDParams,
	req bridgeskills.SkillUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return adaptLegacyResult(s.executeSkillUpdateAction(params, req, traceID))
}

func (s *bridgeService) executeSkillDeleteActionResult(params bridgeskills.SkillIDParams, traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeSkillDeleteAction(params, traceID))
}

func (s *bridgeService) executeToolListActionResult(traceID string) (ServiceResult, error) {
	return adaptLegacyResult(s.executeToolListAction(traceID))
}

func (s *bridgeService) executeToolUpdateActionResult(
	params toolNameParams,
	req toolUpdateRequest,
	traceID string,
) (ServiceResult, error) {
	return adaptLegacyResult(s.executeToolUpdateAction(params, req, traceID))
}
