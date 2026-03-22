package memoryaug

import (
	"fmt"
	"strings"

	"ghost-os/bridge/memorystore"
)

const (
	promptSummaryMaxChars = 96
	promptSummaryEllipsis = "..."
)

func FormatPromptBlock(output RecallOutput) string {
	lines := make([]string, 0, 16)
	activeLines := formatActiveEventLines(output)
	if len(activeLines) > 0 {
		lines = append(lines, "Active event:")
		lines = append(lines, activeLines...)
	}
	memoryLines := formatRelevantEventMemoryLines(output)
	if len(memoryLines) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "Relevant event memory:")
		lines = append(lines, memoryLines...)
	}
	preferenceLines := formatGlobalPreferenceLines(output.GlobalPreferences)
	if len(preferenceLines) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "Global preferences:")
		lines = append(lines, preferenceLines...)
	}
	return strings.Join(lines, "\n")
}

func formatActiveEventLines(output RecallOutput) []string {
	lines := make([]string, 0, 4)
	if output.PrimaryEvent != nil {
		lines = append(lines, fmt.Sprintf("- primary: %s", trimPromptSummary(output.PrimaryEvent.Event.Title)))
		if summary := trimPromptSummary(output.PrimaryEvent.Event.Summary); summary != "" {
			lines = append(lines, fmt.Sprintf("- summary: %s", summary))
		}
	}
	for _, adjacent := range output.AdjacentEvents {
		lines = append(lines, fmt.Sprintf("- adjacent: %s", trimPromptSummary(adjacent.Event.Title)))
	}
	return lines
}

func formatRelevantEventMemoryLines(output RecallOutput) []string {
	lines := make([]string, 0, len(output.PrimaryMemories)+len(output.AdjacentMemories))
	for _, hit := range output.PrimaryMemories {
		lines = append(lines, formatEventMemoryLine("primary", hit))
	}
	for _, hit := range output.AdjacentMemories {
		lines = append(lines, formatEventMemoryLine("adjacent", hit))
	}
	return lines
}

func formatEventMemoryLine(role string, hit RecallMemoryHit) string {
	label := trimPromptSummary(hit.Event.Title)
	if label == "" {
		label = hit.Event.ID
	}
	return fmt.Sprintf(
		"- [%s/%s] %s: %s",
		role,
		hit.Entry.MemoryType,
		label,
		trimPromptSummary(hit.Entry.Summary),
	)
}

func formatGlobalPreferenceLines(items []memorystore.MemoryEntry) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		value := slotValueFromEntry(item)
		if value == "" {
			value = trimPromptSummary(item.Summary)
		}
		if item.MemoryKey == "" || value == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s=%s", item.MemoryKey, value))
	}
	return lines
}

func trimPromptSummary(raw string) string {
	value := strings.TrimSpace(raw)
	if len(value) <= promptSummaryMaxChars {
		return value
	}
	limit := promptSummaryMaxChars - len(promptSummaryEllipsis)
	return strings.TrimSpace(value[:limit]) + promptSummaryEllipsis
}
