package sessionturn

import (
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
	ID             string
	Title          string
	Messages       []IndexedSessionMessageInput
	CreatedAt      time.Time
	UpdatedAt      time.Time
	MessageCount   int
	Page           SessionMessagePageInput
	TokenCount     int
	AssistantDraft *AssistantDraftInput
	TurnDraft      *SessionTurnDraftInput
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
