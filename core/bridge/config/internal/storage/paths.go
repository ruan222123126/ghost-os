package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ResolveUserPath(pathValue string) (string, error) {
	trimmed := strings.TrimSpace(pathValue)
	if trimmed == "" {
		return "", errors.New("path is empty")
	}

	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home directory: %w", err)
		}
		if trimmed == "~" {
			return homeDir, nil
		}
		return filepath.Join(homeDir, strings.TrimPrefix(trimmed, "~/")), nil
	}

	return filepath.Clean(trimmed), nil
}

func NormalizeConfiguredPathList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func CloneOptionalStringPointer(raw *string) *string {
	if raw == nil {
		return nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil
	}
	return &value
}

func CloneBoolPointer(raw *bool) *bool {
	if raw == nil {
		return nil
	}
	value := *raw
	return &value
}

func CloneIntPointer(raw *int) *int {
	if raw == nil {
		return nil
	}
	value := *raw
	return &value
}

func OptionalStringPointer(raw string) *string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	return &value
}

func StringPointer(raw string) *string {
	value := strings.TrimSpace(raw)
	return &value
}

func StringValue(raw *string) string {
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(*raw)
}
