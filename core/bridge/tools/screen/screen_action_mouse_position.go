package screen

import (
	"context"

	"ghost-os/bridge/tools/internal/tooljson"
)

type screenMousePositionPayload struct {
	X         int      `json:"x"`
	Y         int      `json:"y"`
	DisplayID *int     `json:"display_id,omitempty"`
	ScaleX    *float64 `json:"scale_x,omitempty"`
	ScaleY    *float64 `json:"scale_y,omitempty"`
}

func (t *ScreenActionTool) queryMousePosition(
	ctx context.Context,
	traceID string,
) (screenMousePositionPayload, error) {
	payload, err := t.execution.Call(ctx, "MOUSE_POSITION", map[string]any{}, traceID)
	if err != nil {
		return screenMousePositionPayload{}, wrapNativeExecutionErrorWithRebuildHint(
			"MOUSE_POSITION",
			err,
			mousePositionUnsupportedActionMessage,
		)
	}
	return tooljson.DecodePayload[screenMousePositionPayload](payload)
}

func (t *ScreenActionTool) executeMousePosition(
	ctx context.Context,
	traceID string,
) (string, error) {
	position, err := t.queryMousePosition(ctx, traceID)
	if err != nil {
		return "", err
	}
	return tooljson.Encode(map[string]any{
		"action":     "mouse_position",
		"x":          position.X,
		"y":          position.Y,
		"display_id": position.DisplayID,
		"scale_x":    position.ScaleX,
		"scale_y":    position.ScaleY,
	})
}
