package screen

import (
	"context"
	"fmt"

	"ghost-os/bridge/tools/internal/tooljson"
	"ghost-os/bridge/tools/internal/toolparams"
)

func (t *ScreenActionTool) executeFindIcon(ctx context.Context, params map[string]any, traceID string) (string, error) {
	payload, err := t.captureAndMatchIcon(ctx, params, traceID)
	if err != nil {
		return "", err
	}
	return tooljson.Encode(buildFindIconPayload(payload, params))
}

func (t *ScreenActionTool) executeClickIcon(ctx context.Context, params map[string]any, traceID string) (string, error) {
	if shouldHoverOnly(params) {
		return t.executeHoverIcon(ctx, params, traceID)
	}
	if request, ok, err := parseDirectClickRequest(params); err != nil {
		return "", err
	} else if ok {
		logClickIcon(traceID, true, params)
		return t.clickDirectPoint(ctx, traceID, request)
	}
	logClickIcon(traceID, false, params)

	iconPayload, err := t.captureAndMatchIcon(ctx, params, traceID)
	if err != nil {
		return "", err
	}
	selected, err := selectIconMatch(iconPayload, params)
	if err != nil {
		return "", err
	}
	clickPayload := buildIconClickPayload(selected, iconPayload, params)
	if _, err := t.execution.Call(ctx, "MOUSE_CLICK", clickPayload, traceID); err != nil {
		return "", fmt.Errorf("execution MOUSE_CLICK failed: %w", err)
	}
	return tooljson.Encode(map[string]any{
		"action":          "click_icon",
		"template_path":   toolparams.OptionalString(params, "template_path", ""),
		"candidate_count": len(iconPayload.Matches),
		"clicked":         true,
		"selected":        selected,
	})
}

func shouldHoverOnly(params map[string]any) bool {
	return toolparams.OptionalBool(params, "hover_only", false)
}

func (t *ScreenActionTool) executeHoverIcon(ctx context.Context, params map[string]any, traceID string) (string, error) {
	point, hasPoint, err := parseOptionalPoint(params)
	if err != nil {
		return "", err
	}
	if hasPoint {
		logClickIcon(traceID, true, params)
		return t.hoverDirectPoint(ctx, traceID, point, params)
	}
	logClickIcon(traceID, false, params)
	iconPayload, err := t.captureAndMatchIcon(ctx, params, traceID)
	if err != nil {
		return "", err
	}
	selected, err := selectIconMatch(iconPayload, params)
	if err != nil {
		return "", err
	}
	return t.hoverMatchedPoint(ctx, traceID, selected, iconPayload, params)
}

func (t *ScreenActionTool) hoverDirectPoint(
	ctx context.Context,
	traceID string,
	point screenPoint,
	params map[string]any,
) (string, error) {
	movePayload := map[string]any{
		"x": point.X,
		"y": point.Y,
	}
	appendDisplayIDFromParams(params, movePayload)
	appendActiveWindowConstraints(params, movePayload)
	if _, err := t.execution.Call(ctx, "MOUSE_MOVE", movePayload, traceID); err != nil {
		return "", fmt.Errorf("execution MOUSE_MOVE failed: %w", err)
	}
	return tooljson.Encode(map[string]any{
		"action":        "click_icon",
		"template_path": toolparams.OptionalString(params, "template_path", ""),
		"hovered":       true,
		"direct":        true,
		"selected": map[string]any{
			"center": point,
		},
	})
}

func (t *ScreenActionTool) hoverMatchedPoint(
	ctx context.Context,
	traceID string,
	selected iconMatch,
	payload iconMatchPayload,
	params map[string]any,
) (string, error) {
	movePayload := map[string]any{
		"x": selected.Center.X,
		"y": selected.Center.Y,
	}
	appendDisplayIDFromIcon(payload, movePayload)
	appendActiveWindowConstraints(params, movePayload)
	if _, err := t.execution.Call(ctx, "MOUSE_MOVE", movePayload, traceID); err != nil {
		return "", fmt.Errorf("execution MOUSE_MOVE failed: %w", err)
	}
	return tooljson.Encode(map[string]any{
		"action":          "click_icon",
		"template_path":   toolparams.OptionalString(params, "template_path", ""),
		"candidate_count": len(payload.Matches),
		"hovered":         true,
		"selected":        selected,
	})
}

