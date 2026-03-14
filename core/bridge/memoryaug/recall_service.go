package memoryaug

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/memorystore"
)

type recallService struct {
	settings Settings
	store    recallStore
}

func NewRecallService(settings Settings, store recallStore) RecallService {
	return &recallService{
		settings: normalizeSettings(settings),
		store:    store,
	}
}

func (s *recallService) Recall(ctx context.Context, input RecallInput) ([]RecallItem, error) {
	if !s.settings.Enabled || !s.settings.RecallEnabled {
		return nil, nil
	}
	if s.store == nil {
		return nil, fmt.Errorf("memory recall store is not configured")
	}
	normalized := normalizeRecallInput(input, s.settings)
	if strings.TrimSpace(normalized.Query) == "" {
		return nil, nil
	}
	items, err := s.collectRecallItems(ctx, normalized)
	if err != nil {
		return nil, err
	}
	selected := s.selectRecallItems(items)
	if err := s.touchSelected(ctx, selected); err != nil {
		return nil, err
	}
	return selected, nil
}

func (s *recallService) collectRecallItems(ctx context.Context, input RecallInput) ([]RecallItem, error) {
	limit := maxInt(16, s.settings.MaxRecallItems*4)
	explicit, err := s.store.SearchExplicitRecallRecords(ctx, input.Query, limit)
	if err != nil {
		return nil, err
	}
	candidates := make([]RecallItem, 0, len(explicit)+limit)
	candidates = append(candidates, s.filterExplicitCandidates(input, explicit)...)
	learned, err := s.loadLearnedCandidates(ctx, input, limit)
	if err != nil {
		return nil, err
	}
	for _, entry := range learned {
		candidates = append(candidates, buildRecallItem(input.Query, entry))
	}
	return candidates, nil
}

func (s *recallService) filterExplicitCandidates(input RecallInput, items []memorystore.MemoryEntry) []RecallItem {
	out := make([]RecallItem, 0, len(items))
	for _, item := range items {
		if item.ScopeType == memorystore.ScopeTypeSession && item.ScopeID != input.SessionID {
			continue
		}
		if item.ScopeType == memorystore.ScopeTypeUser && item.ScopeID != input.UserScope {
			continue
		}
		out = append(out, buildRecallItem(input.Query, item))
	}
	return out
}

func (s *recallService) loadLearnedCandidates(ctx context.Context, input RecallInput, limit int) ([]memorystore.MemoryEntry, error) {
	out := make([]memorystore.MemoryEntry, 0, limit)
	if s.settings.SessionScopeEnabled {
		items, _, err := s.store.ListLearned(ctx, memorystore.LearnedListFilter{
			ScopeType: memorystore.ScopeTypeSession,
			ScopeID:   input.SessionID,
			Statuses:  []string{memorystore.MemoryStatusActive, memorystore.MemoryStatusSuperseded},
			Query:     input.Query,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	if s.settings.UserScopeEnabled {
		items, _, err := s.store.ListLearned(ctx, memorystore.LearnedListFilter{
			ScopeType: memorystore.ScopeTypeUser,
			ScopeID:   input.UserScope,
			Statuses:  []string{memorystore.MemoryStatusActive, memorystore.MemoryStatusSuperseded},
			Query:     input.Query,
			Limit:     limit,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	return out, nil
}

func (s *recallService) selectRecallItems(items []RecallItem) []RecallItem {
	sort.SliceStable(items, func(i int, j int) bool {
		return compareRecallItem(items[i], items[j])
	})
	selected := make([]RecallItem, 0, s.settings.MaxRecallItems)
	for _, item := range items {
		if s.isSuppressedByExplicit(item, selected) {
			continue
		}
		selected = append(selected, item)
		if len(selected) >= s.settings.MaxRecallItems {
			break
		}
	}
	return selected
}

func (s *recallService) isSuppressedByExplicit(item RecallItem, selected []RecallItem) bool {
	if item.Entry.SourceKind != memorystore.SourceKindLearned {
		return false
	}
	for _, existing := range selected {
		if existing.Entry.SourceKind != memorystore.SourceKindExplicit {
			continue
		}
		if memorySimilarity(item.Entry, existing.Entry) >= 0.88 {
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
	reason := fmt.Sprintf(
		"source=%s scope=%s status=%s score=%.2f confidence=%.2f",
		entry.SourceKind,
		entry.ScopeType,
		entry.Status,
		score,
		entry.Confidence,
	)
	return RecallItem{
		Entry:     entry,
		TextScore: score,
		Reason:    reason,
	}
}

func compareRecallItem(left RecallItem, right RecallItem) bool {
	if sourcePriority(left.Entry) != sourcePriority(right.Entry) {
		return sourcePriority(left.Entry) < sourcePriority(right.Entry)
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

func normalizeRecallInput(input RecallInput, settings Settings) RecallInput {
	userScope := strings.TrimSpace(input.UserScope)
	if userScope == "" {
		userScope = settings.UserScopeID
	}
	return RecallInput{
		SessionID: strings.TrimSpace(input.SessionID),
		UserScope: userScope,
		Query:     strings.TrimSpace(input.Query),
	}
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
