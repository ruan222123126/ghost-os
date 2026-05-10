package orchestration

import (
	"context"

	apptools "ghost-os/bridge/orchestration/internal/app/tools"
	"ghost-os/bridge/tools"
)

const mousePositionAction = apptools.MousePositionAction

func (s *bridgeService) executeMousePositionActionResult(
	ctx context.Context,
	_ mousePositionRequest,
	traceID string,
) (ServiceResult, error) {
	payload, err := s.runMousePosition(ctx, traceID)
	if err != nil {
		logAction(traceID, mousePositionAction, "error", err)
		return ServiceResult{}, err
	}
	logAction(traceID, mousePositionAction, "success", nil)
	return serviceResultSuccess(payload), nil
}

func (s *bridgeService) runMousePosition(
	ctx context.Context,
	traceID string,
) (mousePositionPayload, error) {
	tool, cleanup, err := s.screenControlTool()
	if err != nil {
		return mousePositionPayload{}, err
	}
	defer cleanup()
	return executeMousePositionToolCall(ctx, tool, traceID)
}

func executeMousePositionToolCall(
	ctx context.Context,
	tool tools.Tool,
	traceID string,
) (mousePositionPayload, error) {
	return apptools.ExecuteMousePositionToolCall(ctx, tool, traceID)
}

func buildMousePositionToolArgs() map[string]any {
	return apptools.BuildMousePositionToolArgs()
}

func decodeMousePositionPayload(raw string) (mousePositionPayload, error) {
	return apptools.DecodeMousePositionPayload(raw)
}

func parseOptionalMousePositionInt(raw any) (int, bool) {
	return apptools.ParseOptionalMousePositionInt(raw)
}

func parseOptionalMousePositionFloat(raw any) (float64, bool) {
	return apptools.ParseOptionalMousePositionFloat(raw)
}
