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