func (t *ScreenActionTool) clickDirectPoint(ctx context.Context, traceID string, request directClickRequest) (string, error) {
	clickPayload := map[string]any{
		"x": request.Point.X,
		"y": request.Point.Y,
	}
	if request.Button != "" {
		clickPayload["button"] = request.Button
	}
	appendDisplayIDFromParams(request.Extra, clickPayload)
	appendActiveWindowConstraints(request.Extra, clickPayload)
	if _, err := t.execution.Call(ctx, "MOUSE_CLICK", clickPayload, traceID); err != nil {
		return "", fmt.Errorf("execution MOUSE_CLICK failed: %w", err)
	}

	result := map[string]any{
		"action":  request.Action,
		"clicked": true,
		"direct":  true,
		"selected": map[string]any{
			"center": request.Point,
		},
	}
	for key, value := range request.Extra {
		switch typed := value.(type) {
		case string:
			if typed == "" {
				continue
			}
		case nil:
			continue
		}
		result[key] = value
	}
	return tooljson.Encode(result)
}

func parseDirectClickRequest(params map[string]any) (directClickRequest, bool, error) {
	point, ok, err := parseOptionalPoint(params)
	if err != nil || !ok {
		return directClickRequest{}, ok, err
	}
	extra := map[string]any{
		"template_path": toolparams.OptionalString(params, "template_path", ""),
	}
	appendDisplayIDFromParams(params, extra)
	appendActiveWindowConstraints(params, extra)
	return directClickRequest{
		Action: "click_icon",
		Point:  point,
		Button: toolparams.OptionalString(params, "button", ""),
		Extra:  extra,
	}, true, nil
}

func selectIconMatch(payload iconMatchPayload, params map[string]any) (iconMatch, error) {
	if len(payload.Matches) > 0 {
		return payload.Matches[0], nil
	}
	return iconMatch{}, fmt.Errorf(
		"no icon match for template %q",
		toolparams.OptionalString(params, "template_path", ""),
	)
}

func buildIconClickPayload(selected iconMatch, payload iconMatchPayload, params map[string]any) map[string]any {
	clickPayload := map[string]any{
		"x": selected.Center.X,
		"y": selected.Center.Y,
	}
	if button := toolparams.OptionalString(params, "button", ""); button != "" {
		clickPayload["button"] = button
	}
	appendDisplayIDFromIcon(payload, clickPayload)
	appendActiveWindowConstraints(params, clickPayload)
	return clickPayload
}

func buildFindIconPayload(payload iconMatchPayload, params map[string]any) map[string]any {
	primaryMatch := firstIconMatch(payload.Matches)
	return map[string]any{
		"action":        "find_icon",
		"template_path": toolparams.OptionalString(params, "template_path", ""),
		"display_id":    payload.DisplayID,
		"image_width":   payload.ImageWidth,
		"image_height":  payload.ImageHeight,
		"scale_x":       payload.ScaleX,
		"scale_y":       payload.ScaleY,
		"origin_x":      payload.OriginX,
		"origin_y":      payload.OriginY,
		"region":        payload.Region,
		"matches":       payload.Matches,
		"primary_match": primaryMatch,
		"exists":        len(payload.Matches) > 0,
		"match_count":   len(payload.Matches),
	}
}

func firstIconMatch(matches []iconMatch) *iconMatch {
	if len(matches) == 0 {
		return nil
	}
	item := matches[0]
	return &item
}

func (t *ScreenActionTool) captureAndMatchIcon(
	ctx context.Context,
	params map[string]any,
	traceID string,
) (iconMatchPayload, error) {
	capture, err := t.captureScreen(ctx, params, traceID)
	if err != nil {
		return iconMatchPayload{}, err
	}
	payload, err := t.execution.Call(
		ctx,
		"TEMPLATE_MATCH_IMAGE",
		buildTemplateMatchParams(capture, params),
		traceID,
	)
	if err != nil {
		return iconMatchPayload{}, fmt.Errorf("execution TEMPLATE_MATCH_IMAGE failed: %w", err)
	}
	result, err := tooljson.DecodePayload[iconMatchImagePayload](payload)
	if err != nil {
		return iconMatchPayload{}, err
	}
	return iconMatchPayload{
		DisplayID:   capture.DisplayID,
		ImageWidth:  capture.ImageWidth,
		ImageHeight: capture.ImageHeight,
		ScaleX:      capture.ScaleX,
		ScaleY:      capture.ScaleY,
		OriginX:     capture.OriginX,
		OriginY:     capture.OriginY,
		Region:      capture.Region,
		Matches:     result.Matches,
	}, nil
}
