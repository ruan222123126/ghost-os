package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Resolve(pathValue string) string {
	resolved, err := ResolveWithError(pathValue)
	if err != nil {
		return strings.TrimSpace(pathValue)
	}
	return resolved
}

func ResolveWithError(pathValue string) (string, error) {
	trimmed := strings.TrimSpace(pathValue)
	if trimmed == "" {
		return "", nil
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
