package sessions

import (
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
	"ghost-os/bridge/session"
)

func BuildMetadataInput(summary session.SessionMetadata) sessionturn.SessionMetadataInput {
	return sessionturn.SessionMetadataInput{
		ID:           summary.ID,
		Title:        summary.Title,
		CreatedAt:    summary.CreatedAt,
		UpdatedAt:    summary.UpdatedAt,
		MessageCount: summary.MessageCount,
		TokenCount:   summary.TokenCount,
	}
}

func BuildDetailInput(sess *session.Session, page session.MessagePage) sessionturn.SessionDetailInput {
	return sessionturn.SessionDetailInput{
		ID:                   sess.ID,
		Title:                sess.Title,
		Messages:             buildIndexedMessages(page.Messages),
		CreatedAt:            sess.CreatedAt,
		UpdatedAt:            sess.UpdatedAt,
		MessageCount:         sess.MessageCount,
		Page:                 BuildMessagePageInput(page),
		TokenCount:           sess.TokenCount,
		AssistantDraft:       buildAssistantDraftInput(sess.AssistantDraft),
		TurnDraft:            BuildTurnDraftInput(sess.TurnDraft),
		LastRuntimeSelection: buildRuntimeSelectionInput(sess.LastRuntimeSelection),
	}
}

func BuildMessagePageInput(page session.MessagePage) sessionturn.SessionMessagePageInput {
	return sessionturn.SessionMessagePageInput{
		Limit:         page.Limit,
		Before:        cloneInt(page.Before),
		StartIndex:    cloneInt(page.StartIndex),
		EndIndex:      cloneInt(page.EndIndex),
		HasMoreBefore: page.HasMoreBefore,
		NextBefore:    cloneInt(page.NextBefore),
	}
}

func BuildTurnDraftInput(draft *session.TurnDraft) *sessionturn.SessionTurnDraftInput {
	if draft == nil {
		return nil
	}
	return &sessionturn.SessionTurnDraftInput{
		TraceID:           draft.TraceID,
		Turn:              draft.Turn,
		Status:            draft.Status,
		Error:             draft.Error,
		PendingQuestions:  buildTurnDraftPendingQuestions(draft.PendingQuestions),
		AssistantSegments: buildTurnDraftSegments(draft.AssistantSegments),
		ThinkingSegments:  buildTurnDraftSegments(draft.ThinkingSegments),
		Tools:             buildTurnDraftTools(draft.Tools),
		ItemOrder:         append([]string(nil), draft.ItemOrder...),
	}
}

func buildIndexedMessages(raw []session.IndexedMessage) []sessionturn.IndexedSessionMessageInput {
	if len(raw) == 0 {
		return nil
	}
	out := make([]sessionturn.IndexedSessionMessageInput, 0, len(raw))
	for _, item := range raw {
		out = append(out, sessionturn.IndexedSessionMessageInput{
			Index:   item.Index,
			Message: item.Message,
		})
	}
	return out
}

func buildAssistantDraftInput(draft *session.AssistantDraft) *sessionturn.AssistantDraftInput {
	if draft == nil {
		return nil
	}
	return &sessionturn.AssistantDraftInput{Text: draft.Text}
}

func buildRuntimeSelectionInput(selection *session.RuntimeSelection) *sessionturn.SessionRuntimeSelectionInput {
	normalized := session.CloneRuntimeSelection(selection)
	if normalized == nil {
		return nil
	}
	return &sessionturn.SessionRuntimeSelectionInput{
		Runtime:      normalized.Runtime,
		Provider:     normalized.Provider,
		ProviderType: normalized.ProviderType,
		Model:        normalized.Model,
		Mode:         normalized.Mode,
	}
}

func buildTurnDraftSegments(raw []session.TurnDraftSegment) []sessionturn.TurnDraftSegmentInput {
	if len(raw) == 0 {
		return nil
	}
	out := make([]sessionturn.TurnDraftSegmentInput, 0, len(raw))
	for _, item := range raw {
		out = append(out, sessionturn.TurnDraftSegmentInput{
			ID:      item.ID,
			Content: item.Content,
		})
	}
	return out
}

func buildTurnDraftTools(raw []session.TurnDraftTool) []sessionturn.TurnDraftToolInput {
	if len(raw) == 0 {
		return nil
	}
	out := make([]sessionturn.TurnDraftToolInput, 0, len(raw))
	for _, item := range raw {
		out = append(out, sessionturn.TurnDraftToolInput{
			ID:         item.ID,
			Content:    item.Content,
			ToolInput:  item.ToolInput,
			ToolName:   item.ToolName,
			ToolStatus: item.ToolStatus,
			ToolCallID: item.ToolCallID,
			TraceID:    item.TraceID,
		})
	}
	return out
}

func buildTurnDraftPendingQuestions(
	raw []session.TurnDraftPendingQuestion,
) []sessionturn.TurnDraftPendingQuestionInput {
	if len(raw) == 0 {
		return nil
	}
	out := make([]sessionturn.TurnDraftPendingQuestionInput, 0, len(raw))
	for _, item := range raw {
		out = append(out, sessionturn.TurnDraftPendingQuestionInput{
			QuestionID:    item.QuestionID,
			Prompt:        item.Prompt,
			SelectionMode: item.SelectionMode,
			Options:       buildHumanQuestionOptions(item.Options),
		})
	}
	return out
}

func buildHumanQuestionOptions(raw []session.HumanQuestionOption) []sessionturn.HumanQuestionOptionInput {
	if len(raw) == 0 {
		return nil
	}
	out := make([]sessionturn.HumanQuestionOptionInput, 0, len(raw))
	for _, item := range raw {
		out = append(out, sessionturn.HumanQuestionOptionInput{
			Label:       item.Label,
			AllowCustom: item.AllowCustom,
		})
	}
	return out
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
