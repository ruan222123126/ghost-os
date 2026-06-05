package sessionturn

import (
	"strings"

	"ghost-os/bridge/llm"
	bridgesession "ghost-os/bridge/session"
)

func BuildSessionTurnDraftPayload(
	sess *bridgesession.Session,
	includeDraft bool,
) *sessionTurnDraft {
	if !includeDraft || sess == nil || sess.TurnDraft == nil {
		return nil
	}

	draft := sess.TurnDraft
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

func BuildAssistantDraftSessionMessage(sess *bridgesession.Session) (sessionMessage, bool) {
	if sess == nil || sess.AssistantDraft == nil {
		return sessionMessage{}, false
	}
	if strings.TrimSpace(sess.AssistantDraft.Text) == "" {
		return sessionMessage{}, false
	}
	return sessionMessage{
		Index:      sess.MessageCount,
		Role:       string(llm.RoleAssistant),
		Text:       sess.AssistantDraft.Text,
		InProgress: true,
	}, true
}

func buildSessionTurnDraftSegments(raw []bridgesession.TurnDraftSegment) []sessionTurnDraftSegment {
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

func buildSessionTurnDraftTools(raw []bridgesession.TurnDraftTool) []sessionTurnDraftTool {
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

func normalizeTurnDraftStatus(draft *bridgesession.TurnDraft) string {
	if draft == nil {
		return bridgesession.TurnDraftStatusStreaming
	}

	switch strings.TrimSpace(draft.Status) {
	case bridgesession.TurnDraftStatusStreaming,
		bridgesession.TurnDraftStatusAwaitingHuman,
		bridgesession.TurnDraftStatusError:
		return strings.TrimSpace(draft.Status)
	}
	if strings.TrimSpace(draft.Error) != "" {
		return bridgesession.TurnDraftStatusError
	}
	if len(draft.PendingQuestions) > 0 {
		return bridgesession.TurnDraftStatusAwaitingHuman
	}
	return bridgesession.TurnDraftStatusStreaming
}

func buildSessionTurnDraftPendingQuestions(
	raw []bridgesession.TurnDraftPendingQuestion,
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

func buildTurnDraftQuestionOptions(raw []bridgesession.HumanQuestionOption) []askHumanOption {
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
