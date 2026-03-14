package memoryaug

import (
	"context"
	"errors"

	"ghost-os/bridge/memorystore"
)

type learningService struct {
	settings  Settings
	store     learningStore
	extractor CandidateExtractor
}

func NewLearningService(settings Settings, store learningStore, extractor CandidateExtractor) LearningService {
	return &learningService{
		settings:  normalizeSettings(settings),
		store:     store,
		extractor: extractor,
	}
}

func (s *learningService) LearnFromTurn(ctx context.Context, input LearnFromTurnInput) error {
	if !s.settings.Enabled || !s.settings.LearningEnabled {
		return nil
	}
	if s.store == nil {
		return errors.New("memory learning store is not configured")
	}
	if s.extractor == nil {
		return errors.New("memory candidate extractor is not configured")
	}
	return s.learn(ctx, normalizeLearnInput(input, s.settings))
}

func (s *learningService) learn(ctx context.Context, input LearnFromTurnInput) error {
	filtered := filterTurnMessages(input.Messages)
	if len(filtered) == 0 {
		return s.recordSkipped(ctx, input, filtered, "no durable candidates after rule filter")
	}
	policy := deriveLearningPolicy(filtered)
	if !policy.shouldLearn {
		return s.recordSkipped(ctx, input, filtered, "turn does not contain a stable slot signal")
	}
	explicitContext, learnedContext, err := s.loadLearningContext(ctx, input, filtered)
	if err != nil {
		return s.recordError(ctx, input, filtered, "", err)
	}
	output, err := s.extractor.Extract(ctx, ExtractInput{
		SessionID:        input.SessionID,
		UserScopeID:      input.UserScope,
		Transcript:       filtered,
		ExistingExplicit: explicitContext,
		ExistingLearned:  learnedContext,
	})
	if err != nil {
		return s.recordError(ctx, input, filtered, "", err)
	}
	outcome, err := s.applyCandidates(ctx, input, output.Items, explicitContext, learnedContext)
	if err != nil {
		return s.recordError(ctx, input, filtered, output.RawJSON, err)
	}
	return s.recordOutcome(ctx, input, filtered, output.RawJSON, outcome)
}

func (s *learningService) loadLearningContext(
	ctx context.Context,
	input LearnFromTurnInput,
	filtered []TurnMessage,
) ([]memorystore.MemoryEntry, []memorystore.MemoryEntry, error) {
	query := buildTranscriptText(filtered)
	explicit, err := s.store.SearchExplicitRecallRecords(ctx, query, 12)
	if err != nil {
		return nil, nil, err
	}
	learned := make([]memorystore.MemoryEntry, 0, 24)
	if s.settings.SessionScopeEnabled {
		items, _, err := s.store.ListLearned(ctx, memorystore.LearnedListFilter{
			ScopeType: memorystore.ScopeTypeSession,
			ScopeID:   input.SessionID,
			Statuses:  []string{memorystore.MemoryStatusActive},
			Query:     query,
			Limit:     12,
		})
		if err != nil {
			return nil, nil, err
		}
		learned = append(learned, items...)
	}
	if s.settings.UserScopeEnabled {
		items, _, err := s.store.ListLearned(ctx, memorystore.LearnedListFilter{
			ScopeType: memorystore.ScopeTypeUser,
			ScopeID:   input.UserScope,
			Statuses:  []string{memorystore.MemoryStatusActive},
			Query:     query,
			Limit:     12,
		})
		if err != nil {
			return nil, nil, err
		}
		learned = append(learned, items...)
	}
	return explicit, learned, nil
}

func (s *learningService) applyCandidates(
	ctx context.Context,
	input LearnFromTurnInput,
	candidates []Candidate,
	explicit []memorystore.MemoryEntry,
	existing []memorystore.MemoryEntry,
) (applyOutcome, error) {
	outcome := applyOutcome{}
	for _, candidate := range candidates {
		if !s.acceptsCandidate(candidate) {
			outcome.Skipped = append(outcome.Skipped, candidateLabel(candidate))
			continue
		}
		entry, supersedes, ok := s.normalizeCandidate(candidate, input)
		if !ok {
			outcome.Skipped = append(outcome.Skipped, candidateLabel(candidate))
			continue
		}
		if s.duplicatesExplicit(entry, explicit) {
			outcome.Skipped = append(outcome.Skipped, entry.Summary)
			continue
		}
		if refreshedID := s.findExactLearnedMatch(entry, existing); refreshedID != "" {
			if err := s.store.RefreshLearned(ctx, refreshedID, entry.Confidence); err != nil {
				return outcome, err
			}
			outcome.Refreshed = append(outcome.Refreshed, refreshedID)
			continue
		}
		validSupersedes, err := s.resolveSupersedes(ctx, entry, supersedes, existing)
		if err != nil {
			return outcome, err
		}
		created, err := s.store.CreateLearned(ctx, memorystore.LearnedMemoryInput{
			ScopeType:  entry.ScopeType,
			ScopeID:    entry.ScopeID,
			MemoryType: entry.MemoryType,
			MemoryKey:  entry.MemoryKey,
			Content:    entry.Content,
			Summary:    entry.Summary,
			Metadata:   entry.Metadata,
			Confidence: entry.Confidence,
		}, validSupersedes)
		if err != nil {
			return outcome, err
		}
		existing = append(existing, created)
		outcome.Created = append(outcome.Created, created)
		outcome.Superseded = append(outcome.Superseded, validSupersedes...)
	}
	return outcome, nil
}
