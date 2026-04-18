package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
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
	session.TokenCount = session.WindowTokenCount
	session.MessageCount = len(session.Messages)
	session.WindowStart = 0
	session.AssistantDraft = cloneAssistantDraft(session.AssistantDraft)
	session.persistedMessageCount = 0
	session.persistedMessages = nil
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

func cloneDynamicSkillLoads(raw map[string]DynamicSkillLoad) map[string]DynamicSkillLoad {
	if len(raw) == 0 {
		return nil
	}

	out := make(map[string]DynamicSkillLoad, len(raw))
	for skillName, load := range raw {
		out[skillName] = normalizeDynamicSkillLoad(skillName, load)
	}
	return out
}
