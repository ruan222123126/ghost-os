package memoryaug

import (
	"context"
	"fmt"
	"strings"

	"ghost-os/bridge/memorystore"
)

type learningService struct {
	settings        Settings
	store           learningStore
	globalExtractor CandidateExtractor
	eventExtractor  EventMemoryExtractor
}

func NewLearningService(
	settings Settings,
	store learningStore,
	globalExtractor CandidateExtractor,
	eventExtractor EventMemoryExtractor,
) LearningService {
	return &learningService{
		settings:        normalizeSettings(settings),
		store:           store,
		globalExtractor: globalExtractor,
		eventExtractor:  eventExtractor,
	}
}

func (s *learningService) LearnFromTurn(ctx context.Context, input LearnFromTurnInput) error {
	if !s.settings.Enabled || !s.settings.LearningEnabled {
		return nil
	}
	if s.store == nil {
		return fmt.Errorf("memory learning store is not configured")
	}
	normalized := normalizeLearnInput(input, s.settings)
	filtered := filterTurnMessages(normalized.Messages)
	if len(filtered) == 0 {
		return s.recordSkipped(ctx, normalized, filtered, "no durable candidates after rule filter")
	}
	if !normalized.AllowWrite {
		return s.recordSkipped(ctx, normalized, filtered, "planner disabled learning for this turn")
	}
	outcome, rawJSON, err := s.learn(ctx, normalized, filtered)
	if err != nil {
		return s.recordError(ctx, normalized, filtered, rawJSON, err)
	}
	return s.recordOutcome(ctx, normalized, filtered, rawJSON, outcome)
}

func (s *learningService) learn(
	ctx context.Context,
	input LearnFromTurnInput,
	filtered []TurnMessage,
) (applyOutcome, string, error) {
	outcome := applyOutcome{}
	rawPayload := map[string]string{}
	if s.settings.UserScopeEnabled && shouldExtractGlobalPreferences(filtered) {
		globalContext, err := s.loadGlobalPreferences(ctx)
		if err != nil {
			return outcome, "", err
		}
		rawPayload["global_preferences"], err = s.applyGlobalPreferences(ctx, input, filtered, globalContext, &outcome)
		if err != nil {
			return outcome, mustMarshalJSON(rawPayload), err
		}
	}
	if !s.settings.SessionScopeEnabled || strings.TrimSpace(input.PrimaryEventID) == "" {
		return outcome, mustMarshalJSON(rawPayload), nil
	}
	eventContext, err := s.loadEventMemories(ctx, input.PrimaryEventID)
	if err != nil {
		return outcome, mustMarshalJSON(rawPayload), err
	}
	rawPayload["event_memories"], err = s.applyEventMemories(ctx, input, filtered, eventContext, &outcome)
	if err != nil {
		return outcome, mustMarshalJSON(rawPayload), err
	}
	return outcome, mustMarshalJSON(rawPayload), nil
}

func (s *learningService) loadGlobalPreferences(ctx context.Context) ([]memorystore.MemoryEntry, error) {
	return s.store.ListGlobalPreferences(ctx, globalPreferenceKeys())
}

func (s *learningService) loadEventMemories(ctx context.Context, eventID string) ([]memorystore.EventMemory, error) {
	items, _, err := s.store.ListEventMemories(ctx, memorystore.EventMemoryListFilter{
		EventID:  eventID,
		Statuses: []string{memorystore.MemoryStatusActive},
		Limit:    16,
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (s *learningService) applyGlobalPreferences(
	ctx context.Context,
	input LearnFromTurnInput,
	filtered []TurnMessage,
	existing []memorystore.MemoryEntry,
	outcome *applyOutcome,
) (string, error) {
	if s.globalExtractor == nil {
		return "", fmt.Errorf("global preference extractor is not configured")
	}
	output, err := s.globalExtractor.Extract(ctx, ExtractInput{
		SessionID:        input.SessionID,
		UserScopeID:      input.UserScope,
		Transcript:       filtered,
		ExistingExplicit: filterMemoryEntriesBySource(existing, memorystore.SourceKindExplicit),
		ExistingLearned:  filterMemoryEntriesBySource(existing, memorystore.SourceKindLearned),
	})
	if err != nil {
		return "", err
	}
	if err := s.applyGlobalCandidates(ctx, input, existing, output.Items, outcome); err != nil {
		return output.RawJSON, err
	}
	return output.RawJSON, nil
}

func (s *learningService) applyEventMemories(
	ctx context.Context,
	input LearnFromTurnInput,
	filtered []TurnMessage,
	existing []memorystore.EventMemory,
	outcome *applyOutcome,
) (string, error) {
	if s.eventExtractor == nil {
		return "", fmt.Errorf("event memory extractor is not configured")
	}
	output, err := s.eventExtractor.Extract(ctx, EventExtractInput{
		SessionID:      input.SessionID,
		PrimaryEventID: input.PrimaryEventID,
		ActiveEventIDs: input.ActiveEventIDs,
		Transcript:     filtered,
		Existing:       existing,
	})
	if err != nil {
		return "", err
	}
	if err := s.applyEventCandidates(ctx, input.PrimaryEventID, existing, output.Items, outcome); err != nil {
		return output.RawJSON, err
	}
	return output.RawJSON, nil
}

func normalizeLearnInput(input LearnFromTurnInput, settings Settings) LearnFromTurnInput {
	userScope := strings.TrimSpace(input.UserScope)
	if userScope == "" {
		userScope = settings.UserScopeID
	}
	return LearnFromTurnInput{
		SessionID:      strings.TrimSpace(input.SessionID),
		UserScope:      userScope,
		TraceID:        strings.TrimSpace(input.TraceID),
		PrimaryEventID: strings.TrimSpace(input.PrimaryEventID),
		ActiveEventIDs: normalizeIDs(input.ActiveEventIDs),
		AllowWrite:     input.AllowWrite,
		Messages:       append([]TurnMessage(nil), input.Messages...),
	}
}

func shouldExtractGlobalPreferences(messages []TurnMessage) bool {
	for _, message := range messages {
		if normalizeRole(message.Role) != "user" {
			continue
		}
		if hasGlobalPreferenceMarker(message.Text) {
			return true
		}
	}
	return false
}

func hasGlobalPreferenceMarker(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return false
	}
	return strings.Contains(lower, "reply in") ||
		strings.Contains(lower, "respond in") ||
		strings.Contains(lower, "请用") ||
		strings.Contains(lower, "默认") ||
		strings.Contains(lower, "response style") ||
		strings.Contains(lower, "approval")
}

func filterMemoryEntriesBySource(items []memorystore.MemoryEntry, sourceKind string) []memorystore.MemoryEntry {
	out := make([]memorystore.MemoryEntry, 0, len(items))
	for _, item := range items {
		if item.SourceKind == sourceKind {
			out = append(out, item)
		}
	}
	return out
}
