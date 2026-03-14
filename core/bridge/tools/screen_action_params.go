package tools

import (
	"fmt"

	"ghost-os/bridge/tools/internal/toolparams"
)

func parseOptionalPoint(params map[string]any) (screenPoint, bool, error) {
	if len(params) == 0 {
		return screenPoint{}, false, nil
	}

	x, hasX := toolparams.OptionalInt(params, "x")
	y, hasY := toolparams.OptionalInt(params, "y")
	if !hasX && !hasY {
		return screenPoint{}, false, nil
	}
	if !hasX || !hasY {
		return screenPoint{}, false, fmt.Errorf("x and y must both be provided for direct click")
	}
	return screenPoint{X: x, Y: y}, true, nil
}

func parseOptionalRegion(params map[string]any) (screenRegion, bool, error) {
	if len(params) == 0 {
		return screenRegion{}, false, nil
	}
	raw, ok := params["region"]
	if !ok || raw == nil {
		return screenRegion{}, false, nil
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return screenRegion{}, false, fmt.Errorf("region must be an object")
	}

	x, ok := toolparams.Int(obj["x"])
	if !ok {
		return screenRegion{}, false, fmt.Errorf("region.x must be a number")
	}
	y, ok := toolparams.Int(obj["y"])
	if !ok {
		return screenRegion{}, false, fmt.Errorf("region.y must be a number")
	}
	width, ok := toolparams.Int(obj["width"])
	if !ok || width <= 0 {
		return screenRegion{}, false, fmt.Errorf("region.width must be a positive number")
	}
	height, ok := toolparams.Int(obj["height"])
	if !ok || height <= 0 {
		return screenRegion{}, false, fmt.Errorf("region.height must be a positive number")
	}

	return screenRegion{X: x, Y: y, Width: width, Height: height}, true, nil
}

func appendActiveWindowConstraints(params map[string]any, payload map[string]any) {
	if title := toolparams.OptionalString(params, "ensure_active_window_title", ""); title != "" {
		payload["ensure_active_window_title"] = title
	}
	if className := toolparams.OptionalString(params, "ensure_active_window_class", ""); className != "" {
		payload["ensure_active_window_class"] = className
	}
}

func appendDisplayScaleFromOCR(payload screenOCRPayload, clickPayload map[string]any) {
	clickPayload["display_id"] = payload.DisplayID
	appendScale(clickPayload, payload.ScaleX, payload.ScaleY)
}

func appendDisplayScaleFromIcon(payload iconMatchPayload, clickPayload map[string]any) {
	clickPayload["display_id"] = payload.DisplayID
	appendScale(clickPayload, payload.ScaleX, payload.ScaleY)
}

func appendDisplayScaleFromParams(params map[string]any, payload map[string]any) {
	if displayID, ok := toolparams.OptionalInt(params, "display_id"); ok {
		payload["display_id"] = displayID
	}
	scaleX, hasScaleX := toolparams.OptionalFloat(params, "scale_x")
	scaleY, hasScaleY := toolparams.OptionalFloat(params, "scale_y")
	if hasScaleX || hasScaleY {
		appendScale(payload, scaleX, scaleY)
	}
}

func appendScale(payload map[string]any, scaleX float64, scaleY float64) {
	if scaleX > 0 {
		payload["scale_x"] = scaleX
	}
	if scaleY > 0 {
		payload["scale_y"] = scaleY
	}
}
