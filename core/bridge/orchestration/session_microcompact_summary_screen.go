package orchestration

import (
	"fmt"
	"strings"
)

type microcompactScreenArgs struct {
	Action    string         `json:"action"`
	Params    map[string]any `json:"params,omitempty"`
	DisplayID *int           `json:"display_id,omitempty"`
}

func summarizeScreenToolResult(pair microcompactToolPair) (string, error) {
	args, err := decodeJSONText[microcompactScreenArgs](string(pair.call.Arguments))
	if err != nil {
		return "", microcompactParseFailed(pair.toolName, pair.toolCallID)
	}
	payload, err := decodeJSONText[map[string]any](pair.envelope.Output)
	if err != nil {
		return "", microcompactUnsupported(pair.toolName, pair.toolCallID, "screen payload")
	}
	parts := []string{pair.toolName, screenActionName(args, payload)}
	parts = append(parts, screenParamDetails(args)...)
	parts = append(parts, screenPayloadDetails(payload)...)
	return strings.Join(compactNonEmpty(parts), " "), nil
}

func screenActionName(args microcompactScreenArgs, payload map[string]any) string {
	if action := nonEmptyString(payload["action"]); action != "" {
		return action
	}
	return strings.TrimSpace(args.Action)
}

func screenParamDetails(args microcompactScreenArgs) []string {
	params := args.Params
	parts := make([]string, 0, 4)
	if args.DisplayID != nil {
		parts = append(parts, fmt.Sprintf("display_id=%d", *args.DisplayID))
	}
	appendScreenStringField(&parts, "text", params["text"])
	appendScreenStringField(&parts, "template_path", params["template_path"])
	appendScreenPointField(&parts, params)
	appendScreenBoolField(&parts, "submit", params["submit"])
	return parts
}

func screenPayloadDetails(payload map[string]any) []string {
	parts := make([]string, 0, 6)
	appendScreenBoolField(&parts, "clicked", payload["clicked"])
	appendScreenBoolField(&parts, "hovered", payload["hovered"])
	appendScreenBoolField(&parts, "exists", payload["exists"])
	appendScreenIntField(&parts, "ocr_matches", payload["candidate_count"])
	appendScreenIntField(&parts, "icon_matches", payload["match_count"])
	if optionalMap(payload["artifact"]) != nil {
		parts = append(parts, "artifact=image")
	}
	appendScreenIntField(&parts, "characters", payload["characters"])
	appendScreenBoolField(&parts, "typed", payload["typed"])
	appendScreenBoolField(&parts, "submitted", payload["submitted"])
	return parts
}

func appendScreenStringField(parts *[]string, key string, value any) {
	if text := nonEmptyString(value); text != "" {
		*parts = append(*parts, fmt.Sprintf("%s=%q", key, truncateMicrocompactText(text, microcompactMaxTextPreview)))
	}
}

func appendScreenPointField(parts *[]string, params map[string]any) {
	if params == nil {
		return
	}
	x, xOK := optionalInt(params["x"])
	y, yOK := optionalInt(params["y"])
	if xOK && yOK {
		*parts = append(*parts, fmt.Sprintf("point=%d,%d", x, y))
	}
}

func appendScreenBoolField(parts *[]string, key string, value any) {
	if typed, ok := optionalBool(value); ok {
		*parts = append(*parts, fmt.Sprintf("%s=%t", key, typed))
	}
}

func appendScreenIntField(parts *[]string, key string, value any) {
	if typed, ok := optionalInt(value); ok {
		*parts = append(*parts, fmt.Sprintf("%s=%d", key, typed))
	}
}

func compactNonEmpty(parts []string) []string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return filtered
}
