package memoryaug

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
			outcome.Skipped = append(outcome.Skipped, strings.TrimSpace(candidate.Summary))
			continue
		}
		entry, supersedes := s.normalizeCandidate(candidate, input)
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

func (s *learningService) acceptsCandidate(candidate Candidate) bool {
	return strings.TrimSpace(candidate.Summary) != "" &&
		strings.TrimSpace(candidate.Content) != "" &&
		candidate.Confidence >= s.settings.MinConfidence
}

func (s *learningService) normalizeCandidate(candidate Candidate, input LearnFromTurnInput) (memorystore.MemoryEntry, []string) {
	scopeType := s.resolveCandidateScope(candidate.ScopeType)
	scopeID := input.UserScope
	if scopeType == memorystore.ScopeTypeSession {
		scopeID = input.SessionID
	}
	return memorystore.MemoryEntry{
		ScopeType:  scopeType,
		ScopeID:    scopeID,
		SourceKind: memorystore.SourceKindLearned,
		MemoryType: normalizeCandidateMemoryType(candidate.MemoryType),
		Summary:    strings.TrimSpace(candidate.Summary),
		Content:    strings.TrimSpace(candidate.Content),
		Metadata:   map[string]any{"learn_reason": strings.TrimSpace(candidate.Reason)},
		Confidence: candidate.Confidence,
		Status:     memorystore.MemoryStatusActive,
	}, candidate.SupersedesID
}

func (s *learningService) resolveCandidateScope(raw string) string {
	scopeType := strings.ToLower(strings.TrimSpace(raw))
	if scopeType == memorystore.ScopeTypeUser && s.settings.UserScopeEnabled {
		return memorystore.ScopeTypeUser
	}
	if s.settings.SessionScopeEnabled {
		return memorystore.ScopeTypeSession
	}
	return memorystore.ScopeTypeUser
}

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
		if item.MemoryType != candidate.MemoryType || item.Status != memorystore.MemoryStatusActive {
			continue
		}
		if memorySimilarity(candidate, item) >= 0.97 {
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
		return s.autoSupersedes(candidate, existing), nil
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

func (s *learningService) autoSupersedes(candidate memorystore.MemoryEntry, existing []memorystore.MemoryEntry) []string {
	out := make([]string, 0, 1)
	for _, item := range existing {
		if item.ScopeType != candidate.ScopeType || item.ScopeID != candidate.ScopeID {
			continue
		}
		if item.MemoryType != candidate.MemoryType || item.Status != memorystore.MemoryStatusActive {
			continue
		}
		if memorySimilarity(candidate, item) >= 0.8 {
			out = append(out, item.ID)
			break
		}
	}
	return out
}

func (s *learningService) recordSkipped(ctx context.Context, input LearnFromTurnInput, filtered []TurnMessage, reason string) error {
	_, err := s.store.CreateLearningEvent(ctx, memorystore.LearningEventInput{
		SessionID:    input.SessionID,
		TraceID:      input.TraceID,
		Status:       "skipped",
		InputJSON:    mustMarshalJSON(input),
		FilteredJSON: mustMarshalJSON(filtered),
		ResultJSON:   mustMarshalJSON(map[string]string{"reason": reason}),
	})
	return err
}

func (s *learningService) recordError(ctx context.Context, input LearnFromTurnInput, filtered []TurnMessage, rawJSON string, err error) error {
	_, recordErr := s.store.CreateLearningEvent(ctx, memorystore.LearningEventInput{
		SessionID:      input.SessionID,
		TraceID:        input.TraceID,
		Status:         "error",
		InputJSON:      mustMarshalJSON(input),
		FilteredJSON:   mustMarshalJSON(filtered),
		CandidatesJSON: strings.TrimSpace(rawJSON),
		ErrorText:      err.Error(),
	})
	if recordErr != nil {
		return errors.Join(err, recordErr)
	}
	return err
}

func (s *learningService) recordOutcome(
	ctx context.Context,
	input LearnFromTurnInput,
	filtered []TurnMessage,
	rawJSON string,
	outcome applyOutcome,
) error {
	status := "success"
	if len(outcome.Created) == 0 && len(outcome.Refreshed) == 0 {
		status = "skipped"
	}
	_, err := s.store.CreateLearningEvent(ctx, memorystore.LearningEventInput{
		SessionID:      input.SessionID,
		TraceID:        input.TraceID,
		Status:         status,
		InputJSON:      mustMarshalJSON(input),
		FilteredJSON:   mustMarshalJSON(filtered),
		CandidatesJSON: strings.TrimSpace(rawJSON),
		ResultJSON:     mustMarshalJSON(outcome),
	})
	return err
}

func normalizeLearnInput(input LearnFromTurnInput, settings Settings) LearnFromTurnInput {
	userScope := strings.TrimSpace(input.UserScope)
	if userScope == "" {
		userScope = settings.UserScopeID
	}
	return LearnFromTurnInput{
		SessionID: strings.TrimSpace(input.SessionID),
		UserScope: userScope,
		TraceID:   strings.TrimSpace(input.TraceID),
		Messages:  append([]TurnMessage(nil), input.Messages...),
	}
}

func normalizeSettings(settings Settings) Settings {
	if settings.MaxRecallItems <= 0 {
		settings.MaxRecallItems = 8
	}
	if settings.MinConfidence <= 0 {
		settings.MinConfidence = 0.7
	}
	if strings.TrimSpace(settings.UserScopeID) == "" {
		settings.UserScopeID = memorystore.DefaultUserScopeID
	}
	return settings
}

func mustMarshalJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, err)
	}
	return string(encoded)
}
