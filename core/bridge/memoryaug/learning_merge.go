package memoryaug

import (
	"context"
	"errors"

	"ghost-os/bridge/memorystore"
)

func (s *learningService) duplicatesExplicit(candidate memorystore.MemoryEntry, explicit []memorystore.MemoryEntry) bool {
	for _, existing := range explicit {
		if candidate.ScopeType != existing.ScopeType || candidate.ScopeID != existing.ScopeID {
			continue
		}
		if memorySimilarity(candidate, existing) >= 0.88 {
			return true
		}
	}
	return false
}

func (s *learningService) findExactLearnedMatch(candidate memorystore.MemoryEntry, existing []memorystore.MemoryEntry) string {
	for _, item := range existing {
		if item.ScopeType != candidate.ScopeType || item.ScopeID != candidate.ScopeID {
			continue
		}
		if item.Status != memorystore.MemoryStatusActive || item.MemoryKey != candidate.MemoryKey {
			continue
		}
		if slotValueFromEntry(item) == slotValueFromEntry(candidate) {
			return item.ID
		}
	}
	return ""
}

func (s *learningService) resolveSupersedes(
	ctx context.Context,
	candidate memorystore.MemoryEntry,
	supersedes []string,
	existing []memorystore.MemoryEntry,
) ([]string, error) {
	candidateIDs := normalizeIDs(supersedes)
	if len(candidateIDs) == 0 {
		return s.autoSupersedes(ctx, candidate, existing)
	}
	items, err := s.store.GetLearnedByIDs(ctx, candidateIDs)
	if err != nil {
		return nil, err
	}
	valid := make([]string, 0, len(items))
	for _, item := range items {
		if item.ScopeType != candidate.ScopeType || item.ScopeID != candidate.ScopeID {
			continue
		}
		if item.Status != memorystore.MemoryStatusActive {
			continue
		}
		valid = append(valid, item.ID)
	}
	return valid, nil
}

func (s *learningService) autoSupersedes(
	ctx context.Context,
	candidate memorystore.MemoryEntry,
	existing []memorystore.MemoryEntry,
) ([]string, error) {
	entry, err := s.store.FindActiveLearnedByMemoryKey(ctx, candidate.ScopeType, candidate.ScopeID, candidate.MemoryKey)
	switch {
	case err == nil:
		return []string{entry.ID}, nil
	case errors.Is(err, memorystore.ErrNotFound):
	default:
		return nil, err
	}
	return s.findSimilarLegacyMatch(candidate, existing), nil
}

func (s *learningService) findSimilarLegacyMatch(
	candidate memorystore.MemoryEntry,
	existing []memorystore.MemoryEntry,
) []string {
	for _, item := range existing {
		if item.ScopeType != candidate.ScopeType || item.ScopeID != candidate.ScopeID {
			continue
		}
		if item.MemoryType != candidate.MemoryType || item.Status != memorystore.MemoryStatusActive {
			continue
		}
		if _, ok := slotSpecForKey(item.MemoryKey); ok {
			continue
		}
		if matchesLegacySlotCandidate(candidate, item) {
			return []string{item.ID}
		}
	}
	return nil
}

func matchesLegacySlotCandidate(candidate memorystore.MemoryEntry, existing memorystore.MemoryEntry) bool {
	if memorySimilarity(candidate, existing) >= 0.8 {
		return true
	}
	return summarySimilarity(candidate, existing) >= 0.6 && memorySimilarity(candidate, existing) >= 0.4
}

func summarySimilarity(left memorystore.MemoryEntry, right memorystore.MemoryEntry) float64 {
	return memorySimilarity(
		memorystore.MemoryEntry{Summary: left.Summary},
		memorystore.MemoryEntry{Summary: right.Summary},
	)
}
