package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/tools"
)

func buildFindIconPreviewToolArgs(req findIconPreviewRequest) map[string]any {
	params := map[string]any{
		"template_path": req.TemplatePath,
	}
	if req.Threshold != nil {
		params["threshold"] = *req.Threshold
	}
	if req.MaxResults != nil {
		params["max_results"] = *req.MaxResults
	}
	if req.DisplayID != nil {
		params["display_id"] = *req.DisplayID
	}
	if req.Region != nil {
		params["region"] = map[string]any{
			"x":      req.Region.X,
			"y":      req.Region.Y,
			"width":  req.Region.Width,
			"height": req.Region.Height,
		}
	}
	return map[string]any{
		"mode":   "atomic",
		"action": "find_icon",
		"params": params,
	}
}

func executeFindIconPreviewHover(
	ctx context.Context,
	tool tools.Tool,
	payload findIconPreviewPayload,
	traceID string,
) error {
	x, y, err := firstFindIconMatchCenter(payload.Matches)
	if err != nil {
		return err
	}
	args, err := json.Marshal(buildFindIconPreviewHoverToolArgs(x, y, payload.DisplayID))
	if err != nil {
		return fmt.Errorf("encode find_icon hover args: %w", err)
	}
	if _, err := tool.Execute(tools.WithToolCallID(ctx, "find-icon-preview-hover"), args, traceID); err != nil {
		return err
	}
	return nil
}

func buildFindIconPreviewHoverToolArgs(x int, y int, displayID *int) map[string]any {
	params := map[string]any{
		"x":          x,
		"y":          y,
		"hover_only": true,
	}
	if displayID != nil {
		params["display_id"] = *displayID
	}
	return map[string]any{
		"mode":   "atomic",
		"action": "click_icon",
		"params": params,
	}
}

func firstFindIconMatchCenter(matches []map[string]any) (int, int, error) {
	if len(matches) == 0 {
		return 0, 0, fmt.Errorf("find_icon output field matches is empty")
	}
	centerRecord, ok := matches[0]["center"].(map[string]any)
	if !ok || centerRecord == nil {
		return 0, 0, fmt.Errorf("find_icon output field matches[0].center must be an object")
	}
	x, ok := parseOptionalFindIconInt(centerRecord["x"])
	if !ok {
		return 0, 0, fmt.Errorf("find_icon output field matches[0].center.x must be a number")
	}
	y, ok := parseOptionalFindIconInt(centerRecord["y"])
	if !ok {
		return 0, 0, fmt.Errorf("find_icon output field matches[0].center.y must be a number")
	}
	return x, y, nil
}

func decodeFindIconPreviewPayload(raw string) (findIconPreviewPayload, error) {
	var decoded map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &decoded); err != nil {
		return findIconPreviewPayload{}, fmt.Errorf("decode find_icon output: %w", err)
	}
	matches, err := parseFindIconMatchList(decoded["matches"])
	if err != nil {
		return findIconPreviewPayload{}, err
	}
	out := findIconPreviewPayload{
		Exists:     len(matches) > 0,
		MatchCount: len(matches),
		Matches:    matches,
	}
	if displayID, ok := parseOptionalFindIconInt(decoded["display_id"]); ok {
		out.DisplayID = &displayID
	}
	if region, ok, err := parseOptionalFindIconRegion(decoded["region"]); err != nil {
		return findIconPreviewPayload{}, err
	} else if ok {
		out.Region = &region
	}
	return out, nil
}

func parseFindIconMatchList(raw any) ([]map[string]any, error) {
	if raw == nil {
		return []map[string]any{}, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("find_icon output field matches must be an array")
	}
	parsed := make([]map[string]any, 0, len(items))
	for index := range items {
		record, ok := items[index].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("find_icon output field matches[%d] must be an object", index)
		}
		parsed = append(parsed, record)
	}
	return parsed, nil
}

func parseOptionalFindIconInt(raw any) (int, bool) {
	value, ok := raw.(float64)
	if !ok {
		return 0, false
	}
	return int(value), true
}

func parseOptionalFindIconRegion(raw any) (findIconPreviewRegion, bool, error) {
	record, ok := raw.(map[string]any)
	if !ok || record == nil {
		return findIconPreviewRegion{}, false, nil
	}
	x, ok := parseOptionalFindIconInt(record["x"])
	if !ok {
		return findIconPreviewRegion{}, false, fmt.Errorf("find_icon output field region.x must be a number")
	}
	y, ok := parseOptionalFindIconInt(record["y"])
	if !ok {
		return findIconPreviewRegion{}, false, fmt.Errorf("find_icon output field region.y must be a number")
	}
	width, ok := parseOptionalFindIconInt(record["width"])
	if !ok {
		return findIconPreviewRegion{}, false, fmt.Errorf("find_icon output field region.width must be a number")
	}
	height, ok := parseOptionalFindIconInt(record["height"])
	if !ok {
		return findIconPreviewRegion{}, false, fmt.Errorf("find_icon output field region.height must be a number")
	}
	return findIconPreviewRegion{X: x, Y: y, Width: width, Height: height}, true, nil
}
