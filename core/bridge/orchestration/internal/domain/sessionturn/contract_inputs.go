package sessionturn

import (
	"strings"
	"time"

	"ghost-os/bridge/llm"
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
	normalized, ok := normalizeSessionRuntimeSelectionInput(*selection)
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

func normalizeSessionRuntimeSelectionInput(
	input SessionRuntimeSelectionInput,
) (SessionRuntimeSelectionInput, bool) {
	selection := SessionRuntimeSelectionInput{
		Runtime:      strings.ToLower(strings.TrimSpace(input.Runtime)),
		Provider:     strings.TrimSpace(input.Provider),
		ProviderType: strings.ToLower(strings.TrimSpace(input.ProviderType)),
		Model:        strings.TrimSpace(input.Model),
		Mode:         strings.ToLower(strings.TrimSpace(input.Mode)),
	}
	if selection.Runtime == "" {
		return SessionRuntimeSelectionInput{}, false
	}
	if selection.Mode == "" {
		selection.Mode = "default"
	}
	if selection.Provider == "" {
		selection.Provider = selection.ProviderType
	}
	if selection.ProviderType == "" && selection.Runtime == "codex" {
		selection.ProviderType = "codex"
	}
	return selection, true
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
