package payloadutil

import (
	"fmt"
	"math"
)

func String(payload map[string]any, field string) (string, error) {
	value, ok := payload[field]
	if !ok {
		return "", fmt.Errorf("invalid payload: missing %s", field)
	}

	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("invalid payload: %s must be a string", field)
	}
	return text, nil
}

func StringSlice(payload map[string]any, field string) ([]string, error) {
	value, ok := payload[field]
	if !ok {
		return nil, fmt.Errorf("invalid payload: missing %s", field)
	}

	switch typed := value.(type) {
	case []string:
		out := make([]string, len(typed))
		copy(out, typed)
		return out, nil
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("invalid payload: %s items must be strings", field)
			}
			out = append(out, text)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("invalid payload: %s must be an array", field)
	}
}

func Int(payload map[string]any, field string) (int, error) {
	value, ok := payload[field]
	if !ok {
		return 0, fmt.Errorf("invalid payload: missing %s", field)
	}

	switch typed := value.(type) {
	case int:
		return typed, nil
	case int32:
		return int(typed), nil
	case int64:
		return int(typed), nil
	case float64:
		if math.Trunc(typed) != typed {
			return 0, fmt.Errorf("invalid payload: %s must be an integer", field)
		}
		return int(typed), nil
	default:
		return 0, fmt.Errorf("invalid payload: %s must be an integer", field)
	}
}

func NumericToInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func AnySlice(payload map[string]any, field string) ([]any, error) {
	value, ok := payload[field]
	if !ok {
		return nil, fmt.Errorf("invalid payload: missing %s", field)
	}

	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid payload: %s must be an array", field)
	}
	out := make([]any, len(items))
	copy(out, items)
	return out, nil
}
