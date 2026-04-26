package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/tools"
)

const mousePositionAction = "MOUSE_POSITION_QUERY"

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
	if s == nil || s.runtimeFactory == nil {
		return mousePositionPayload{}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("runtime factory is not configured"))
	}
	deps, err := s.runtimeFactory.Build(s.configStore)
	if err != nil {
		return mousePositionPayload{}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("build runtime dependencies: %w", err))
	}
	defer deps.Close()
	if deps.registry == nil {
		return mousePositionPayload{}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("tool registry is not configured"))
	}
	tool := deps.registry.Get(screenControlToolID)
	if tool == nil {
		return mousePositionPayload{}, wrapServiceError(ServiceErrorUnavailable, fmt.Errorf("tool %q is not available", screenControlToolID))
	}
	return executeMousePositionToolCall(ctx, tool, traceID)
}

func executeMousePositionToolCall(
	ctx context.Context,
	tool tools.Tool,
	traceID string,
) (mousePositionPayload, error) {
	args, err := json.Marshal(buildMousePositionToolArgs())
	if err != nil {
		return mousePositionPayload{}, wrapServiceError(ServiceErrorInternal, fmt.Errorf("encode mouse_position args: %w", err))
	}
	output, err := tool.Execute(tools.WithToolCallID(ctx, "mouse-position"), args, traceID)
	if err != nil {
		return mousePositionPayload{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	payload, err := decodeMousePositionPayload(output)
	if err != nil {
		return mousePositionPayload{}, wrapServiceError(ServiceErrorInternal, err)
	}
	return payload, nil
}

func buildMousePositionToolArgs() map[string]any {
	return map[string]any{
		"mode":   "atomic",
		"action": "mouse_position",
	}
}

func decodeMousePositionPayload(raw string) (mousePositionPayload, error) {
	var decoded map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &decoded); err != nil {
		return mousePositionPayload{}, fmt.Errorf("decode mouse_position output: %w", err)
	}
	x, ok := parseOptionalMousePositionInt(decoded["x"])
	if !ok {
		return mousePositionPayload{}, fmt.Errorf("mouse_position output field x must be a number")
	}
	y, ok := parseOptionalMousePositionInt(decoded["y"])
	if !ok {
		return mousePositionPayload{}, fmt.Errorf("mouse_position output field y must be a number")
	}
	out := mousePositionPayload{X: x, Y: y}
	if displayID, ok := parseOptionalMousePositionInt(decoded["display_id"]); ok {
		out.DisplayID = &displayID
	}
	if scaleX, ok := parseOptionalMousePositionFloat(decoded["scale_x"]); ok {
		out.ScaleX = &scaleX
	}
	if scaleY, ok := parseOptionalMousePositionFloat(decoded["scale_y"]); ok {
		out.ScaleY = &scaleY
	}
	return out, nil
}

func parseOptionalMousePositionInt(raw any) (int, bool) {
	value, ok := raw.(float64)
	if !ok {
		return 0, false
	}
	return int(value), true
}

func parseOptionalMousePositionFloat(raw any) (float64, bool) {
	value, ok := raw.(float64)
	if !ok {
		return 0, false
	}
	return value, true
}
