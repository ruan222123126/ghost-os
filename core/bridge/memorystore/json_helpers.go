package memorystore

import (
	"encoding/json"
	"fmt"
	"strings"
)

func marshalString(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode json string: %w", err)
	}
	return string(encoded), nil
}

func unmarshalStringSlice(raw string, target *[]string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		*target = nil
		return nil
	}
	var decoded []string
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return fmt.Errorf("decode string slice: %w", err)
	}
	*target = normalizeIdentifierList(decoded)
	return nil
}
