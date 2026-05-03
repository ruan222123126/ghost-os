package orchestration

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
)

var reasoningPriorityKeys = []string{
	"thinking",
	"text",
	"summary_text",
	"reasoning_text",
	"content",
	"summary",
	"reasoning",
	"value",
}

func extractReasoningDisplayText(raw json.RawMessage) string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return ""
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return compactReasoningJSON(trimmed)
	}

	if text := strings.TrimSpace(extractReasoningTextValue(decoded)); text != "" {
		return text
	}
	return compactReasoningJSON(trimmed)
}

func extractReasoningTextValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		return joinReasoningParts(extractReasoningArrayText(typed))
	case map[string]any:
		return joinReasoningParts(extractReasoningObjectText(typed))
	default:
		return ""
	}
}

func extractReasoningArrayText(items []any) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text := strings.TrimSpace(extractReasoningTextValue(item)); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func extractReasoningObjectText(record map[string]any) []string {
	keys := prioritizedReasoningKeys(record)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if text := strings.TrimSpace(extractReasoningTextValue(record[key])); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func prioritizedReasoningKeys(record map[string]any) []string {
	keys := make([]string, 0, len(record))
	seen := make(map[string]struct{}, len(record))
	for _, key := range reasoningPriorityKeys {
		if _, ok := record[key]; !ok {
			continue
		}
		keys = append(keys, key)
		seen[key] = struct{}{}
	}
	rest := make([]string, 0, len(record)-len(keys))
	for key := range record {
		if _, ok := seen[key]; ok {
			continue
		}
		rest = append(rest, key)
	}
	sort.Strings(rest)
	return append(keys, rest...)
}

func joinReasoningParts(parts []string) string {
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func compactReasoningJSON(raw string) string {
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, []byte(raw)); err != nil {
		return strings.TrimSpace(raw)
	}
	return buffer.String()
}
