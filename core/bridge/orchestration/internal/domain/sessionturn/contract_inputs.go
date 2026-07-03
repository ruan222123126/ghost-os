package sessionturn

import (
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

const (
	TurnDraftStatusStreaming     = "streaming"
	TurnDraftStatusAwaitingHuman = "awaiting_human"
	TurnDraftStatusError         = "error"
)

type SessionMetadataInput struct {
	ID           string
	Title        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	MessageCount int
	TokenCount   int
}

type SessionDetailInput struct {
	ID                   string
	Title                string
	Messages             []IndexedSessionMessageInput
	CreatedAt            time.Time
	UpdatedAt            time.Time
	MessageCount         int
	Page                 SessionMessagePageInput
	TokenCount           int
	AssistantDraft       *AssistantDraftInput
	TurnDraft            *SessionTurnDraftInput
	LastRuntimeSelection *SessionRuntimeSelectionInput
}

type SessionRuntimeSelectionInput struct {
	Runtime      string
	Provider     string
	ProviderType string
	Model        string
	Mode         string
}

func buildSessionRuntimeSelectionPayload(selection *SessionRuntimeSelectionInput) *sessionRuntimeSelection {
	if selection == nil {
		return nil
	}
	normalized, ok := session.NormalizeRuntimeSelection(session.RuntimeSelection{
		Runtime:      selection.Runtime,
		Provider:     selection.Provider,
		ProviderType: selection.ProviderType,
		Model:        selection.Model,
		Mode:         selection.Mode,
	})
	if !ok {
		return nil
	}
	return &sessionRuntimeSelection{
		Runtime:      normalized.Runtime,
		Provider:     normalized.Provider,
		ProviderType: normalized.ProviderType,
		Model:        normalized.Model,
		Mode:         normalized.Mode,
	}
}

type IndexedSessionMessageInput struct {
	Index   int
	Message llm.Message
}

type SessionMessagePageInput struct {
	Limit         int
	Before        *int
	StartIndex    *int
	EndIndex      *int
	HasMoreBefore bool
	NextBefore    *int
}

type AssistantDraftInput struct {
	Text string
}

type SessionTurnDraftInput struct {
	TraceID           string
	Turn              int
	Status            string
	Error             string
	PendingQuestions  []TurnDraftPendingQuestionInput
	AssistantSegments []TurnDraftSegmentInput
	ThinkingSegments  []TurnDraftSegmentInput
	Tools             []TurnDraftToolInput
	ItemOrder         []string
}

type TurnDraftSegmentInput struct {
	ID      string
	Content string
}

type TurnDraftToolInput struct {
	ID         string
	Content    string
	ToolInput  string
	ToolName   string
	ToolStatus string
	ToolCallID string
	TraceID    string
}

type TurnDraftPendingQuestionInput struct {
	QuestionID    string
	Prompt        string
	SelectionMode string
	Options       []HumanQuestionOptionInput
}
