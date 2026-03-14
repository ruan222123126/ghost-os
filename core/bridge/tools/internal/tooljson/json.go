package tooljson

import (
	"encoding/json"
	"fmt"
)

func DecodePayload[T any](payload map[string]any) (T, error) {
	var out T
	encoded, err := json.Marshal(payload)
	if err != nil {
		return out, fmt.Errorf("encode payload: %w", err)
	}
	if err := json.Unmarshal(encoded, &out); err != nil {
		return out, fmt.Errorf("decode payload: %w", err)
	}
	return out, nil
}

func Encode(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode payload: %w", err)
	}
	return string(encoded), nil
}
