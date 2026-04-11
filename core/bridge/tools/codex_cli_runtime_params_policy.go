package tools

import (
	"fmt"
	"strings"
)

func (t *CodexCLITool) validateReadPath(path string) error {
	roots, err := t.allowedReadRoots()
	if err != nil {
		return err
	}
	if pathWithinAnyRoot(path, roots) {
		return nil
	}
	return fmt.Errorf("cwd is outside allowed read paths")
}

func (t *CodexCLITool) validateWritePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	roots, err := t.allowedWriteRoots()
	if err != nil {
		return err
	}
	if pathWithinAnyRoot(path, roots) {
		return nil
	}
	return fmt.Errorf("output_path is outside allowed write paths")
}

func (t *CodexCLITool) allowedReadRoots() ([]string, error) {
	if t == nil || len(t.allowedReadPaths) == 0 {
		return defaultAllowedRoots()
	}
	return normalizeAllowedRoots(t.allowedReadPaths)
}

func (t *CodexCLITool) allowedWriteRoots() ([]string, error) {
	if t == nil || len(t.allowedWritePaths) == 0 {
		return defaultAllowedRoots()
	}
	return normalizeAllowedRoots(t.allowedWritePaths)
}
