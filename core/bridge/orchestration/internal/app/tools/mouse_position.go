package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	bridgetools "ghost-os/bridge/tools"
)

const MousePositionAction = "MOUSE_POSITION_QUERY"

func ExecuteMousePositionToolCall(
	ctx context.Context,
	tool bridgetools.Tool,
	traceID string,
) (api.MousePositionPayload, error) {
	args, err := json.Marshal(BuildMousePositionToolArgs())
	if err != nil {
		return api.MousePositionPayload{}, bus.WrapError(bus.ServiceErrorInternal, fmt.Errorf("encode mouse_position args: %w", err))
	}
	output, err := tool.Execute(bridgetools.WithToolCallID(ctx, "mouse-position"), args, traceID)
	if err != nil {
		return api.MousePositionPayload{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	payload, err := DecodeMousePositionPayload(output)
	if err != nil {
		return api.MousePositionPayload{}, bus.WrapError(bus.ServiceErrorInternal, err)
	}
	return payload, nil
}

func BuildMousePositionToolArgs() map[string]any {
	return map[string]any{"mode": "atomic", "action": "mouse_position"}
}

func DecodeMousePositionPayload(raw string) (api.MousePositionPayload, error) {
	var decoded map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &decoded); err != nil {
		return api.MousePositionPayload{}, fmt.Errorf("decode mouse_position output: %w", err)
	}
	x, ok := ParseOptionalMousePositionInt(decoded["x"])
	if !ok {
		return api.MousePositionPayload{}, fmt.Errorf("mouse_position output field x must be a number")
	}
	y, ok := ParseOptionalMousePositionInt(decoded["y"])
	if !ok {
		return api.MousePositionPayload{}, fmt.Errorf("mouse_position output field y must be a number")
	}
	return buildMousePositionPayload(decoded, x, y), nil
}

func buildMousePositionPayload(decoded map[string]any, x int, y int) api.MousePositionPayload {
	out := api.MousePositionPayload{X: x, Y: y}
	if displayID, ok := ParseOptionalMousePositionInt(decoded["display_id"]); ok {
		out.DisplayID = &displayID
	}
	if scaleX, ok := ParseOptionalMousePositionFloat(decoded["scale_x"]); ok {
		out.ScaleX = &scaleX
	}
	if scaleY, ok := ParseOptionalMousePositionFloat(decoded["scale_y"]); ok {
		out.ScaleY = &scaleY
	}
	return out
}

func ParseOptionalMousePositionInt(raw any) (int, bool) {
	value, ok := raw.(float64)
	if !ok {
		return 0, false
	}
	return int(value), true
}

func ParseOptionalMousePositionFloat(raw any) (float64, bool) {
	value, ok := raw.(float64)
	if !ok {
		return 0, false
	}
	return value, true
}
