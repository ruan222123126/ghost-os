package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	bridgeconfig "ghost-os/bridge/config"
)

const (
	memoryRootDirName = ".ghost-os"
	memoryDirName     = "memory"
	memoryDayDirName  = "day"
	memoryDirPerm     = 0o755
	memoryFilePerm    = 0o644
)

func prepareMemoryMode(cfg Config) error {
	if !cfg.MemoryModeEnabled {
		return nil
	}
	return ensureMemoryDayFile(time.Now())
}

func activePromptContentForInsertPoint(
	promptLibrary []bridgeconfig.SystemPromptLibraryItem,
	insertPoint bridgeconfig.SystemPromptInsertPoint,
) (string, bool) {
	for _, item := range promptLibrary {
		if item.InsertPoint == insertPoint && item.Active {
			return strings.TrimSpace(item.Content), true
		}
	}
	return "", false
}

func ensureMemoryDayFile(now time.Time) error {
	path, err := memoryDayFilePath(now)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), memoryDirPerm); err != nil {
		return fmt.Errorf("create memory day directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, memoryFilePerm)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("create memory day file %q: %w", path, err)
	}
	return file.Close()
}

func memoryDayFilePath(now time.Time) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home for memory path: %w", err)
	}
	homeDir = strings.TrimSpace(homeDir)
	if homeDir == "" {
		return "", errors.New("resolve user home for memory path: empty home directory")
	}
	filename := now.Format("2006-01-02") + ".md"
	return filepath.Join(homeDir, memoryRootDirName, memoryDirName, memoryDayDirName, filename), nil
}
