package sessionturn

import (
	"encoding/json"
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

func projectToolSessionMessage(
	payload *sessionMessage,
	message llm.Message,
	toolCallNames map[string]string,
) {
	trimmed := strings.TrimSpace(message.Text)
	if payload == nil || trimmed == "" {
		return
	}

	result, ok := llm.ParseToolResultEnvelope(trimmed)
	if !ok {
		if projectLegacyCodexToolResult(payload, message, toolCallNames) {
			return
		}
		payload.Text = message.Text
		return
	}

	humanInteraction := decodeSessionHumanInteraction(result)
	payload.ToolResult = buildSessionToolResultPayload(result, humanInteraction)
	payload.HumanInteraction = humanInteraction
	payload.Text = formatSessionToolText(result, humanInteraction)
}

func buildSessionToolCallNameLookup(messages []IndexedSessionMessageInput) map[string]string {
	if len(messages) == 0 {
		return nil
	}

	lookup := make(map[string]string, len(messages))
	for _, item := range messages {
		for _, call := range item.Message.ToolCalls {
			callID := strings.TrimSpace(call.ID)
			toolName := strings.TrimSpace(call.Name)
			if callID == "" || toolName == "" {
				continue
			}
			lookup[callID] = toolName
		}
	}
	if len(lookup) == 0 {
		return nil
	}
	return lookup
}

// Legacy external Codex sessions stored terminal tool results as plain text.
// They are still terminal tool messages, so project them back into tool_result
// to avoid replaying historical cards as perpetual "running" tools.
func projectLegacyCodexToolResult(
	payload *sessionMessage,
	message llm.Message,
	toolCallNames map[string]string,
) bool {
	if payload == nil || len(toolCallNames) == 0 {
		return false
	}

	toolCallID := strings.TrimSpace(message.ToolCallID)
	if toolCallID == "" {
		return false
	}
	toolName := strings.TrimSpace(toolCallNames[toolCallID])
	if !strings.HasPrefix(toolName, "codex_") {
		return false
	}

	result := llm.ToolResultEnvelope{
		Status: "success",
		Tool:   toolName,
		Output: message.Text,
	}
	payload.ToolResult = buildSessionToolResultPayload(result, nil)
	payload.Text = formatSessionToolText(result, nil)
	return true
}

func buildSessionToolResultPayload(result llm.ToolResultEnvelope, humanInteraction *sessionHumanInteraction) *sessionToolResult {
	payload := &sessionToolResult{
		Status: result.Status,
		Tool:   result.Tool,
	}
	if strings.TrimSpace(result.TraceID) != "" {
		payload.TraceID = result.TraceID
	}
	if strings.TrimSpace(result.Error) != "" {
		payload.Error = result.Error
	}
	if humanInteraction == nil && strings.TrimSpace(result.Output) != "" {
		payload.Output = result.Output
	}
	return payload
}

func formatSessionToolText(result llm.ToolResultEnvelope, humanInteraction *sessionHumanInteraction) string {
	if humanInteraction != nil {
		if strings.TrimSpace(humanInteraction.Answer) != "" {
			return humanInteraction.Prompt + "\n" + humanInteraction.Answer
		}
		return humanInteraction.Prompt
	}
	if strings.TrimSpace(result.Error) != "" {
		return result.Error
	}
	if strings.TrimSpace(result.Output) != "" {
		return result.Output
	}
	if strings.TrimSpace(result.Tool) != "" {
		return "[" + result.Tool + "]"
	}
	return "[tool]"
}

func buildSessionContentParts(parts []llm.ContentPart) []sessionContentPart {
	out := make([]sessionContentPart, 0, len(parts))
	for _, part := range parts {
		mapped := sessionContentPart{Type: part.Type}
		if strings.TrimSpace(part.Text) != "" {
			mapped.Text = part.Text
		}
		if part.Image != nil {
			mapped.Image = &sessionImageContent{
				Path:     part.Image.Path,
				URL:      part.Image.URL,
				MimeType: part.Image.MimeType,
				Width:    part.Image.Width,
				Height:   part.Image.Height,
				SHA256:   part.Image.SHA256,
				Bytes:    part.Image.Bytes,
			}
		}
		out = append(out, mapped)
	}
	return out
}

func buildSessionToolCalls(calls []llm.ToolCall) []sessionToolCall {
	out := make([]sessionToolCall, 0, len(calls))
	for _, call := range calls {
		out = append(out, sessionToolCall{
			ID:        call.ID,
			Name:      call.Name,
			Arguments: decodeSessionToolArguments(call.Arguments),
		})
	}
	return out
}

func decodeSessionToolArguments(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return map[string]any{}
	}
	if decoded == nil {
		return map[string]any{}
	}
	return decoded
}
