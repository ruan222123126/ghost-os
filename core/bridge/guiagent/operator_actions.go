package guiagent

import (
	"context"
	"strings"
)

func executeClick(
	o *DesktopOperator,
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	return o.executeMouseAction(ctx, "MOUSE_CLICK", request, observation, action.Target, traceID)
}

func executeDoubleClick(
	o *DesktopOperator,
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	return o.executeMouseAction(ctx, "MOUSE_DOUBLE_CLICK", request, observation, action.Target, traceID)
}

func executeRightClick(
	o *DesktopOperator,
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	return o.executeMouseAction(ctx, "MOUSE_RIGHT_CLICK", request, observation, action.Target, traceID)
}

func executeTypeText(
	o *DesktopOperator,
	ctx context.Context,
	_ Request,
	_ Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	return o.execution.Call(ctx, "TEXT_INPUT", map[string]any{"text": action.Text}, traceID)
}

func executeHotkey(
	o *DesktopOperator,
	ctx context.Context,
	_ Request,
	_ Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	return o.execution.Call(ctx, "KEY_HOTKEY", map[string]any{"keys": action.Keys}, traceID)
}

func executeScrollAction(
	o *DesktopOperator,
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	return o.executeScroll(ctx, request, observation, action, traceID)
}

func executeDragAction(
	o *DesktopOperator,
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	return o.executeDrag(ctx, request, observation, action, traceID)
}

func (o *DesktopOperator) executeMouseAction(
	ctx context.Context,
	actionName string,
	request Request,
	observation Observation,
	target *ActionTarget,
	traceID string,
) (map[string]any, error) {
	point, err := absoluteCenterPoint(observation, target)
	if err != nil {
		return nil, err
	}
	params := mouseParamsFromPoint(point, request, observation.DisplayID)
	return o.execution.Call(ctx, actionName, params, traceID)
}

func (o *DesktopOperator) executeScroll(
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	params := map[string]any{
		"delta_x": action.DeltaX,
		"delta_y": action.DeltaY,
	}
	if action.Target != nil && action.Target.Box != nil {
		point, err := absoluteCenterPoint(observation, action.Target)
		if err != nil {
			return nil, err
		}
		params["x"] = point.X
		params["y"] = point.Y
	}
	if request.DisplayID != nil {
		params["display_id"] = *request.DisplayID
	}
	return o.execution.Call(ctx, "MOUSE_SCROLL", params, traceID)
}

func (o *DesktopOperator) executeDrag(
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	traceID string,
) (map[string]any, error) {
	start, err := absoluteCenterPoint(observation, action.Target)
	if err != nil {
		return nil, err
	}
	end, err := absoluteCenterPoint(observation, action.Destination)
	if err != nil {
		return nil, err
	}
	params := map[string]any{
		"start_x": start.X,
		"start_y": start.Y,
		"end_x":   end.X,
		"end_y":   end.Y,
	}
	if action.DurationMs > 0 {
		params["duration_ms"] = action.DurationMs
	}
	appendWindowConstraints(params, request)
	if request.DisplayID != nil {
		params["display_id"] = *request.DisplayID
	}
	return o.execution.Call(ctx, "MOUSE_DRAG", params, traceID)
}

type absolutePoint struct {
	X int
	Y int
}

func absoluteCenterPoint(observation Observation, target *ActionTarget) (absolutePoint, error) {
	if target == nil || target.Box == nil {
		return absolutePoint{}, &RunError{Code: ErrorInvalidTargetBox, Message: "target.box is required"}
	}
	box := *target.Box
	centerX := int(((box[0] + box[2]) / 2) * float64(observation.ImageWidth))
	centerY := int(((box[1] + box[3]) / 2) * float64(observation.ImageHeight))
	return absolutePoint{
		X: observation.OriginX + centerX,
		Y: observation.OriginY + centerY,
	}, nil
}

func mouseParamsFromPoint(point absolutePoint, request Request, displayID int) map[string]any {
	params := map[string]any{
		"x": point.X,
		"y": point.Y,
	}
	appendWindowConstraints(params, request)
	if request.DisplayID != nil {
		params["display_id"] = *request.DisplayID
	} else if displayID >= 0 {
		params["display_id"] = displayID
	}
	return params
}

func appendWindowConstraints(params map[string]any, request Request) {
	if title := strings.TrimSpace(request.Target.WindowTitle); title != "" {
		params["ensure_active_window_title"] = title
	}
	if className := strings.TrimSpace(request.Target.WindowClass); className != "" {
		params["ensure_active_window_class"] = className
	}
}
