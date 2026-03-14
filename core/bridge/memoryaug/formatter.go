package memoryaug

import (
	"fmt"
	"strings"
)

func FormatPromptBlock(items []RecallItem) string {
	if len(items) == 0 {
		return ""
	}
	sections := []struct {
		title string
		items []string
	}{
		{title: "Explicit memories"},
		{title: "Session-learned memories"},
		{title: "User-learned memories"},
	}
	for _, item := range items {
		switch {
		case item.Entry.SourceKind == "explicit":
			sections[0].items = append(sections[0].items, formatPromptLine(item, len(items) <= 2))
		case item.Entry.ScopeType == "session":
			sections[1].items = append(sections[1].items, formatPromptLine(item, len(items) <= 2))
		default:
			sections[2].items = append(sections[2].items, formatPromptLine(item, len(items) <= 2))
		}
	}
	lines := make([]string, 0, len(items)+4)
	lines = append(lines, "Memory context:")
	for _, section := range sections {
		lines = append(lines, section.title+":")
		if len(section.items) == 0 {
			lines = append(lines, "- (none)")
			continue
		}
		lines = append(lines, section.items...)
	}
	return strings.Join(lines, "\n")
}

func formatPromptLine(item RecallItem, includeContent bool) string {
	line := fmt.Sprintf(
		"- [%s] %s | source_kind=%s | confidence=%.2f",
		item.Entry.MemoryType,
		strings.TrimSpace(item.Entry.Summary),
		item.Entry.SourceKind,
		item.Entry.Confidence,
	)
	if includeContent && strings.TrimSpace(item.Entry.Content) != "" && item.Entry.Content != item.Entry.Summary {
		line += " | content=" + trimPromptField(item.Entry.Content)
	}
	return line
}

func trimPromptField(raw string) string {
	value := strings.TrimSpace(raw)
	if len(value) <= 160 {
		return value
	}
	return strings.TrimSpace(value[:157]) + "..."
}
