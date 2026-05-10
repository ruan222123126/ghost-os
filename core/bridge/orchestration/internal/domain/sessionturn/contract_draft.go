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
		TraceID:           strings.TrimSpace(draft.TraceID),
		Turn:              draft.Turn,
		AssistantSegments: buildSessionTurnDraftSegments(draft.AssistantSegments),
		ThinkingSegments:  buildSessionTurnDraftSegments(draft.ThinkingSegments),
		Tools:             buildSessionTurnDraftTools(draft.Tools),
		ItemOrder:         append([]string(nil), draft.ItemOrder...),
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
