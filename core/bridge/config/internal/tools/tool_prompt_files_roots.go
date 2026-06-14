package tools

import (
	"errors"
	"path/filepath"
	"strings"
)

func ensureToolPromptRoots(promptsDir string, defaults map[string]string) ([]string, error) {
	dirs, err := resolveToolPromptDirs(promptsDir)
	if err != nil {
		return nil, err
	}

	roots := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		root, ensureErr := ensureToolPromptFiles(dir, defaults)
		if ensureErr != nil {
			return nil, ensureErr
		}
		roots = append(roots, root)
	}
	return roots, nil
}

func resolveToolPromptDirs(promptsDir string) ([]string, error) {
	trimmed := strings.TrimSpace(promptsDir)
	if trimmed == "" {
		return nil, errors.New("prompts_dir is empty")
	}
	return []string{filepath.Clean(trimmed)}, nil
}
