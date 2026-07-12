package sessionturn

import (
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

func BuildSessionMetadataPayload(summary SessionMetadataInput) sessionMetadata {
	return sessionMetadata{
		ID:           summary.ID,
		Title:        strings.TrimSpace(summary.Title),
		CreatedAt:    summary.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    summary.UpdatedAt.UTC().Format(time.RFC3339),
		MessageCount: summary.MessageCount,
		TokenCount:   summary.TokenCount,
	}
}

func BuildSessionDetailPayload(
	detail SessionDetailInput,
	includeDraft bool,
) sessionDetail {
	turnDraft := BuildSessionTurnDraftPayload(detail.TurnDraft, includeDraft)
	toolCallNames := buildSessionToolCallNameLookup(detail.Messages)
	messages := make([]sessionMessage, 0, len(detail.Messages))
	for _, item := range detail.Messages {
		messages = append(messages, buildSessionMessagePayloadWithLookup(item.Index, item.Message, toolCallNames))
	}
	if includeDraft && turnDraft == nil {
		if draft, ok := BuildAssistantDraftSessionMessage(detail.AssistantDraft, detail.MessageCount); ok {
			messages = append(messages, draft)
		}
	}

	return sessionDetail{
		ID:           detail.ID,
		Title:        strings.TrimSpace(detail.Title),
		Messages:     messages,
		CreatedAt:    detail.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    detail.UpdatedAt.UTC().Format(time.RFC3339),
		MessageCount: detail.MessageCount,
		Page:         BuildSessionMessagePagePayload(detail.Page),
		TokenCount:   detail.TokenCount,
		TurnDraft:    turnDraft,
	}
}

func BuildSessionMessagePagePayload(page SessionMessagePageInput) sessionMessagePage {
	return sessionMessagePage{
		Limit:         page.Limit,
		Before:        cloneIntPointer(page.Before),
		StartIndex:    cloneIntPointer(page.StartIndex),
		EndIndex:      cloneIntPointer(page.EndIndex),
		HasMoreBefore: page.HasMoreBefore,
		NextBefore:    cloneIntPointer(page.NextBefore),
	}
}

func BuildSessionMessagePayload(index int, message llm.Message) sessionMessage {
	return buildSessionMessagePayloadWithLookup(index, message, nil)
}

func buildSessionMessagePayloadWithLookup(
	index int,
	message llm.Message,
	toolCallNames map[string]string,
) sessionMessage {
	payload := sessionMessage{
		Index: index,
		Role:  string(message.Role),
	}
	if message.Role == llm.RoleTool {
		projectToolSessionMessage(&payload, message, toolCallNames)
	} else if strings.TrimSpace(message.Text) != "" {
		payload.Text = message.Text
	}
	if len(message.Content) > 0 {
		payload.Content = buildSessionContentParts(message.Content)
	}
	if len(message.ToolCalls) > 0 {
		payload.ToolCalls = buildSessionToolCalls(message.ToolCalls)
	}
	if payload.Role == string(llm.RoleAssistant) {
		if thinking := extractReasoningDisplayText(message.ReasoningContent); thinking != "" {
			payload.Thinking = thinking
		}
	}
	if strings.TrimSpace(message.ToolCallID) != "" {
		payload.ToolCallID = message.ToolCallID
	}
	return payload
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
