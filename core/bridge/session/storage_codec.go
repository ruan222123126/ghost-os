package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

func decodeStoredSession(expectedID string, data []byte, now time.Time) (*Session, error) {
	var loaded Session
	if err := json.Unmarshal(data, &loaded); err != nil {
		return nil, fmt.Errorf("%w: id=%s: %v", ErrSessionCorrupted, expectedID, err)
	}

	normalizeLoadedSession(&loaded, expectedID, now)
	if loaded.ID != expectedID {
		return nil, fmt.Errorf("%w: id mismatch file=%q payload=%q", ErrSessionCorrupted, expectedID, loaded.ID)
	}
	return &loaded, nil
}

func encodeStoredSession(session *Session) ([]byte, error) {
	snapshot := cloneSession(session)
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func decodeSessionMetadata(expectedID string, data []byte, now time.Time) (SessionMetadata, error) {
	var envelope sessionMetadataEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return SessionMetadata{}, fmt.Errorf("%w: id=%s: %v", ErrSessionCorrupted, expectedID, err)
	}

	loadedID := strings.TrimSpace(envelope.ID)
	if loadedID == "" {
		loadedID = expectedID
	}
	if loadedID != expectedID {
		return SessionMetadata{}, fmt.Errorf("%w: id mismatch file=%q payload=%q", ErrSessionCorrupted, expectedID, envelope.ID)
	}

	createdAt := envelope.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = now
	}
	updatedAt := envelope.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	return SessionMetadata{
		ID:           expectedID,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		MessageCount: len(envelope.Messages),
		TokenCount:   envelope.TokenCount,
	}, nil
}

func normalizeLoadedSession(session *Session, expectedID string, now time.Time) {
	if session == nil {
		return
	}

	if session.ID == "" {
		session.ID = expectedID
	} else {
		session.ID = strings.TrimSpace(session.ID)
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	if session.UpdatedAt.IsZero() {
		session.UpdatedAt = session.CreatedAt
	}
	session.RecalculateTokenCount()
}

func cloneSession(session *Session) Session {
	if session == nil {
		return Session{}
	}

	cloned := *session
	cloned.Messages = llm.CloneMessages(session.Messages)
	cloned.PendingQuestions = clonePendingQuestions(session.PendingQuestions)
	cloned.HumanAnswers = cloneHumanAnswers(session.HumanAnswers)
	cloned.DynamicToolLoads = cloneDynamicToolLoads(session.DynamicToolLoads)
	cloned.IterationRuntime = cloneIterationRuntime(session.IterationRuntime)
	return cloned
}

func clonePendingQuestions(raw map[string]PendingHumanQuestion) map[string]PendingHumanQuestion {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]PendingHumanQuestion, len(raw))
	for id, question := range raw {
		question.Options = cloneHumanQuestionOptions(question.Options)
		out[id] = question
	}
	return out
}

func cloneHumanAnswers(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]string, len(raw))
	for questionID, answer := range raw {
		out[questionID] = answer
	}
	return out
}

func cloneIterationRuntime(raw *IterationRuntime) *IterationRuntime {
	if raw == nil {
		return nil
	}

	cloned := *raw
	if len(raw.Records) > 0 {
		cloned.Records = append([]IterationRecord(nil), raw.Records...)
	}
	return &cloned
}

func cloneDynamicToolLoads(raw map[string]DynamicToolLoad) map[string]DynamicToolLoad {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]DynamicToolLoad, len(raw))
	for toolName, load := range raw {
		out[toolName] = normalizeDynamicToolLoad(toolName, load)
	}
	return out
}
