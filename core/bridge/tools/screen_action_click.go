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
