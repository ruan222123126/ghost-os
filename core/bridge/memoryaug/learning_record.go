package memoryaug

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/memorystore"
)

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
