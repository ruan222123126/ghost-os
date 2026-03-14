package tools

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"ghost-os/bridge/tools/internal/tooljson"
	"ghost-os/bridge/tools/internal/toolparams"
)

type directClickRequest struct {
	Action string
	Point  screenPoint
	Button string
	Extra  map[string]any
}

func (t *ScreenActionTool) executeClickText(ctx context.Context, params map[string]any, traceID string) (string, error) {
	query, err := toolparams.RequiredString(params, "text")
	if err != nil {
		return "", err
	}
	if err := validateClickTextQuery(query, params); err != nil {
		return "", err
	}

	point, hasPoint, err := parseOptionalPoint(params)
	if err != nil {
		return "", err
	}
	logClickText(traceID, query, params)

	matchMode, occurrence, maxDistance, err := parseTextMatchOptions(query, params)
	if err != nil {
		return "", err
	}
	ocrPayload, cached, err := t.loadOCRPayloadForClick(ctx, params, traceID)
	if err != nil {
		return "", err
	}

	selected, matches, err := selectOCRMatch(ocrPayload.Items, query, matchMode, maxDistance, hasPoint, point, occurrence)
	if err != nil {
		return "", err
	}
	clickPayload := buildOCRClickPayload(selected, ocrPayload, params)
	if _, err := t.execution.Call(ctx, "MOUSE_CLICK", clickPayload, traceID); err != nil {
		return "", fmt.Errorf("execution MOUSE_CLICK failed: %w", err)
	}
	return encodeClickTextResult(query, matchMode, occurrence, hasPoint, cached, matches, selected)
}

func (t *ScreenActionTool) executeFindIcon(ctx context.Context, params map[string]any, traceID string) (string, error) {
	payload, err := t.captureAndMatchIcon(ctx, params, traceID)
	if err != nil {
		return "", err
	}
	return tooljson.Encode(payload)
}

func (t *ScreenActionTool) executeClickIcon(ctx context.Context, params map[string]any, traceID string) (string, error) {
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

func validateClickTextQuery(query string, params map[string]any) error {
	if utf8.RuneCountInString(query) != 1 {
		return nil
	}
	if toolparams.OptionalBool(params, "allow_single_char", false) {
		return nil
	}
	return fmt.Errorf("single-character text clicks require allow_single_char=true")
}

func parseTextMatchOptions(query string, params map[string]any) (string, int, int, error) {
	matchMode := strings.ToLower(toolparams.OptionalString(params, "match_mode", "exact"))
	if !isMatchModeSupported(matchMode) {
		return "", 0, 0, fmt.Errorf("match_mode must be one of: exact, contains, case_insensitive, normalized, fuzzy")
	}
	occurrence := toolparams.OptionalPositiveInt(params, "occurrence", 1)
	maxDistance := toolparams.OptionalPositiveInt(params, "max_distance", defaultFuzzyDistance(query))
	return matchMode, occurrence, maxDistance, nil
}

func (t *ScreenActionTool) loadOCRPayloadForClick(
	ctx context.Context,
	params map[string]any,
	traceID string,
) (screenOCRPayload, bool, error) {
	cacheKey, err := buildOCRCacheKey(params)
	if err != nil {
		return screenOCRPayload{}, false, err
	}
	cacheTTL := parseCacheTTL(params)
	reuseCache := toolparams.OptionalBool(params, "reuse_cache", true)
	if payload, ok := t.ensureOCRCache().load(cacheKey, cacheTTL, reuseCache); ok {
		return payload, true, nil
	}

	ocrPayload, err := t.captureAndOCR(ctx, params, traceID)
	if err != nil {
		return screenOCRPayload{}, false, err
	}
	t.ensureOCRCache().store(cacheKey, ocrPayload)
	return ocrPayload, false, nil
}

func selectOCRMatch(
	items []screenOCRItem,
	query string,
	matchMode string,
	maxDistance int,
	hasPoint bool,
	point screenPoint,
	occurrence int,
) (screenOCRItem, []screenOCRItem, error) {
	matches := filterOCRItems(items, query, matchMode, maxDistance)
	if hasPoint {
		sortOCRItemsByPoint(matches, point)
	}
	if len(matches) == 0 {
		return screenOCRItem{}, nil, fmt.Errorf("no OCR text match for %q", query)
	}
	if occurrence > len(matches) {
		return screenOCRItem{}, nil, fmt.Errorf(
			"requested occurrence %d but found %d OCR match(es) for %q",
			occurrence,
			len(matches),
			query,
		)
	}
	return matches[occurrence-1], matches, nil
}

func buildOCRClickPayload(selected screenOCRItem, payload screenOCRPayload, params map[string]any) map[string]any {
	clickPayload := map[string]any{
		"x": selected.Center.X,
		"y": selected.Center.Y,
	}
	if button := toolparams.OptionalString(params, "button", ""); button != "" {
		clickPayload["button"] = button
	}
	appendDisplayIDFromOCR(payload, clickPayload)
	appendActiveWindowConstraints(params, clickPayload)
	return clickPayload
}

func encodeClickTextResult(
	query string,
	matchMode string,
	occurrence int,
	hasPoint bool,
	cached bool,
	matches []screenOCRItem,
	selected screenOCRItem,
) (string, error) {
	return tooljson.Encode(map[string]any{
		"action":          "click_text",
		"query":           query,
		"match_mode":      matchMode,
		"occurrence":      occurrence,
		"candidate_count": len(matches),
		"hinted":          hasPoint,
		"cached":          cached,
		"clicked":         true,
		"selected":        selected,
	})
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
	return iconMatch{}, fmt.Errorf("no icon match for template %q", toolparams.OptionalString(params, "template_path", ""))
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
