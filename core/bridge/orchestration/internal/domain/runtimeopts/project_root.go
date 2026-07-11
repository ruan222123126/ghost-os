package runtimeopts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrProjectRootRequired = errors.New("project_root is required")
	ErrProjectRootAbsolute = errors.New("project_root must be an absolute path")
	ErrProjectRootDir      = errors.New("project_root must be a directory")
	ErrProjectRootMissing  = errors.New("project_root does not exist")
)

type RequestOptions struct {
	ProjectRoot string
}

func NormalizeRequestOptions(rawProjectRoot string) (*RequestOptions, error) {
	trimmed := strings.TrimSpace(rawProjectRoot)
	if trimmed == "" {
		return nil, nil
	}
	projectRoot, err := NormalizeProjectRoot(trimmed)
	if err != nil {
		return nil, err
	}
	return &RequestOptions{ProjectRoot: projectRoot}, nil
}

func NormalizeProjectRoot(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrProjectRootRequired
	}

	resolved, err := resolveUserPath(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve project_root: %w", err)
	}
	if !filepath.IsAbs(resolved) {
		return "", ErrProjectRootAbsolute
	}

	cleaned := filepath.Clean(resolved)
	info, err := os.Stat(cleaned)
	if errors.Is(err, os.ErrNotExist) {
		return "", ErrProjectRootMissing
	}
	if err != nil {
		return "", fmt.Errorf("stat project_root: %w", err)
	}
	if !info.IsDir() {
		return "", ErrProjectRootDir
	}
	return cleaned, nil
}

func resolveUserPath(pathValue string) (string, error) {
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
