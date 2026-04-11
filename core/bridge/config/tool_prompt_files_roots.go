package config

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
	primary := filepath.Clean(trimmed)
	dirs := []string{primary}

	compat := mirrorPromptsDir(primary)
	if compat == "" {
		return dirs, nil
	}

	compat = filepath.Clean(compat)
	if compat != primary {
		dirs = append(dirs, compat)
	}
	return dirs, nil
}

func mirrorPromptsDir(primary string) string {
	cleaned := filepath.Clean(strings.TrimSpace(primary))
	if filepath.Base(cleaned) != promptsDirName {
		return ""
	}

	storeRoot := filepath.Base(filepath.Dir(cleaned))
	homeDir := filepath.Dir(filepath.Dir(cleaned))
	switch storeRoot {
	case ghostOSDirName:
		return filepath.Join(homeDir, ghostDirName, promptsDirName)
	case ghostDirName:
		return filepath.Join(homeDir, ghostOSDirName, promptsDirName)
	default:
		return ""
	}
}
