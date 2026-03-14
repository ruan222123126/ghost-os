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

func appendDisplayIDFromOCR(payload screenOCRPayload, clickPayload map[string]any) {
	if payload.DisplayID > 0 {
		clickPayload["display_id"] = payload.DisplayID
	}
}

func appendDisplayIDFromIcon(payload iconMatchPayload, clickPayload map[string]any) {
	if payload.DisplayID > 0 {
		clickPayload["display_id"] = payload.DisplayID
	}
}

func appendDisplayIDFromParams(params map[string]any, payload map[string]any) {
	if displayID, ok := toolparams.OptionalInt(params, "display_id"); ok {
		payload["display_id"] = displayID
	}
}

func buildScreenCaptureParams(params map[string]any) (map[string]any, error) {
	payload := map[string]any{}
	if displayID, ok := toolparams.OptionalInt(params, "display_id"); ok {
		payload["display_id"] = displayID
	}
	if region, ok, err := parseOptionalRegion(params); err != nil {
		return nil, err
	} else if ok {
		payload["region"] = region
	}
	return payload, nil
}

func buildOCRImageParams(capture screenCapturePayload, params map[string]any) map[string]any {
	payload := map[string]any{
		"image_base64": capture.ImageBase64,
		"origin_x":     capture.OriginX,
		"origin_y":     capture.OriginY,
	}
	if raw, ok := params["languages"]; ok {
		payload["languages"] = raw
	}
	if confidence, ok := toolparams.OptionalFloat(params, "min_confidence"); ok {
		payload["min_confidence"] = confidence
	}
	return payload
}

func buildTemplateMatchParams(capture screenCapturePayload, params map[string]any) map[string]any {
	payload := map[string]any{
		"image_base64":  capture.ImageBase64,
		"origin_x":      capture.OriginX,
		"origin_y":      capture.OriginY,
		"template_path": toolparams.OptionalString(params, "template_path", ""),
	}
	if threshold, ok := toolparams.OptionalFloat(params, "threshold"); ok {
		payload["threshold"] = threshold
	}
	if maxResults, ok := toolparams.OptionalInt(params, "max_results"); ok {
		payload["max_results"] = maxResults
	}
	if raw, ok := params["scale_range"]; ok && raw != nil {
		payload["scale_range"] = raw
	}
	return payload
}
