package session

import (
	"strings"
	"time"
)

// AssistantDraft 保存 assistant 流式中间草稿，避免中断时整段丢失。
type AssistantDraft struct {
	Text      string    `json:"text"`
	TraceID   string    `json:"trace_id,omitempty"`
	Turn      int       `json:"turn,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Session) AppendAssistantDraft(delta string, traceID string, turn int, at time.Time) bool {
	if s == nil {
		debugNilReceiver("AppendAssistantDraft")
		return false
	}
	if delta == "" {
		return false
	}

	updatedAt := normalizeDraftUpdatedAt(at)
	normalizedTraceID := strings.TrimSpace(traceID)
	if draftBelongsToAnotherTurn(s.AssistantDraft, normalizedTraceID, turn) {
		s.AssistantDraft = &AssistantDraft{
			Text:      delta,
			TraceID:   normalizedTraceID,
			Turn:      turn,
			UpdatedAt: updatedAt,
		}
		s.UpdatedAt = updatedAt
		return true
	}

	s.AssistantDraft.Text += delta
	s.AssistantDraft.UpdatedAt = updatedAt
	s.UpdatedAt = updatedAt
	return true
}

func (s *Session) ClearAssistantDraft(at time.Time) bool {
	if s == nil {
		debugNilReceiver("ClearAssistantDraft")
		return false
	}
	if s.AssistantDraft == nil {
		return false
	}
	s.AssistantDraft = nil
	s.UpdatedAt = normalizeDraftUpdatedAt(at)
	return true
}

func cloneAssistantDraft(raw *AssistantDraft) *AssistantDraft {
	if raw == nil {
		return nil
	}

	return &AssistantDraft{
		Text:      raw.Text,
		TraceID:   strings.TrimSpace(raw.TraceID),
		Turn:      raw.Turn,
		UpdatedAt: raw.UpdatedAt.UTC(),
	}
}

func normalizeDraftUpdatedAt(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func draftBelongsToAnotherTurn(draft *AssistantDraft, traceID string, turn int) bool {
	if draft == nil {
		return true
	}
	return strings.TrimSpace(draft.TraceID) != traceID || draft.Turn != turn
}
