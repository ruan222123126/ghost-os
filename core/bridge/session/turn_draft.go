package session

import (
	"strings"
	"time"
)

type TurnDraftSegment struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

type TurnDraftTool struct {
	ID         string `json:"id"`
	Content    string `json:"content"`
	ToolInput  string `json:"tool_input,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
	ToolStatus string `json:"tool_status,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
}

const (
	TurnDraftStatusStreaming     = "streaming"
	TurnDraftStatusAwaitingHuman = "awaiting_human"
	TurnDraftStatusError         = "error"
)

type TurnDraftPendingQuestion struct {
	QuestionID    string                `json:"question_id"`
	Prompt        string                `json:"prompt"`
	SelectionMode string                `json:"selection_mode,omitempty"`
	Options       []HumanQuestionOption `json:"options,omitempty"`
}

type TurnDraft struct {
	TraceID           string                     `json:"trace_id"`
	Turn              int                        `json:"turn"`
	Status            string                     `json:"status"`
	Error             string                     `json:"error,omitempty"`
	PendingQuestions  []TurnDraftPendingQuestion `json:"pending_questions,omitempty"`
	AssistantSegments []TurnDraftSegment         `json:"assistant_segments,omitempty"`
	ThinkingSegments  []TurnDraftSegment         `json:"thinking_segments,omitempty"`
	Tools             []TurnDraftTool            `json:"tools,omitempty"`
	ItemOrder         []string                   `json:"item_order,omitempty"`
	ToolTagState      *TurnDraftToolTagState     `json:"tool_tag_state,omitempty"`
}

type TurnDraftToolTagState struct {
	Mode            string `json:"mode,omitempty"`
	NormalCandidate string `json:"normal_candidate,omitempty"`
	RawTagPrefix    string `json:"raw_tag_prefix,omitempty"`
	CurrentToolID   string `json:"current_tool_id,omitempty"`
	CloseCandidate  string `json:"close_candidate,omitempty"`
	ArgsBuffer      string `json:"args_buffer,omitempty"`
	InString        bool   `json:"in_string,omitempty"`
	Escaped         bool   `json:"escaped,omitempty"`
	CurrentCallSeq  int    `json:"current_call_seq,omitempty"`
	NextCallSeq     int    `json:"next_call_seq,omitempty"`
}

func (s *Session) ClearTurnDraft(at time.Time) bool {
	if s == nil {
		debugNilReceiver("ClearTurnDraft")
		return false
	}
	if s.TurnDraft == nil {
		return false
	}
	s.TurnDraft = nil
	s.UpdatedAt = normalizeDraftUpdatedAt(at)
	return true
}

func cloneTurnDraft(raw *TurnDraft) *TurnDraft {
	if raw == nil {
		return nil
	}

	return &TurnDraft{
		TraceID:           strings.TrimSpace(raw.TraceID),
		Turn:              raw.Turn,
		Status:            strings.TrimSpace(raw.Status),
		Error:             strings.TrimSpace(raw.Error),
		PendingQuestions:  cloneTurnDraftPendingQuestions(raw.PendingQuestions),
		AssistantSegments: cloneTurnDraftSegments(raw.AssistantSegments),
		ThinkingSegments:  cloneTurnDraftSegments(raw.ThinkingSegments),
		Tools:             cloneTurnDraftTools(raw.Tools),
		ItemOrder:         append([]string(nil), raw.ItemOrder...),
		ToolTagState:      cloneTurnDraftToolTagState(raw.ToolTagState),
	}
}

func cloneTurnDraftSegments(raw []TurnDraftSegment) []TurnDraftSegment {
	if len(raw) == 0 {
		return nil
	}

	out := make([]TurnDraftSegment, 0, len(raw))
	for _, item := range raw {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		out = append(out, TurnDraftSegment{ID: id, Content: item.Content})
	}
	return out
}

func cloneTurnDraftTools(raw []TurnDraftTool) []TurnDraftTool {
	if len(raw) == 0 {
		return nil
	}

	out := make([]TurnDraftTool, 0, len(raw))
	for _, item := range raw {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		out = append(out, TurnDraftTool{
			ID:         id,
			Content:    item.Content,
			ToolInput:  item.ToolInput,
			ToolName:   strings.TrimSpace(item.ToolName),
			ToolStatus: strings.TrimSpace(item.ToolStatus),
			ToolCallID: strings.TrimSpace(item.ToolCallID),
			TraceID:    strings.TrimSpace(item.TraceID),
		})
	}
	return out
}

func cloneTurnDraftToolTagState(raw *TurnDraftToolTagState) *TurnDraftToolTagState {
	if raw == nil {
		return nil
	}

	cloned := *raw
	cloned.Mode = strings.TrimSpace(raw.Mode)
	cloned.CurrentToolID = strings.TrimSpace(raw.CurrentToolID)
	return &cloned
}

func cloneTurnDraftPendingQuestions(raw []TurnDraftPendingQuestion) []TurnDraftPendingQuestion {
	if len(raw) == 0 {
		return nil
	}

	out := make([]TurnDraftPendingQuestion, 0, len(raw))
	for _, item := range raw {
		questionID := strings.TrimSpace(item.QuestionID)
		prompt := strings.TrimSpace(item.Prompt)
		if questionID == "" || prompt == "" {
			continue
		}
		out = append(out, TurnDraftPendingQuestion{
			QuestionID:    questionID,
			Prompt:        prompt,
			SelectionMode: strings.TrimSpace(item.SelectionMode),
			Options:       cloneHumanQuestionOptions(item.Options),
		})
	}
	return out
}
