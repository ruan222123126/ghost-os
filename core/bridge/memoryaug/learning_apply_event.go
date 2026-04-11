package memoryaug

import (
	"context"
	"errors"
	"strings"

	"ghost-os/bridge/memorystore"
)

const (
	eventMemoryMatchSimilarityThreshold     = 0.9
	eventMemorySupersedeSimilarityThreshold = 0.82
)

func (s *learningService) applyEventCandidates(
	ctx context.Context,
	eventID string,
	existing []memorystore.EventMemory,
	candidates []EventMemoryCandidate,
	outcome *applyOutcome,
) error {
	for _, candidate := range candidates {
		entry, ok := buildEventMemoryInput(candidate, eventID, s.settings.MinConfidence)
		if !ok {
			outcome.Skipped = append(outcome.Skipped, candidateLabel(candidate.Summary, candidate.Content))
			continue
		}
		refreshedID := findMatchingEventMemory(existing, entry)
		if refreshedID != "" {
			if err := s.store.RefreshEventMemory(ctx, refreshedID, entry.Confidence); err != nil {
				return err
			}
			outcome.EventRefreshed = append(outcome.EventRefreshed, refreshedID)
			continue
		}
		supersedesID, err := resolveEventMemorySupersedes(ctx, s.store, existing, entry)
		if err != nil {
			return err
		}
		created, err := s.store.CreateEventMemory(ctx, entry, optionalSingleID(supersedesID))
		if err != nil {
			return err
		}
		existing = append(existing, created)
		outcome.EventCreated = append(outcome.EventCreated, created)
		if supersedesID != "" {
			outcome.Superseded = append(outcome.Superseded, supersedesID)
		}
	}
	return nil
}

func buildEventMemoryInput(candidate EventMemoryCandidate, eventID string, minConfidence float64) (memorystore.EventMemoryInput, bool) {
	if candidate.Confidence < minConfidence {
		return memorystore.EventMemoryInput{}, false
	}
	memoryType := resolveCandidateMemoryType(candidate.MemoryType)
	if strings.TrimSpace(candidate.Summary) == "" || strings.TrimSpace(candidate.Content) == "" {
		return memorystore.EventMemoryInput{}, false
	}
	return memorystore.EventMemoryInput{
		EventID:    eventID,
		MemoryType: memoryType,
		MemoryKey:  candidate.MemoryKey,
		Summary:    candidate.Summary,
		Content:    candidate.Content,
		Metadata:   map[string]any{"learn_reason": strings.TrimSpace(candidate.Reason)},
		Confidence: candidate.Confidence,
	}, true
}

func findMatchingEventMemory(existing []memorystore.EventMemory, entry memorystore.EventMemoryInput) string {
	for _, item := range existing {
		if item.MemoryKey != "" && item.MemoryKey == memorystore.NormalizeMemoryKey(entry.MemoryKey) {
			if item.Content == strings.TrimSpace(entry.Content) {
				return item.ID
			}
		}
		if eventMemorySimilarity(item, entry) >= eventMemoryMatchSimilarityThreshold {
			return item.ID
		}
	}
	return ""
}

func resolveEventMemorySupersedes(
	ctx context.Context,
	store learningStore,
	existing []memorystore.EventMemory,
	entry memorystore.EventMemoryInput,
) (string, error) {
	if key := memorystore.NormalizeMemoryKey(entry.MemoryKey); key != "" {
		item, err := store.FindActiveEventMemoryByKey(ctx, entry.EventID, key)
		switch {
		case err == nil:
			return item.ID, nil
		case !errors.Is(err, memorystore.ErrNotFound):
			return "", err
		}
	}
	for _, item := range existing {
		if eventMemorySimilarity(item, entry) >= eventMemorySupersedeSimilarityThreshold {
			return item.ID, nil
		}
	}
	return "", nil
}

func eventMemorySimilarity(existing memorystore.EventMemory, candidate memorystore.EventMemoryInput) float64 {
	return memorySimilarity(
		memorystore.MemoryEntry{Summary: existing.Summary, Content: existing.Content},
		memorystore.MemoryEntry{Summary: candidate.Summary, Content: candidate.Content},
	)
}

func resolveCandidateMemoryType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case memorystore.MemoryTypePreference:
		return memorystore.MemoryTypePreference
	case memorystore.MemoryTypeWorkflow:
		return memorystore.MemoryTypeWorkflow
	case memorystore.MemoryTypeProfile:
		return memorystore.MemoryTypeProfile
	default:
		return memorystore.MemoryTypeFact
	}
}
