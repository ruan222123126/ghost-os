package tools

import (
	"context"
	"fmt"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	bridgetools "ghost-os/bridge/tools"
)

func (s Service) FindIconTemplateUpload(
	req api.FindIconTemplateUploadRequest,
	traceID string,
) (bus.ServiceResult, error) {
	payload, err := ExecuteFindIconTemplateUpload(req)
	if err != nil {
		s.log(traceID, FindIconTemplateUploadAction, "error", err)
		return bus.ServiceResult{}, err
	}
	s.log(traceID, FindIconTemplateUploadAction, "success", nil)
	return bus.ResultCreated(payload), nil
}

func (s Service) FindIconPreview(
	ctx context.Context,
	req api.FindIconPreviewRequest,
	traceID string,
) (bus.ServiceResult, error) {
	params, err := NormalizeFindIconPreviewRequest(req)
	if err != nil {
		err = bus.WrapError(bus.ServiceErrorInvalidInput, err)
		s.log(traceID, FindIconPreviewAction, "error", err)
		return bus.ServiceResult{}, err
	}
	payload, err := s.runScreenControlTool(func(tool bridgetools.Tool) (any, error) {
		return ExecuteFindIconPreviewToolCall(ctx, tool, params, traceID)
	})
	if err != nil {
		s.log(traceID, FindIconPreviewAction, "error", err)
		return bus.ServiceResult{}, err
	}
	s.log(traceID, FindIconPreviewAction, "success", nil)
	return bus.ResultSuccess(payload), nil
}

func (s Service) MousePosition(
	ctx context.Context,
	_ api.MousePositionRequest,
	traceID string,
) (bus.ServiceResult, error) {
	payload, err := s.runScreenControlTool(func(tool bridgetools.Tool) (any, error) {
		return ExecuteMousePositionToolCall(ctx, tool, traceID)
	})
	if err != nil {
		s.log(traceID, MousePositionAction, "error", err)
		return bus.ServiceResult{}, err
	}
	s.log(traceID, MousePositionAction, "success", nil)
	return bus.ResultSuccess(payload), nil
}

func (s Service) runScreenControlTool(execute func(bridgetools.Tool) (any, error)) (any, error) {
	if s.ToolProvider == nil {
		return nil, bus.WrapError(bus.ServiceErrorUnavailable, fmt.Errorf("tool provider is not configured"))
	}
	tool, cleanup, err := s.ToolProvider.Tool(ScreenControlToolID)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return execute(tool)
}
