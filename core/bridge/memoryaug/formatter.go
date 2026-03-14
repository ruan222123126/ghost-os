package memoryaug

import (
	"fmt"
	"sort"
	"strings"
)

func FormatPromptBlock(items []RecallItem) string {
	if len(items) == 0 {
		return ""
	}
	slotLines := formatSlotLines(items)
	otherLines := formatOtherMemoryLines(items, len(items) <= 2)
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

func formatOtherMemoryLines(items []RecallItem, includeContent bool) []string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := slotSpecForKey(item.Entry.MemoryKey); ok && slotValueFromEntry(item.Entry) != "" {
			continue
		}
		lines = append(lines, formatOtherPromptLine(item, includeContent))
	}
	return lines
}

func formatOtherPromptLine(item RecallItem, includeContent bool) string {
	line := fmt.Sprintf(
		"- [%s/%s] %s",
		item.Entry.ScopeType,
		item.Entry.MemoryType,
		strings.TrimSpace(item.Entry.Summary),
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
