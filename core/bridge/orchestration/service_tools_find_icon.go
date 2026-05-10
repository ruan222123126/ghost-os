package orchestration

import (
	"context"
	"fmt"

	apptools "ghost-os/bridge/orchestration/internal/app/tools"
	"ghost-os/bridge/tools"
)

const (
	findIconTemplateUploadAction = apptools.FindIconTemplateUploadAction
	findIconPreviewAction        = apptools.FindIconPreviewAction
	findIconTemplateRoot         = apptools.FindIconTemplateRoot
	screenControlToolID          = apptools.ScreenControlToolID
)

func (s *bridgeService) executeFindIconTemplateUploadActionResult(
	req findIconTemplateUploadRequest,
	traceID string,
) (ServiceResult, error) {
	payload, err := executeFindIconTemplateUpload(req)
	if err != nil {
		logAction(traceID, findIconTemplateUploadAction, "error", err)
		return ServiceResult{}, err
	}
	logAction(traceID, findIconTemplateUploadAction, "success", nil)
	return serviceResultCreated(payload), nil
}

func executeFindIconTemplateUpload(req findIconTemplateUploadRequest) (findIconTemplateUploadPayload, error) {
	return apptools.ExecuteFindIconTemplateUpload(req)
}

func resolveFindIconTemplateRoot() (string, error) {
	return apptools.ResolveFindIconTemplateRoot()
}

func (s *bridgeService) executeFindIconPreviewActionResult(
	ctx context.Context,
	req findIconPreviewRequest,
	traceID string,
) (ServiceResult, error) {
	params, err := normalizeFindIconPreviewRequest(req)
	if err != nil {
		logAction(traceID, findIconPreviewAction, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	payload, err := s.runFindIconPreview(ctx, params, traceID)
	if err != nil {
		logAction(traceID, findIconPreviewAction, "error", err)
		return ServiceResult{}, err
	}
	logAction(traceID, findIconPreviewAction, "success", nil)
	return serviceResultSuccess(payload), nil
}

func normalizeFindIconPreviewRequest(req findIconPreviewRequest) (findIconPreviewRequest, error) {
	return apptools.NormalizeFindIconPreviewRequest(req)
}

func (s *bridgeService) runFindIconPreview(
	ctx context.Context,
	req findIconPreviewRequest,
	traceID string,
) (findIconPreviewPayload, error) {
	tool, cleanup, err := s.screenControlTool()
	if err != nil {
		return findIconPreviewPayload{}, err
	}
	defer cleanup()
	return executeFindIconPreviewToolCall(ctx, tool, req, traceID)
}

func (s *bridgeService) screenControlTool() (tools.Tool, func(), error) {
	if s == nil || s.runtimeFactory == nil {
		return nil, func() {}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("runtime factory is not configured"))
	}
	deps, err := s.runtimeFactory.Build(s.configStore)
	if err != nil {
		return nil, func() {}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("build runtime dependencies: %w", err))
	}
	if deps.registry == nil {
		deps.Close()
		return nil, func() {}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("tool registry is not configured"))
	}
	tool := deps.registry.Get(screenControlToolID)
	if tool == nil {
		deps.Close()
		return nil, func() {}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("tool %q is not available", screenControlToolID))
	}
	return tool, deps.Close, nil
}

func executeFindIconPreviewToolCall(
	ctx context.Context,
	tool tools.Tool,
	req findIconPreviewRequest,
	traceID string,
) (findIconPreviewPayload, error) {
	return apptools.ExecuteFindIconPreviewToolCall(ctx, tool, req, traceID)
}
