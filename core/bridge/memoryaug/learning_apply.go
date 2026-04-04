package memoryaug

import (
	"context"
	"errors"
	"strings"

	"ghost-os/bridge/memorystore"
)

type globalCandidateApplyInput struct {
	UserScope  string
	Existing   []memorystore.MemoryEntry
	Candidates []Candidate
	Outcome    *applyOutcome
}

type globalCandidateApplyState struct {
	userScope string
	existing  []memorystore.MemoryEntry
	outcome   *applyOutcome
}

func (s *learningService) applyGlobalCandidates(
	ctx context.Context,
	input globalCandidateApplyInput,
) error {
	state := globalCandidateApplyState{
		userScope: strings.TrimSpace(input.UserScope),
		existing:  append([]memorystore.MemoryEntry(nil), input.Existing...),
		outcome:   input.Outcome,
	}
	for _, candidate := range input.Candidates {
		if err := s.applyGlobalCandidate(ctx, &state, candidate); err != nil {
			return err
		}
	}
	return nil
}

func (s *learningService) applyGlobalCandidate(
	ctx context.Context,
	state *globalCandidateApplyState,
	candidate Candidate,
) error {
	if candidate.Confidence < s.settings.MinConfidence {
		appendApplyOutcomeSkipped(state.outcome, candidateLabel(candidate.Summary, candidate.Content))
		return nil
	}
	entry, ok := buildGlobalPreferenceInput(candidate, state.userScope)
	if !ok {
		appendApplyOutcomeSkipped(state.outcome, candidateLabel(candidate.Summary, candidate.Content))
		return nil
	}
	if hasExplicitGlobalPreference(state.existing, entry.MemoryKey) {
		appendApplyOutcomeSkipped(state.outcome, entry.MemoryKey)
		return nil
	}
	refreshedID := findMatchingGlobalPreference(state.existing, entry)
	if refreshedID != "" {
		if err := s.store.RefreshLearned(ctx, refreshedID, entry.Confidence); err != nil {
			return err
		}
		appendApplyOutcomeRefreshed(state.outcome, refreshedID)
		return nil
	}
	return s.createGlobalPreference(ctx, state, entry)
}

func (s *learningService) createGlobalPreference(
	ctx context.Context,
	state *globalCandidateApplyState,
	entry memorystore.LearnedMemoryInput,
) error {
	supersedesID, err := resolveGlobalPreferenceSupersedes(ctx, s.store, entry.MemoryKey, state.userScope)
	if err != nil {
		return err
	}
	created, err := s.store.CreateLearned(ctx, entry, optionalSingleID(supersedesID))
	if err != nil {
		return err
	}
	state.existing = append(state.existing, created)
	appendApplyOutcomeCreated(state.outcome, created)
	if supersedesID != "" {
		appendApplyOutcomeSuperseded(state.outcome, supersedesID)
	}
	return nil
}

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

func buildGlobalPreferenceInput(candidate Candidate, userScope string) (memorystore.LearnedMemoryInput, bool) {
	spec, ok := slotSpecForKey(candidate.MemoryKey)
	if !ok {
		return memorystore.LearnedMemoryInput{}, false
	}
	value := normalizeSlotValue(spec, candidate)
	if value == "" {
		return memorystore.LearnedMemoryInput{}, false
	}
	return memorystore.LearnedMemoryInput{
		ScopeType:  memorystore.ScopeTypeUser,
		ScopeID:    userScope,
		MemoryType: spec.MemoryType,
		MemoryKey:  spec.Key,
		Content:    renderSlotContent(spec, value),
		Summary:    spec.Summary,
		Metadata:   buildSlotMetadata(value, candidate.Reason),
		Confidence: candidate.Confidence,
	}, true
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

func hasExplicitGlobalPreference(existing []memorystore.MemoryEntry, memoryKey string) bool {
	for _, item := range existing {
		if item.SourceKind == memorystore.SourceKindExplicit && item.MemoryKey == memoryKey {
			return true
		}
	}
	return false
}

func findMatchingGlobalPreference(existing []memorystore.MemoryEntry, entry memorystore.LearnedMemoryInput) string {
	for _, item := range existing {
		if item.SourceKind != memorystore.SourceKindLearned || item.MemoryKey != entry.MemoryKey {
			continue
		}
		if slotValueFromEntry(item) == slotValueFromLearnedInput(entry) {
			return item.ID
		}
	}
	return ""
}

func findMatchingEventMemory(existing []memorystore.EventMemory, entry memorystore.EventMemoryInput) string {
	for _, item := range existing {
		if item.MemoryKey != "" && item.MemoryKey == memorystore.NormalizeMemoryKey(entry.MemoryKey) {
			if item.Content == strings.TrimSpace(entry.Content) {
				return item.ID
			}
		}
		if eventMemorySimilarity(item, entry) >= 0.9 {
			return item.ID
		}
	}
	return ""
}

func resolveGlobalPreferenceSupersedes(
	ctx context.Context,
	store learningStore,
	memoryKey string,
	userScope string,
) (string, error) {
	entry, err := store.FindActiveLearnedByMemoryKey(ctx, memorystore.ScopeTypeUser, userScope, memoryKey)
	switch {
	case err == nil:
		return entry.ID, nil
	case errors.Is(err, memorystore.ErrNotFound):
		return "", nil
	default:
		return "", err
	}
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
		if eventMemorySimilarity(item, entry) >= 0.82 {
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
