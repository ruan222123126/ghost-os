package runtime

import (
	"errors"
	goruntime "runtime"
	"strconv"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	ctxmgr "ghost-os/bridge/context"
)

var errContextPlaceholderRequired = errors.New(
	"active context prompt cards require system.default to include {{context}}",
)

func resolveContextSection(
	cfg Config,
	promptManager *ctxmgr.PromptManager,
	promptLibrary []bridgeconfig.SystemPromptLibraryItem,
) (string, bool, error) {
	activeCount, contents := activeContextPromptContents(promptLibrary)
	if activeCount == 0 {
		return "", false, nil
	}
	if promptManager == nil || !promptManager.ReferencesVariable("context") {
		return "", false, errContextPlaceholderRequired
	}

	parts := []string{defaultPromptContextSection(cfg)}
	parts = append(parts, contents...)
	return strings.Join(parts, "\n\n"), true, nil
}

func activeContextPromptContents(
	promptLibrary []bridgeconfig.SystemPromptLibraryItem,
) (int, []string) {
	count := 0
	contents := make([]string, 0, len(promptLibrary))
	for _, item := range promptLibrary {
		if item.InsertPoint != bridgeconfig.SystemPromptInsertPointContext || !item.Active {
			continue
		}
		count += 1
		content := strings.TrimSpace(item.Content)
		if content != "" {
			contents = append(contents, content)
		}
	}
	return count, contents
}

func resolvePromptContextSection(cfg Config, override string) string {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return trimmed
	}
	return defaultPromptContextSection(cfg)
}

func defaultPromptContextSection(cfg Config) string {
	return strings.Join([]string{
		"OS: " + goruntime.GOOS,
		"Root: " + resolvePromptProjectRoot(cfg.ProjectRoot),
		"Max turns: " + strconv.Itoa(cfg.MaxTurns),
	}, " | ")
}
