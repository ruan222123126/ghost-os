package memoryaug

import (
	"strings"

	"ghost-os/bridge/memorystore"
)

func slotValueFromLearnedInput(entry memorystore.LearnedMemoryInput) string {
	value, _ := entry.Metadata[slotMetadataValueKey].(string)
	return strings.TrimSpace(value)
}

func candidateLabel(summary string, content string) string {
	if strings.TrimSpace(summary) != "" {
		return strings.TrimSpace(summary)
	}
	return strings.TrimSpace(content)
}

func appendApplyOutcomeSkipped(outcome *applyOutcome, value string) {
	if outcome == nil || strings.TrimSpace(value) == "" {
		return
	}
	outcome.Skipped = append(outcome.Skipped, value)
}

func appendApplyOutcomeRefreshed(outcome *applyOutcome, value string) {
	if outcome == nil || strings.TrimSpace(value) == "" {
		return
	}
	outcome.Refreshed = append(outcome.Refreshed, value)
}

func appendApplyOutcomeCreated(outcome *applyOutcome, entry memorystore.MemoryEntry) {
	if outcome == nil {
		return
	}
	outcome.Created = append(outcome.Created, entry)
}

func appendApplyOutcomeSuperseded(outcome *applyOutcome, value string) {
	if outcome == nil || strings.TrimSpace(value) == "" {
		return
	}
	outcome.Superseded = append(outcome.Superseded, value)
}

func optionalSingleID(id string) []string {
	if strings.TrimSpace(id) == "" {
		return nil
	}
	return []string{strings.TrimSpace(id)}
}
