package tools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var defaultAllowedRootMediaDirNames = []string{"Files", "Apps", "App"}

func resolveAbsolutePath(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("path is empty")
	}
	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home directory: %w", err)
		}
		if trimmed == "~" {
			trimmed = homeDir
		} else {
			trimmed = filepath.Join(homeDir, strings.TrimPrefix(trimmed, "~/"))
		}
	}
	cleaned := filepath.Clean(trimmed)
	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}
	return absPath, nil
}

func ensureDirExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist: %w", err)
	}
	if !info.IsDir() {
		return errors.New("path is not a directory")
	}
	return nil
}

func defaultAllowedRoots() ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve cwd: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir: %w", err)
	}
	roots := []string{cwd, home}
	user := filepath.Base(home)
	mediaRoot := filepath.Join(string(filepath.Separator), "media", user)
	for _, name := range defaultAllowedRootMediaDirNames {
		candidate := filepath.Join(mediaRoot, name)
		if isDir(candidate) {
			roots = append(roots, candidate)
		}
	}
	return uniquePaths(roots), nil
}

func normalizeAllowedRoots(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, errors.New("allowed paths are empty")
	}
	roots := make([]string, 0, len(raw))
	for _, value := range raw {
		resolved, err := resolveAbsolutePath(value)
		if err != nil {
			return nil, fmt.Errorf("resolve allowed path %q: %w", value, err)
		}
		if err := ensureDirExists(resolved); err != nil {
			return nil, fmt.Errorf("allowed path %q: %w", resolved, err)
		}
		roots = append(roots, resolved)
	}
	return uniquePaths(roots), nil
}

func pathWithinAnyRoot(target string, roots []string) bool {
	for _, root := range roots {
		if isWithinRoot(target, root) {
			return true
		}
	}
	return false
}

func isWithinRoot(target, root string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func uniquePaths(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
