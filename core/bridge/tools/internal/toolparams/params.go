package toolparams

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func RequiredString(params map[string]any, field string) (string, error) {
	value := OptionalString(params, field, "")
	if value == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return value, nil
}

func OptionalString(params map[string]any, field string, fallback string) string {
	if len(params) == 0 {
		return fallback
	}
	text, _ := params[field].(string)
	text = strings.TrimSpace(text)
	if text == "" {
		return fallback
	}
	return text
}

func OptionalBool(params map[string]any, field string, fallback bool) bool {
	if len(params) == 0 {
		return fallback
	}
	value, ok := params[field].(bool)
	if !ok {
		return fallback
	}
	return value
}

func OptionalInt(params map[string]any, field string) (int, bool) {
	if len(params) == 0 {
		return 0, false
	}
	return Int(params[field])
}

func Int(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case float32:
		return int(typed), true
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return int(parsed), true
		}
		parsed, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func OptionalFloat(params map[string]any, field string) (float64, bool) {
	if len(params) == 0 {
		return 0, false
	}
	switch typed := params[field].(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		return parsed, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func OptionalPositiveInt(params map[string]any, field string, fallback int) int {
	value, ok := OptionalInt(params, field)
	if !ok || value <= 0 {
		return fallback
	}
	return value
}

func DurationMillis(params map[string]any, field string, fallback time.Duration) time.Duration {
	value, ok := OptionalInt(params, field)
	if !ok || value < 0 {
		return fallback
	}
	return time.Duration(value) * time.Millisecond
}
