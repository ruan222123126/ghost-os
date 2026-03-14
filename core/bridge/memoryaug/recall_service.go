package memoryaug

import (
	"context"
	"errors"
	"fmt"
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
	slotMatches, err := s.loadSlotCandidates(ctx, input)
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, slotMatches...)
	learned, err := s.loadLearnedCandidates(ctx, input, limit)
	if err != nil {
		return nil, err
	}
	for _, entry := range learned {
		candidates = append(candidates, buildRecallItem(input.Query, entry))
	}
	return candidates, nil
}

func (s *recallService) loadSlotCandidates(ctx context.Context, input RecallInput) ([]RecallItem, error) {
	specs := matchSlotsForQuery(input.Query)
	if len(specs) == 0 {
		return nil, nil
	}
	items := make([]RecallItem, 0, len(specs)*2)
	for _, spec := range specs {
		matched, err := s.loadSlotCandidate(ctx, input, spec)
		if err != nil {
			return nil, err
		}
		items = append(items, matched...)
	}
	return items, nil
}

func (s *recallService) loadSlotCandidate(ctx context.Context, input RecallInput, spec SlotSpec) ([]RecallItem, error) {
	scopes := s.slotScopesForRecall(spec, input)
	items := make([]RecallItem, 0, len(scopes))
	for _, scope := range scopes {
		entry, err := s.store.FindActiveLearnedByMemoryKey(ctx, scope.scopeType, scope.scopeID, spec.Key)
		switch {
		case err == nil:
			items = append(items, buildSlotRecallItem(entry))
		case errors.Is(err, memorystore.ErrNotFound):
			continue
		default:
			return nil, err
		}
	}
	return items, nil
}

func (s *recallService) slotScopesForRecall(spec SlotSpec, input RecallInput) []scopeRef {
	if spec.DefaultScope == memorystore.ScopeTypeSession {
		out := make([]scopeRef, 0, 2)
		if s.settings.SessionScopeEnabled {
			out = append(out, scopeRef{scopeType: memorystore.ScopeTypeSession, scopeID: input.SessionID})
		}
		if s.settings.UserScopeEnabled {
			out = append(out, scopeRef{scopeType: memorystore.ScopeTypeUser, scopeID: input.UserScope})
		}
		return out
	}
	if s.settings.UserScopeEnabled {
		return []scopeRef{{scopeType: memorystore.ScopeTypeUser, scopeID: input.UserScope}}
	}
	return []scopeRef{{scopeType: memorystore.ScopeTypeSession, scopeID: input.SessionID}}
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
