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

func NormalizeFindIconPreviewRequest(req api.FindIconPreviewRequest) (api.FindIconPreviewRequest, error) {
	templatePath := strings.TrimSpace(req.TemplatePath)
	if templatePath == "" {
		return api.FindIconPreviewRequest{}, fmt.Errorf("template_path is required")
	}
	out := req
	out.TemplatePath = templatePath
	if req.Threshold != nil && (*req.Threshold < 0 || *req.Threshold > 1) {
		return api.FindIconPreviewRequest{}, fmt.Errorf("threshold must be between 0 and 1")
	}
	if req.MaxResults != nil && *req.MaxResults < 1 {
		return api.FindIconPreviewRequest{}, fmt.Errorf("max_results must be >= 1")
	}
	if req.DisplayID != nil && *req.DisplayID < 0 {
		return api.FindIconPreviewRequest{}, fmt.Errorf("display_id must be >= 0")
	}
	return out, nil
}

func ExecuteFindIconPreviewToolCall(
	ctx context.Context,
	tool bridgetools.Tool,
	req api.FindIconPreviewRequest,
	traceID string,
) (api.FindIconPreviewPayload, error) {
	args, err := json.Marshal(BuildFindIconPreviewToolArgs(req))
	if err != nil {
		return api.FindIconPreviewPayload{}, bus.WrapError(bus.ServiceErrorInternal, fmt.Errorf("encode find_icon args: %w", err))
	}
	output, err := tool.Execute(bridgetools.WithToolCallID(ctx, "find-icon-preview"), args, traceID)
	if err != nil {
		return api.FindIconPreviewPayload{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
	}
	return decodeFindIconPreviewOutput(ctx, tool, req, traceID, output)
}

func decodeFindIconPreviewOutput(
	ctx context.Context,
	tool bridgetools.Tool,
	req api.FindIconPreviewRequest,
	traceID string,
	output string,
) (api.FindIconPreviewPayload, error) {
	payload, err := DecodeFindIconPreviewPayload(output)
	if err != nil {
		return api.FindIconPreviewPayload{}, bus.WrapError(bus.ServiceErrorInternal, err)
	}
	if req.HoverAfterMatch && payload.Exists {
		if err := ExecuteFindIconPreviewHover(ctx, tool, payload, traceID); err != nil {
			return api.FindIconPreviewPayload{}, bus.WrapError(bus.ServiceErrorInvalidInput, err)
		}
		payload.Hovered = true
	}
	return payload, nil
}

func BuildFindIconPreviewToolArgs(req api.FindIconPreviewRequest) map[string]any {
	params := map[string]any{"template_path": req.TemplatePath}
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
			"x": req.Region.X, "y": req.Region.Y,
			"width": req.Region.Width, "height": req.Region.Height,
		}
	}
	return map[string]any{"mode": "atomic", "action": "find_icon", "params": params}
}

func ExecuteFindIconPreviewHover(
	ctx context.Context,
	tool bridgetools.Tool,
	payload api.FindIconPreviewPayload,
	traceID string,
) error {
	x, y, err := FirstFindIconMatchCenter(payload.Matches)
	if err != nil {
		return err
	}
	args, err := json.Marshal(BuildFindIconPreviewHoverToolArgs(x, y, payload.DisplayID))
	if err != nil {
		return fmt.Errorf("encode find_icon hover args: %w", err)
	}
	_, err = tool.Execute(bridgetools.WithToolCallID(ctx, "find-icon-preview-hover"), args, traceID)
	return err
}

func BuildFindIconPreviewHoverToolArgs(x int, y int, displayID *int) map[string]any {
	params := map[string]any{"x": x, "y": y, "hover_only": true}
	if displayID != nil {
		params["display_id"] = *displayID
	}
	return map[string]any{"mode": "atomic", "action": "click_icon", "params": params}
}

func FirstFindIconMatchCenter(matches []map[string]any) (int, int, error) {
	if len(matches) == 0 {
		return 0, 0, fmt.Errorf("find_icon output field matches is empty")
	}
	centerRecord, ok := matches[0]["center"].(map[string]any)
	if !ok || centerRecord == nil {
		return 0, 0, fmt.Errorf("find_icon output field matches[0].center must be an object")
	}
	return parseFindIconCenter(centerRecord)
}

func parseFindIconCenter(centerRecord map[string]any) (int, int, error) {
	x, ok := ParseOptionalFindIconInt(centerRecord["x"])
	if !ok {
		return 0, 0, fmt.Errorf("find_icon output field matches[0].center.x must be a number")
	}
	y, ok := ParseOptionalFindIconInt(centerRecord["y"])
	if !ok {
		return 0, 0, fmt.Errorf("find_icon output field matches[0].center.y must be a number")
	}
	return x, y, nil
}

func DecodeFindIconPreviewPayload(raw string) (api.FindIconPreviewPayload, error) {
	var decoded map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &decoded); err != nil {
		return api.FindIconPreviewPayload{}, fmt.Errorf("decode find_icon output: %w", err)
	}
	matches, err := ParseFindIconMatchList(decoded["matches"])
	if err != nil {
		return api.FindIconPreviewPayload{}, err
	}
	return buildFindIconPreviewPayload(decoded, matches)
}

func buildFindIconPreviewPayload(
	decoded map[string]any,
	matches []map[string]any,
) (api.FindIconPreviewPayload, error) {
	out := api.FindIconPreviewPayload{Exists: len(matches) > 0, MatchCount: len(matches), Matches: matches}
	if displayID, ok := ParseOptionalFindIconInt(decoded["display_id"]); ok {
		out.DisplayID = &displayID
	}
	if region, ok, err := ParseOptionalFindIconRegion(decoded["region"]); err != nil {
		return api.FindIconPreviewPayload{}, err
	} else if ok {
		out.Region = &region
	}
	return out, nil
}

func ParseFindIconMatchList(raw any) ([]map[string]any, error) {
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

func ParseOptionalFindIconInt(raw any) (int, bool) {
	value, ok := raw.(float64)
	if !ok {
		return 0, false
	}
	return int(value), true
}

func ParseOptionalFindIconRegion(raw any) (api.FindIconPreviewRegion, bool, error) {
	record, ok := raw.(map[string]any)
	if !ok || record == nil {
		return api.FindIconPreviewRegion{}, false, nil
	}
	return parseFindIconRegionRecord(record)
}

func parseFindIconRegionRecord(record map[string]any) (api.FindIconPreviewRegion, bool, error) {
	x, ok := ParseOptionalFindIconInt(record["x"])
	if !ok {
		return api.FindIconPreviewRegion{}, false, fmt.Errorf("find_icon output field region.x must be a number")
	}
	y, ok := ParseOptionalFindIconInt(record["y"])
	if !ok {
		return api.FindIconPreviewRegion{}, false, fmt.Errorf("find_icon output field region.y must be a number")
	}
	width, ok := ParseOptionalFindIconInt(record["width"])
	if !ok {
		return api.FindIconPreviewRegion{}, false, fmt.Errorf("find_icon output field region.width must be a number")
	}
	height, ok := ParseOptionalFindIconInt(record["height"])
	if !ok {
		return api.FindIconPreviewRegion{}, false, fmt.Errorf("find_icon output field region.height must be a number")
	}
	return api.FindIconPreviewRegion{X: x, Y: y, Width: width, Height: height}, true, nil
}
