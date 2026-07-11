package sessionturn

import (
	"strings"

	"ghost-os/bridge/llm"
)

func BuildSessionTurnDraftPayload(
	draft *SessionTurnDraftInput,
	includeDraft bool,
) *sessionTurnDraft {
	if !includeDraft || draft == nil {
		return nil
	}

	if strings.TrimSpace(draft.TraceID) == "" {
		return nil
	}
	return &sessionTurnDraft{
		Status:            normalizeTurnDraftStatus(draft),
		Error:             strings.TrimSpace(draft.Error),
		PendingQuestions:  buildSessionTurnDraftPendingQuestions(draft.PendingQuestions),
		TraceID:           strings.TrimSpace(draft.TraceID),
		Turn:              draft.Turn,
		AssistantSegments: buildSessionTurnDraftSegments(draft.AssistantSegments),
		ThinkingSegments:  buildSessionTurnDraftSegments(draft.ThinkingSegments),
		Tools:             buildSessionTurnDraftTools(draft.Tools),
		ItemOrder:         buildSessionTurnDraftItemOrder(draft.ItemOrder),
	}
}

func BuildAssistantDraftSessionMessage(draft *AssistantDraftInput, messageCount int) (sessionMessage, bool) {
	if draft == nil {
		return sessionMessage{}, false
	}
	if strings.TrimSpace(draft.Text) == "" {
		return sessionMessage{}, false
	}
	return sessionMessage{
		Index:      messageCount,
		Role:       string(llm.RoleAssistant),
		Text:       draft.Text,
		InProgress: true,
	}, true
}

func buildSessionTurnDraftSegments(raw []TurnDraftSegmentInput) []sessionTurnDraftSegment {
	if len(raw) == 0 {
		return []sessionTurnDraftSegment{}
	}

	out := make([]sessionTurnDraftSegment, 0, len(raw))
	for _, item := range raw {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		out = append(out, sessionTurnDraftSegment{ID: id, Content: item.Content})
	}
	return out
}

func buildSessionTurnDraftTools(raw []TurnDraftToolInput) []sessionTurnDraftTool {
	if len(raw) == 0 {
		return []sessionTurnDraftTool{}
	}

	out := make([]sessionTurnDraftTool, 0, len(raw))
	for _, item := range raw {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		out = append(out, sessionTurnDraftTool{
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

func buildSessionTurnDraftItemOrder(raw []string) []string {
	if len(raw) == 0 {
		return []string{}
	}

	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}

func normalizeTurnDraftStatus(draft *SessionTurnDraftInput) string {
	if draft == nil {
		return TurnDraftStatusStreaming
	}

	switch strings.TrimSpace(draft.Status) {
	case TurnDraftStatusStreaming,
		TurnDraftStatusAwaitingHuman,
		TurnDraftStatusError:
		return strings.TrimSpace(draft.Status)
	}
	if strings.TrimSpace(draft.Error) != "" {
		return TurnDraftStatusError
	}
	if len(draft.PendingQuestions) > 0 {
		return TurnDraftStatusAwaitingHuman
	}
	return TurnDraftStatusStreaming
}

func buildSessionTurnDraftPendingQuestions(
	raw []TurnDraftPendingQuestionInput,
) []sessionTurnDraftPendingQuestion {
	if len(raw) == 0 {
		return []sessionTurnDraftPendingQuestion{}
	}

	out := make([]sessionTurnDraftPendingQuestion, 0, len(raw))
	for _, item := range raw {
		questionID := strings.TrimSpace(item.QuestionID)
		prompt := strings.TrimSpace(item.Prompt)
		if questionID == "" || prompt == "" {
			continue
		}
		out = append(out, sessionTurnDraftPendingQuestion{
			QuestionID:    questionID,
			Prompt:        prompt,
			SelectionMode: strings.TrimSpace(item.SelectionMode),
			Options:       buildTurnDraftQuestionOptions(item.Options),
		})
	}
	return out
}

func buildTurnDraftQuestionOptions(raw []HumanQuestionOptionInput) []askHumanOption {
	if len(raw) == 0 {
		return nil
	}

	out := make([]askHumanOption, 0, len(raw))
	for _, item := range raw {
		label := strings.TrimSpace(item.Label)
		if label == "" {
			continue
		}
		out = append(out, askHumanOption{
			Label:       label,
			AllowCustom: item.AllowCustom,
		})
	}
	return out
}
