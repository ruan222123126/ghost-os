package memoryaug

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/memorystore"
)

type scopeRef struct {
	scopeType string
	scopeID   string
}

func (s *recallService) selectRecallItems(items []RecallItem) []RecallItem {
	sort.SliceStable(items, func(i int, j int) bool {
		return compareRecallItem(items[i], items[j])
	})
	selected := make([]RecallItem, 0, s.settings.MaxRecallItems)
	seenIDs := make(map[string]bool, len(items))
	for _, item := range items {
		if seenIDs[item.Entry.ID] || s.isSuppressedByExplicit(item, selected) {
			continue
		}
		seenIDs[item.Entry.ID] = true
		selected = append(selected, item)
		if len(selected) >= s.settings.MaxRecallItems {
			return selected
		}
	}
	return selected
}

func (s *recallService) isSuppressedByExplicit(item RecallItem, selected []RecallItem) bool {
	if item.Entry.SourceKind != memorystore.SourceKindLearned {
		return false
	}
	for _, existing := range selected {
		if existing.Entry.SourceKind == memorystore.SourceKindExplicit &&
			memorySimilarity(item.Entry, existing.Entry) >= 0.88 {
			return true
		}
	}
	return false
}

func (s *recallService) touchSelected(ctx context.Context, items []RecallItem) error {
	explicitIDs := make([]string, 0, len(items))
	learnedIDs := make([]string, 0, len(items))
	for _, item := range items {
		if item.Entry.SourceKind == memorystore.SourceKindExplicit {
			explicitIDs = append(explicitIDs, strings.TrimPrefix(item.Entry.ID, "explicit:"))
			continue
		}
		learnedIDs = append(learnedIDs, item.Entry.ID)
	}
	if err := s.store.TouchExplicitRecords(ctx, explicitIDs); err != nil {
		return err
	}
	return s.store.TouchLearned(ctx, learnedIDs)
}

func buildRecallItem(query string, entry memorystore.MemoryEntry) RecallItem {
	score := computeTextScore(query, entry)
	return RecallItem{
		Entry:     entry,
		TextScore: score,
		Reason: fmt.Sprintf(
			"source=%s scope=%s status=%s score=%.2f confidence=%.2f",
			entry.SourceKind,
			entry.ScopeType,
			entry.Status,
			score,
			entry.Confidence,
		),
	}
}

func buildSlotRecallItem(entry memorystore.MemoryEntry) RecallItem {
	return RecallItem{
		Entry:     entry,
		TextScore: 1,
		SlotMatch: true,
		Reason: fmt.Sprintf(
			"source=%s scope=%s status=%s slot=%s value=%s",
			entry.SourceKind,
			entry.ScopeType,
			entry.Status,
			entry.MemoryKey,
			slotValueFromEntry(entry),
		),
	}
}

func compareRecallItem(left RecallItem, right RecallItem) bool {
	if sourcePriority(left.Entry) != sourcePriority(right.Entry) {
		return sourcePriority(left.Entry) < sourcePriority(right.Entry)
	}
	if slotPriority(left) != slotPriority(right) {
		return slotPriority(left) < slotPriority(right)
	}
	if scopePriority(left.Entry) != scopePriority(right.Entry) {
		return scopePriority(left.Entry) < scopePriority(right.Entry)
	}
	if statusPriority(left.Entry) != statusPriority(right.Entry) {
		return statusPriority(left.Entry) < statusPriority(right.Entry)
	}
	if left.Entry.Confidence != right.Entry.Confidence {
		return left.Entry.Confidence > right.Entry.Confidence
	}
	if left.TextScore != right.TextScore {
		return left.TextScore > right.TextScore
	}
	if !left.Entry.LastUsedAt.Equal(right.Entry.LastUsedAt) {
		return left.Entry.LastUsedAt.After(right.Entry.LastUsedAt)
	}
	if !left.Entry.UpdatedAt.Equal(right.Entry.UpdatedAt) {
		return left.Entry.UpdatedAt.After(right.Entry.UpdatedAt)
	}
	return left.Entry.ID < right.Entry.ID
}

func slotPriority(item RecallItem) int {
	if item.SlotMatch {
		return 0
	}
	return 1
}

func sourcePriority(entry memorystore.MemoryEntry) int {
	if entry.SourceKind == memorystore.SourceKindExplicit {
		return 0
	}
	return 1
}

func scopePriority(entry memorystore.MemoryEntry) int {
	if entry.ScopeType == memorystore.ScopeTypeSession {
		return 0
	}
	return 1
}

func statusPriority(entry memorystore.MemoryEntry) int {
	switch entry.Status {
	case memorystore.MemoryStatusActive:
		return 0
	case memorystore.MemoryStatusSuperseded:
		return 1
	default:
		return 2
	}
}
