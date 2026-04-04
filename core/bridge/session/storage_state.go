package session

import (
	"encoding/json"
	"fmt"
	"time"

	"ghost-os/bridge/guiagent"
	"ghost-os/bridge/llm"
)

const sessionTimeLayout = time.RFC3339Nano

type sessionStoredState struct {
	ConversationState      llm.ConversationState           `json:"conversation_state,omitempty"`
	IterationRuntime       *IterationRuntime               `json:"iteration_runtime,omitempty"`
	PendingQuestions       map[string]PendingHumanQuestion `json:"pending_questions,omitempty"`
	HumanAnswers           map[string]string               `json:"human_answers,omitempty"`
	PendingComputerUseRuns map[string]guiagent.State       `json:"pending_computer_use_runs,omitempty"`
	DynamicToolLoads       map[string]DynamicToolLoad      `json:"dynamic_tool_loads,omitempty"`
}

type sessionRecord struct {
	ID               string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	EndedAt          time.Time
	TurnIndex        int
	TokenCount       int
	MessageCount     int
	WindowStart      int
	WindowTokenCount int
	State            sessionStoredState
}

func encodeSessionState(sess *Session) (string, error) {
	state := sessionStoredState{
		ConversationState:      sess.ConversationState,
		IterationRuntime:       cloneIterationRuntime(sess.IterationRuntime),
		PendingQuestions:       clonePendingQuestions(sess.PendingQuestions),
		HumanAnswers:           cloneHumanAnswers(sess.HumanAnswers),
		PendingComputerUseRuns: clonePendingComputerUseRuns(sess.PendingComputerUseRuns),
		DynamicToolLoads:       cloneDynamicToolLoads(sess.DynamicToolLoads),
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		return "", fmt.Errorf("encode session state: %w", err)
	}
	return string(encoded), nil
}

func decodeSessionState(raw string) (sessionStoredState, error) {
	if raw == "" {
		return sessionStoredState{}, nil
	}

	var state sessionStoredState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return sessionStoredState{}, fmt.Errorf("%w: session state: %v", ErrSessionCorrupted, err)
	}
	return state, nil
}

func sessionFromRecord(record sessionRecord, messages []llm.Message) *Session {
	sess := &Session{
		ID:                     record.ID,
		Messages:               llm.CloneMessages(messages),
		CreatedAt:              record.CreatedAt,
		UpdatedAt:              record.UpdatedAt,
		EndedAt:                record.EndedAt,
		TurnIndex:              record.TurnIndex,
		TokenCount:             record.TokenCount,
		MessageCount:           record.MessageCount,
		WindowStart:            record.WindowStart,
		WindowTokenCount:       record.WindowTokenCount,
		ConversationState:      record.State.ConversationState,
		IterationRuntime:       cloneIterationRuntime(record.State.IterationRuntime),
		PendingQuestions:       clonePendingQuestions(record.State.PendingQuestions),
		HumanAnswers:           cloneHumanAnswers(record.State.HumanAnswers),
		PendingComputerUseRuns: clonePendingComputerUseRuns(record.State.PendingComputerUseRuns),
		DynamicToolLoads:       cloneDynamicToolLoads(record.State.DynamicToolLoads),
	}
	sess.setPersistedSnapshot()
	return sess
}

func formatSessionTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(sessionTimeLayout)
}

func parseSessionTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, nil
	}

	parsed, err := time.Parse(sessionTimeLayout, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: invalid session timestamp %q", ErrSessionCorrupted, raw)
	}
	return parsed.UTC(), nil
}
