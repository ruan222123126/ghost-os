package memoryaug

import (
	"fmt"
	"strings"

	"ghost-os/bridge/memorystore"
)

func normalizeEventRecallInput(input RecallInput) (RecallInput, error) {
	activeIDs := normalizeIDs(input.ActiveEventIDs)
	primaryEventID := strings.TrimSpace(input.PrimaryEventID)
	if primaryEventID == "" {
		return RecallInput{}, fmt.Errorf("primary_event_id is required")
	}
	if len(activeIDs) == 0 {
		activeIDs = []string{primaryEventID}
	}
	if activeIDs[0] != primaryEventID {
		activeIDs = append([]string{primaryEventID}, filterEventIDs(activeIDs, primaryEventID)...)
	}
	if len(activeIDs) > maxActiveEventCount {
		return RecallInput{}, fmt.Errorf("active_event_ids cannot exceed 3")
	}
	return RecallInput{
		SessionID:      strings.TrimSpace(input.SessionID),
		PrimaryEventID: primaryEventID,
		ActiveEventIDs: activeIDs,
		FocusText:      strings.TrimSpace(input.FocusText),
		RecallPlan:     normalizeRecallPlan(input.RecallPlan, activeIDs),
	}, nil
}

func normalizeRecallPlan(plan RecallPlan, activeIDs []string) RecallPlan {
	normalized := plan
	activeSet := make(map[string]struct{}, len(activeIDs))
	for _, id := range normalizeIDs(activeIDs) {
		activeSet[id] = struct{}{}
	}
	filtered := make([]string, 0, len(plan.EventIDs))
	for _, id := range normalizeIDs(plan.EventIDs) {
		if _, ok := activeSet[id]; !ok {
			continue
		}
		filtered = append(filtered, id)
	}
	normalized.EventIDs = filtered
	if len(normalized.EventIDs) == 0 {
		normalized.EventIDs = append([]string(nil), activeIDs...)
	}
	return normalized
}

func planRecallsEvent(plan RecallPlan, eventID string) bool {
	if len(plan.EventIDs) == 0 {
		return true
	}
	for _, id := range plan.EventIDs {
		if id == eventID {
			return true
		}
	}
	return false
}

func allowsMemoryType(plan RecallPlan, memoryType string) bool {
	switch memoryType {
	case memorystore.MemoryTypeWorkflow:
		return plan.IncludeWorkflow
	case memorystore.MemoryTypePreference:
		return plan.IncludePreference
	case memorystore.MemoryTypeProfile:
		return plan.IncludeProfile
	case memorystore.MemoryTypeFact:
		return plan.IncludeFact
	default:
		return false
	}
}

func planIncludesEventMemory(plan RecallPlan) bool {
	return plan.IncludeWorkflow || plan.IncludePreference || plan.IncludeFact || plan.IncludeProfile
}

func computeEventMemoryScore(query string, entry memorystore.EventMemory) float64 {
	memoryEntry := memorystore.MemoryEntry{
		ID:         entry.ID,
		MemoryType: entry.MemoryType,
		MemoryKey:  entry.MemoryKey,
		Content:    entry.Content,
		Summary:    entry.Summary,
	}
	return computeTextScore(query, memoryEntry)
}

func compareScoredEventMemory(left scoredEventMemory, right scoredEventMemory) bool {
	if left.textScore != right.textScore {
		return left.textScore > right.textScore
	}
	if left.hit.Entry.Confidence != right.hit.Entry.Confidence {
		return left.hit.Entry.Confidence > right.hit.Entry.Confidence
	}
	if !left.hit.Entry.LastUsedAt.Equal(right.hit.Entry.LastUsedAt) {
		return left.hit.Entry.LastUsedAt.After(right.hit.Entry.LastUsedAt)
	}
	if !left.hit.Entry.UpdatedAt.Equal(right.hit.Entry.UpdatedAt) {
		return left.hit.Entry.UpdatedAt.After(right.hit.Entry.UpdatedAt)
	}
	return left.hit.Entry.ID < right.hit.Entry.ID
}

func filterEventIDs(ids []string, excluded string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == excluded {
			continue
		}
		out = append(out, id)
	}
	return out
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func roundShare(total int, weight int, totalWeight int) int {
	if total <= 0 || weight <= 0 || totalWeight <= 0 {
		return 0
	}
	return (total*weight + totalWeight/2) / totalWeight
}
