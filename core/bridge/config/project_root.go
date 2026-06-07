package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	errProjectRootRequired = errors.New("project_root is required")
	errProjectRootAbsolute = errors.New("project_root must be an absolute path")
	errProjectRootDir      = errors.New("project_root must be a directory")
	errProjectRootMissing  = errors.New("project_root does not exist")
)

func NormalizeProjectRoot(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errProjectRootRequired
	}

	resolved, err := resolveUserPath(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve project_root: %w", err)
	}
	if !filepath.IsAbs(resolved) {
		return "", errProjectRootAbsolute
	}

	cleaned := filepath.Clean(resolved)
	info, err := os.Stat(cleaned)
	if errors.Is(err, os.ErrNotExist) {
		return "", errProjectRootMissing
	}
	if err != nil {
		return "", fmt.Errorf("stat project_root: %w", err)
	}
	if !info.IsDir() {
		return "", errProjectRootDir
	}
	return cleaned, nil
}

func applyProjectRootUpdate(fileCfg *bridgeFileConfig, raw *string) error {
	if fileCfg == nil || raw == nil {
		return nil
	}
	if strings.TrimSpace(*raw) == "" {
		fileCfg.ProjectRoot = nil
		return nil
	}

	normalized, err := NormalizeProjectRoot(*raw)
	if err != nil {
		return err
	}
	fileCfg.ProjectRoot = stringPointer(normalized)
	return nil
}
