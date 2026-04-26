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

const defaultMemoryPrompt = `- After completing one meaningful non-greeting task, append a concise memory note to ~/.ghost-os/memory/day/YYYY-MM-DD.md.
- Do not write entries for greetings, acknowledgements, or trivial status-only turns.
- At the start of each new topic, search ~/.ghost-os/memory/day by keywords from the request, then read matched day files before taking action.
- Use today's local date in YYYY-MM-DD format when choosing the target memory file name.`

func resolveMemorySection(
	cfg Config,
	promptLibrary []bridgeconfig.SystemPromptLibraryItem,
) (string, error) {
	if !cfg.MemoryModeEnabled {
		return "", nil
	}
	if err := ensureMemoryDayFile(time.Now()); err != nil {
		return "", err
	}
	content, found := activePromptContentForInsertPoint(promptLibrary, bridgeconfig.SystemPromptInsertPointMemory)
	if !found {
		content = defaultMemoryPrompt
	}
	return renderMemorySection(content), nil
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

func renderMemorySection(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "## Memory"
	}
	return "## Memory\n" + trimmed
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
