package orchestration

import (
	"encoding/json"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/artifacts"
	"ghost-os/bridge/llm"
	bridgesession "ghost-os/bridge/session"
)

func buildSessionMetadataPayload(summary bridgesession.SessionMetadata) sessionMetadata {
	return sessionMetadata{
		ID:           summary.ID,
		CreatedAt:    summary.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    summary.UpdatedAt.UTC().Format(time.RFC3339),
		MessageCount: summary.MessageCount,
		TokenCount:   summary.TokenCount,
	}
}

func buildSessionDetailPayload(sess *bridgesession.Session) sessionDetail {
	messages := make([]sessionMessage, 0, len(sess.Messages))
	for _, message := range sess.Messages {
		messages = append(messages, buildSessionMessagePayload(message))
	}

	return sessionDetail{
		ID:         sess.ID,
		Messages:   messages,
		CreatedAt:  sess.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  sess.UpdatedAt.UTC().Format(time.RFC3339),
		TokenCount: sess.TokenCount,
	}
}

func buildSessionMessagePayload(message llm.Message) sessionMessage {
	payload := sessionMessage{
		Role: string(message.Role),
	}
	if message.Role == llm.RoleTool {
		projectToolSessionMessage(&payload, message.Text)
	} else if strings.TrimSpace(message.Text) != "" {
		payload.Text = message.Text
	}
	if len(message.Content) > 0 {
		payload.Content = buildSessionContentParts(message.Content)
	}
	if len(message.ToolCalls) > 0 {
		payload.ToolCalls = buildSessionToolCalls(message.ToolCalls)
	}
	if strings.TrimSpace(message.ToolCallID) != "" {
		payload.ToolCallID = message.ToolCallID
	}
	return payload
}

type answeredHumanInteractionPayload struct {
	QuestionID    string           `json:"question_id"`
	Prompt        string           `json:"prompt"`
	SelectionMode string           `json:"selection_mode,omitempty"`
	Options       []askHumanOption `json:"options,omitempty"`
	Answer        string           `json:"answer"`
}

func projectToolSessionMessage(payload *sessionMessage, rawText string) {
	trimmed := strings.TrimSpace(rawText)
	if payload == nil || trimmed == "" {
		return
	}

	result, ok := agent.ParseToolResultEnvelope(trimmed)
	if !ok {
		payload.Text = rawText
		return
	}

	if sendFileResult, ok := decodeSessionFileResult(result); ok {
		payload.Content = []sessionContentPart{{
			Type: "file",
			File: &sessionFileContent{
				ArtifactID:  sendFileResult.Artifact.ArtifactID,
				Name:        sendFileResult.Artifact.Name,
				MimeType:    sendFileResult.Artifact.MimeType,
				Bytes:       int(sendFileResult.Artifact.Bytes),
				SHA256:      sendFileResult.Artifact.SHA256,
				DownloadURL: sendFileResult.Artifact.DownloadURL,
				SourcePath:  sendFileResult.Artifact.SourcePath,
				Note:        sendFileResult.Artifact.Note,
			},
		}}
		result.Output = strings.TrimSpace(sendFileResult.Message)
	}

	humanInteraction := decodeSessionHumanInteraction(result)
	payload.ToolResult = buildSessionToolResultPayload(result, humanInteraction)
	payload.HumanInteraction = humanInteraction
	payload.Text = formatSessionToolText(result, humanInteraction)
}

func buildSessionToolResultPayload(result agent.ToolResultEnvelope, humanInteraction *sessionHumanInteraction) *sessionToolResult {
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

func decodeSessionHumanInteraction(result agent.ToolResultEnvelope) *sessionHumanInteraction {
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

func formatSessionToolText(result agent.ToolResultEnvelope, humanInteraction *sessionHumanInteraction) string {
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

func decodeSessionFileResult(result agent.ToolResultEnvelope) (artifacts.SendFileResult, bool) {
	if strings.TrimSpace(result.Tool) != "send_file" || strings.TrimSpace(result.Output) == "" {
		return artifacts.SendFileResult{}, false
	}
	return artifacts.DecodeSendFileResult(result.Output)
}
