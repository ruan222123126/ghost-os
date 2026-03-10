package execution

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type nativeBinaryLocator struct {
	configuredPath string
	roots          []string
	candidates     []string
}

func locateNativeBinary(locator nativeBinaryLocator) (string, error) {
	if configured := strings.TrimSpace(locator.configuredPath); configured != "" {
		path, err := normalizeBinaryPath(configured)
		if err != nil {
			return "", fmt.Errorf("invalid native binary path: %w", err)
		}
		if !binaryExists(path) {
			return "", fmt.Errorf("native binary path points to missing binary: %s", path)
		}
		return path, nil
	}

	roots := locator.roots
	if len(roots) == 0 {
		roots = defaultNativeBinaryRoots()
	}
	candidates := locator.candidates
	if len(candidates) == 0 {
		candidates = nativeBinaryCandidates()
	}

	path, err := locateNativeBinaryInRoots(candidates, roots)
	if err == nil {
		return path, nil
	}

	return "", errors.New("native binary not found; configure native binary path or run `cargo build` in drivers/native first")
}

func nativeBinaryCandidates() []string {
	return []string{
		"native",
		"native.exe",
		"../../drivers/native/target/release/native",
		"drivers/native/target/release/native",
		"../../drivers/native/target/release/native.exe",
		"drivers/native/target/release/native.exe",
		"../../drivers/native/target/debug/native",
		"drivers/native/target/debug/native",
		"../../drivers/native/target/debug/native.exe",
		"drivers/native/target/debug/native.exe",
	}
}

func defaultNativeBinaryRoots() []string {
	roots := []string{"."}
	if executablePath, err := os.Executable(); err == nil {
		if resolved, resolveErr := filepath.EvalSymlinks(executablePath); resolveErr == nil {
			executablePath = resolved
		}
		roots = append(roots, filepath.Dir(executablePath))
	}
	return roots
}

func locateNativeBinaryInRoots(candidates []string, roots []string) (string, error) {
	seen := make(map[string]struct{}, len(candidates)*len(roots))
	for _, root := range roots {
		base := strings.TrimSpace(root)
		if base == "" {
			base = "."
		}

		for _, candidate := range candidates {
			if strings.TrimSpace(candidate) == "" {
				continue
			}

			path := candidate
			if !filepath.IsAbs(candidate) {
				path = filepath.Join(base, candidate)
			}

			resolved, err := normalizeBinaryPath(path)
			if err != nil {
				continue
			}
			if _, exists := seen[resolved]; exists {
				continue
			}
			seen[resolved] = struct{}{}

			if binaryExists(resolved) {
				return resolved, nil
			}
		}
	}

	return "", errors.New("native binary not found in roots")
}

func normalizeBinaryPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("path is empty")
	}

	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}

	return filepath.Clean(abs), nil
}

func binaryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
