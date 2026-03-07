// Memory path helpers centralize on-disk layout for warm/cold memory persistence.

package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolveMemoryPath(pathValue string) string {
	resolved, err := resolveMemoryPathWithError(pathValue)
	if err != nil {
		return strings.TrimSpace(pathValue)
	}
	return resolved
}

func resolveMemoryPathWithError(pathValue string) (string, error) {
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
