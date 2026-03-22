package memorystore

import (
	"encoding/json"
	"fmt"
	"strings"
)

func mustMarshalString(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(encoded)
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
