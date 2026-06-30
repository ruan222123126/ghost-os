package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

const sessionTimeLayout = time.RFC3339Nano

type sessionStoredState struct {
	Title                string                          `json:"title,omitempty"`
	ConversationState    llm.ConversationState           `json:"conversation_state,omitempty"`
	RelayRuntime         *RelayRuntime                   `json:"relay_runtime,omitempty"`
	ExternalRuntime      *ExternalRuntime                `json:"external_runtime,omitempty"`
	PendingQuestions     map[string]PendingHumanQuestion `json:"pending_questions,omitempty"`
	HumanAnswers         map[string]string               `json:"human_answers,omitempty"`
	DynamicToolLoads     map[string]DynamicToolLoad      `json:"dynamic_tool_loads,omitempty"`
	DynamicSkillLoads    map[string]DynamicSkillLoad     `json:"dynamic_skill_loads,omitempty"`
	AssistantDraft       *AssistantDraft                 `json:"assistant_draft,omitempty"`
	TurnDraft            *TurnDraft                      `json:"turn_draft,omitempty"`
	LastRuntimeSelection *RuntimeSelection               `json:"last_runtime_selection,omitempty"`
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
		Title:                strings.TrimSpace(sess.Title),
		ConversationState:    sess.ConversationState,
		RelayRuntime:         cloneRelayRuntime(sess.RelayRuntime),
		ExternalRuntime:      cloneExternalRuntime(sess.ExternalRuntime),
		PendingQuestions:     clonePendingQuestions(sess.PendingQuestions),
		HumanAnswers:         cloneHumanAnswers(sess.HumanAnswers),
		DynamicToolLoads:     cloneDynamicToolLoads(sess.DynamicToolLoads),
		DynamicSkillLoads:    cloneDynamicSkillLoads(sess.DynamicSkillLoads),
		AssistantDraft:       cloneAssistantDraft(sess.AssistantDraft),
		TurnDraft:            cloneTurnDraft(sess.TurnDraft),
		LastRuntimeSelection: CloneRuntimeSelection(sess.LastRuntimeSelection),
	}
	return encodeSessionRecordState(state)
}

func encodeSessionRecordState(state sessionStoredState) (string, error) {
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
		ID:                   record.ID,
		Title:                strings.TrimSpace(record.State.Title),
		Messages:             llm.CloneMessages(messages),
		CreatedAt:            record.CreatedAt,
		UpdatedAt:            record.UpdatedAt,
		EndedAt:              record.EndedAt,
		TurnIndex:            record.TurnIndex,
		TokenCount:           record.TokenCount,
		MessageCount:         record.MessageCount,
		WindowStart:          record.WindowStart,
		WindowTokenCount:     record.WindowTokenCount,
		ConversationState:    record.State.ConversationState,
		RelayRuntime:         cloneRelayRuntime(record.State.RelayRuntime),
		ExternalRuntime:      cloneExternalRuntime(record.State.ExternalRuntime),
		PendingQuestions:     clonePendingQuestions(record.State.PendingQuestions),
		HumanAnswers:         cloneHumanAnswers(record.State.HumanAnswers),
		DynamicToolLoads:     cloneDynamicToolLoads(record.State.DynamicToolLoads),
		DynamicSkillLoads:    cloneDynamicSkillLoads(record.State.DynamicSkillLoads),
		AssistantDraft:       cloneAssistantDraft(record.State.AssistantDraft),
		TurnDraft:            cloneTurnDraft(record.State.TurnDraft),
		LastRuntimeSelection: CloneRuntimeSelection(record.State.LastRuntimeSelection),
	}
	sess.setPersistedSnapshot()
	return sess
}

func mergeExistingTitle(sess *Session, existing *sessionRecord) {
	if sess == nil || existing == nil {
		return
	}
	if strings.TrimSpace(sess.Title) != "" {
		sess.Title = strings.TrimSpace(sess.Title)
		return
	}
	sess.Title = strings.TrimSpace(existing.State.Title)
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
