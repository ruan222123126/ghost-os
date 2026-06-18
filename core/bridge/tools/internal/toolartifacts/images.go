package toolartifacts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultScreenshotsPath = "~/.ghost-os/screenshots"

func ResolveScreenshotsRoot() (string, error) {
	baseDir := strings.TrimSpace(os.Getenv("GHOST_SCREENSHOTS_PATH"))
	if baseDir == "" {
		baseDir = defaultScreenshotsPath
	}
	if expanded, err := expandHome(baseDir); err != nil {
		return "", err
	} else {
		baseDir = expanded
	}
	absolute, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("resolve screenshots directory: %w", err)
	}
	return filepath.Clean(absolute), nil
}

func SanitizePathComponent(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	var builder strings.Builder
	builder.Grow(len(trimmed))
	for _, ch := range trimmed {
		switch {
		case ch >= 'a' && ch <= 'z':
			builder.WriteRune(ch)
		case ch >= 'A' && ch <= 'Z':
			builder.WriteRune(ch)
		case ch >= '0' && ch <= '9':
			builder.WriteRune(ch)
		case ch == '-' || ch == '_':
			builder.WriteRune(ch)
		default:
			builder.WriteByte('_')
		}
	}
	out := strings.Trim(builder.String(), "_")
	if out == "" {
		return fallback
	}
	return out
}

func FormatUTCTimestamp(now time.Time) string {
	return fmt.Sprintf("%s%09dZ", now.Format("20060102T150405"), now.Nanosecond())
}

func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home for screenshots directory: %w", err)
	}
	if path == "~" {
		return homeDir, nil
	}
	return filepath.Join(homeDir, strings.TrimPrefix(path, "~/")), nil
}
