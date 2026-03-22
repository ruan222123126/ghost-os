package memorystore

import "strings"

func normalizeEventStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case EventStatusArchived:
		return EventStatusArchived
	case EventStatusActive:
		return EventStatusActive
	default:
		return ""
	}
}

func normalizeEventStatusList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		status := normalizeEventStatus(value)
		if status == "" || seen[status] {
			continue
		}
		seen[status] = true
		out = append(out, status)
	}
	return out
}

func resolveEventStatus(raw string) string {
	if status := normalizeEventStatus(raw); status != "" {
		return status
	}
	return EventStatusActive
}

func normalizeEventEdgeType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case EventEdgeParentOf:
		return EventEdgeParentOf
	case EventEdgeBlocks:
		return EventEdgeBlocks
	case EventEdgeRelatedTo:
		return EventEdgeRelatedTo
	case EventEdgeSameGoal:
		return EventEdgeSameGoal
	default:
		return ""
	}
}
