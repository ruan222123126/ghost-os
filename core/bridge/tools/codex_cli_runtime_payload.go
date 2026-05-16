package tools

import (
	"fmt"
	"strings"
)

func requiredPayloadString(payload map[string]any, field string) (string, error) {
	value, err := optionalPayloadString(payload, field)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("invalid payload: missing %s", field)
	}
	return value, nil
}

func optionalPayloadString(payload map[string]any, field string) (string, error) {
	raw, ok := payload[field]
	if !ok || raw == nil {
		return "", nil
	}
	value, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("invalid payload: %s must be a string", field)
	}
	return strings.TrimSpace(value), nil
}

func optionalPayloadInt(payload map[string]any, field string) (*int, error) {
	raw, ok := payload[field]
	if !ok || raw == nil {
		return nil, nil
	}
	switch value := raw.(type) {
	case int:
		return intPointer(value), nil
	case int32:
		return intPointer(int(value)), nil
	case int64:
		return intPointer(int(value)), nil
	case float64:
		result := int(value)
		if float64(result) != value {
			return nil, fmt.Errorf("invalid payload: %s must be an integer", field)
		}
		return intPointer(result), nil
	default:
		return nil, fmt.Errorf("invalid payload: %s must be an integer", field)
	}
}

func intPointer(value int) *int {
	result := value
	return &result
}
