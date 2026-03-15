package memoryaug

import (
	"fmt"
	"sort"
	"strings"
)

const (
	promptOtherMemoryLimit     = 2
	promptOtherSummaryMaxChars = 80
	promptOtherSummaryEllipsis = "..."
)

func FormatPromptBlock(items []RecallItem) string {
	if len(items) == 0 {
		return ""
	}
	slotLines := formatSlotLines(items)
	otherLines := formatOtherMemoryLines(items)
	lines := make([]string, 0, len(slotLines)+len(otherLines)+4)
	if len(slotLines) > 0 {
		lines = append(lines, "Memory slots:")
		lines = append(lines, slotLines...)
	}
	if len(otherLines) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "Other memory context:")
		lines = append(lines, otherLines...)
	}
	return strings.Join(lines, "\n")
}

func formatSlotLines(items []RecallItem) []string {
	bestByKey := make(map[string]RecallItem, len(items))
	for _, item := range items {
		spec, ok := slotSpecForKey(item.Entry.MemoryKey)
		if !ok {
			continue
		}
		if slotValueFromEntry(item.Entry) == "" {
			continue
		}
		existing, found := bestByKey[spec.Key]
		if !found || compareRecallItem(item, existing) {
			bestByKey[spec.Key] = item
		}
	}
	if len(bestByKey) == 0 {
		return nil
	}
	specs := make([]SlotSpec, 0, len(bestByKey))
	for _, spec := range knownSlots {
		if _, ok := bestByKey[spec.Key]; ok {
			specs = append(specs, spec)
		}
	}
	sort.SliceStable(specs, func(i int, j int) bool {
		if specs[i].PromptPriority != specs[j].PromptPriority {
			return specs[i].PromptPriority < specs[j].PromptPriority
		}
		return specs[i].Key < specs[j].Key
	})
	lines := make([]string, 0, len(specs))
	for _, spec := range specs {
		lines = append(lines, fmt.Sprintf("- %s=%s", spec.Key, slotValueFromEntry(bestByKey[spec.Key].Entry)))
	}
	return lines
}

func formatOtherMemoryLines(items []RecallItem) []string {
	lines := make([]string, 0, promptOtherMemoryLimit)
	for _, item := range items {
		if isPromptSlotItem(item) {
			continue
		}
		lines = append(lines, formatOtherPromptLine(item))
		if len(lines) >= promptOtherMemoryLimit {
			return lines
		}
	}
	return lines
}

func isPromptSlotItem(item RecallItem) bool {
	_, ok := slotSpecForKey(item.Entry.MemoryKey)
	return ok && slotValueFromEntry(item.Entry) != ""
}

func formatOtherPromptLine(item RecallItem) string {
	return fmt.Sprintf(
		"- [%s/%s] %s",
		item.Entry.ScopeType,
		item.Entry.MemoryType,
		trimPromptSummary(item.Entry.Summary),
	)
}

func trimPromptSummary(raw string) string {
	value := strings.TrimSpace(raw)
	if len(value) <= promptOtherSummaryMaxChars {
		return value
	}
	limit := promptOtherSummaryMaxChars - len(promptOtherSummaryEllipsis)
	return strings.TrimSpace(value[:limit]) + promptOtherSummaryEllipsis
}
