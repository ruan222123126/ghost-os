package sessionturn

import (
	"encoding/json"
	"strings"

	"ghost-os/bridge/llm"
)

type AnsweredHumanQuestionInput struct {
	QuestionID    string
	Prompt        string
	SelectionMode string
	Options       []HumanQuestionOptionInput
	Answer        string
}

type HumanQuestionOptionInput struct {
	Label       string
	AllowCustom bool
}

func BuildResolvedHumanQuestionToolResult(item AnsweredHumanQuestionInput) (string, string, string) {
	return askHumanResolvedQuestionToolResult(item)
}

func askHumanResolvedQuestionToolResult(item AnsweredHumanQuestionInput) (string, string, string) {
	payload := map[string]any{
		"question_id": item.QuestionID,
		"prompt":      item.Prompt,
		"answer":      item.Answer,
	}
	if selectionMode := strings.TrimSpace(item.SelectionMode); selectionMode != "" {
		payload["selection_mode"] = selectionMode
	}
	if options := askHumanResolvedQuestionOptions(item.Options); len(options) > 0 {
		payload["options"] = options
	}
	return "ask_human", mustEncodeResolvedQuestionPayload(payload), ""
}

func askHumanResolvedQuestionOptions(
	raw []HumanQuestionOptionInput,
) []map[string]any {
	if len(raw) == 0 {
		return nil
	}

	out := make([]map[string]any, 0, len(raw))
	for _, option := range raw {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			continue
		}
		out = append(out, map[string]any{
			"label":        label,
			"allow_custom": option.AllowCustom,
		})
	}
	return out
}

func mustEncodeResolvedQuestionPayload(payload map[string]any) string {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return `{}`
	}
	return string(encoded)
}

type answeredHumanInteractionPayload struct {
	QuestionID    string           `json:"question_id"`
	Prompt        string           `json:"prompt"`
	SelectionMode string           `json:"selection_mode,omitempty"`
	Options       []askHumanOption `json:"options,omitempty"`
	Answer        string           `json:"answer"`
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

func decodeSessionHumanInteraction(result llm.ToolResultEnvelope) *sessionHumanInteraction {
	if strings.TrimSpace(result.Tool) != "ask_human" || strings.TrimSpace(result.Output) == "" {
		return nil
	}

	var payload answeredHumanInteractionPayload
	if err := json.Unmarshal([]byte(result.Output), &payload); err != nil {
		return nil
	}

	questionID := strings.TrimSpace(payload.QuestionID)
	prompt := strings.TrimSpace(payload.Prompt)
	if questionID == "" || prompt == "" {
		return nil
	}
	humanInteraction := &sessionHumanInteraction{
		QuestionID: questionID,
		Prompt:     prompt,
	}
	if selectionMode := strings.TrimSpace(payload.SelectionMode); selectionMode != "" {
		humanInteraction.SelectionMode = selectionMode
	}
	if len(payload.Options) > 0 {
		humanInteraction.Options = cloneSessionHumanInteractionOptions(payload.Options)
	}
	if strings.TrimSpace(payload.Answer) != "" {
		humanInteraction.Answer = payload.Answer
	}
	return humanInteraction
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

func cloneSessionHumanInteractionOptions(options []askHumanOption) []askHumanOption {
	if len(options) == 0 {
		return nil
	}
	cloned := make([]askHumanOption, 0, len(options))
	for _, option := range options {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			continue
		}
		cloned = append(cloned, askHumanOption{
			Label:       label,
			AllowCustom: option.AllowCustom,
		})
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}
