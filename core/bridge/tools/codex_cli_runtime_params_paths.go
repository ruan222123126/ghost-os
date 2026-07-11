package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (t *CodexCLITool) resolveStartPaths(cwd string, outputPath string) (string, string, error) {
	baseDir, err := t.resolveBaseWorkingDir()
	if err != nil {
		return "", "", err
	}
	workingDir, err := resolveStartWorkingDir(cwd, baseDir)
	if err != nil {
		return "", "", err
	}
	if err := t.validateReadPath(workingDir); err != nil {
		return "", "", err
	}
	resolvedOutputPath, err := resolveOptionalOutputPath(outputPath, baseDir)
	if err != nil {
		return "", "", err
	}
	if err := t.validateWritePath(resolvedOutputPath); err != nil {
		return "", "", err
	}
	return workingDir, resolvedOutputPath, nil
}

func resolveStartWorkingDir(rawCwd string, baseDir string) (string, error) {
	trimmed := strings.TrimSpace(rawCwd)
	if trimmed == "" {
		if err := ensureDirExists(baseDir); err != nil {
			return "", err
		}
		return baseDir, nil
	}
	resolved, err := resolvePathFromBase(trimmed, baseDir)
	if err != nil {
		return "", err
	}
	if err := ensureDirExists(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

func resolveOptionalOutputPath(rawOutputPath string, baseDir string) (string, error) {
	trimmed := strings.TrimSpace(rawOutputPath)
	if trimmed == "" {
		return "", nil
	}
	return resolvePathFromBase(trimmed, baseDir)
}

func (t *CodexCLITool) resolveBaseWorkingDir() (string, error) {
	if t != nil && t.resolveProjectRoot != nil {
		root, err := t.resolveProjectRoot()
		if err != nil {
			return "", fmt.Errorf("resolve project root failed: %w", err)
		}
		if strings.TrimSpace(root) != "" {
			resolved, err := resolveAbsolutePath(root)
			if err != nil {
				return "", err
			}
			if err := ensureDirExists(resolved); err != nil {
				return "", err
			}
			return resolved, nil
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve cwd: %w", err)
	}
	resolved, err := resolveAbsolutePath(cwd)
	if err != nil {
		return "", err
	}
	if err := ensureDirExists(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

func resolvePathFromBase(pathValue string, baseDir string) (string, error) {
	if filepath.IsAbs(pathValue) {
		return resolveAbsolutePath(pathValue)
	}
	return resolveAbsolutePath(filepath.Join(baseDir, pathValue))
}
