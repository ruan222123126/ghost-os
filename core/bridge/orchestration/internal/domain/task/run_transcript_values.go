package task

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func transcriptRecord(value any) map[string]any {
	record, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return record
}

func transcriptNestedRecord(value any, keys ...string) map[string]any {
	current := transcriptRecord(value)
	for _, key := range keys {
		next, ok := current[strings.TrimSpace(key)]
		if !ok {
			return nil
		}
		current = transcriptRecord(next)
	}
	return current
}

func transcriptString(record map[string]any, key string) string {
	if record == nil {
		return ""
	}
	value, ok := record[strings.TrimSpace(key)]
	if !ok {
		return ""
	}
	return transcriptAnyString(value)
}

func transcriptAnyString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return ""
	}
}

func transcriptInt(record map[string]any, key string) int {
	if record == nil {
		return 0
	}
	switch value := record[strings.TrimSpace(key)].(type) {
	case int:
		return value
	case float64:
		return int(value)
	case json.Number:
		parsed, _ := strconv.Atoi(value.String())
		return parsed
	default:
		return 0
	}
}

func transcriptBool(record map[string]any, key string) bool {
	if record == nil {
		return false
	}
	value, ok := record[strings.TrimSpace(key)].(bool)
	return ok && value
}

func transcriptSlice(record map[string]any, key string) []any {
	if record == nil {
		return nil
	}
	value, ok := record[strings.TrimSpace(key)].([]any)
	if !ok {
		return nil
	}
	return value
}

func transcriptStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text := transcriptAnyString(item); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func transcriptFormatValue(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return strings.TrimSpace(string(encoded))
}
