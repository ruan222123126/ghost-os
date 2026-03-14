package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

func codexMessagesToInput(messages []Message) ([]codexInputItem, error) {
	input := make([]codexInputItem, 0, len(messages))
	for _, msg := range messages {
		switch msg.Role {
		case RoleSystem:
			continue
		case RoleUser:
			item, ok, err := toCodexMessageInput(msg)
			if err != nil {
				return nil, err
			}
			if ok {
				input = append(input, item)
			}
		case RoleAssistant:
			if len(msg.ToolCalls) == 0 {
				item, ok, err := toCodexMessageInput(msg)
				if err != nil {
					return nil, err
				}
				if ok {
					input = append(input, item)
				}
			}
			for _, call := range msg.ToolCalls {
				input = append(input, codexInputItem{
					Type:      "function_call",
					CallID:    strings.TrimSpace(call.ID),
					Name:      strings.TrimSpace(call.Name),
					Arguments: string(normalizeJSONObject(call.Arguments)),
				})
			}
		case RoleTool:
			input = append(input, codexInputItem{
				Type:   "function_call_output",
				CallID: strings.TrimSpace(msg.ToolCallID),
				Output: toCodexToolOutput(msg),
			})
		default:
			return nil, fmt.Errorf("unsupported message role for codex provider: %q", msg.Role)
		}
	}
	return input, nil
}

func codexFallbackInput(messages []Message) ([]codexInputItem, error) {
	lastUserIndex := -1
	for index := len(messages) - 1; index >= 0; index-- {
		if messages[index].Role == RoleUser {
			lastUserIndex = index
			break
		}
	}
	if lastUserIndex < 0 {
		return codexMessagesToInput(messages)
	}

	input := make([]codexInputItem, 0, len(messages)-lastUserIndex)
	if userItem, ok, err := codexFallbackUserItem(messages[:lastUserIndex], messages[lastUserIndex]); err != nil {
		return nil, err
	} else if ok {
		input = append(input, userItem)
	}

	for _, msg := range messages[lastUserIndex+1:] {
		switch msg.Role {
		case RoleSystem, RoleUser:
			continue
		case RoleAssistant:
			for _, call := range msg.ToolCalls {
				input = append(input, codexInputItem{
					Type:      "function_call",
					CallID:    strings.TrimSpace(call.ID),
					Name:      strings.TrimSpace(call.Name),
					Arguments: string(normalizeJSONObject(call.Arguments)),
				})
			}
		case RoleTool:
			input = append(input, codexInputItem{
				Type:   "function_call_output",
				CallID: strings.TrimSpace(msg.ToolCallID),
				Output: toCodexToolOutput(msg),
			})
		default:
			return nil, fmt.Errorf("unsupported message role for codex provider: %q", msg.Role)
		}
	}

	return input, nil
}

func codexFallbackUserItem(prior []Message, current Message) (codexInputItem, bool, error) {
	currentText := strings.TrimSpace(codexMessageText(current))
	if currentText == "" {
		item, ok, err := toCodexMessageInput(current)
		if err != nil || !ok {
			return codexInputItem{}, ok, err
		}
		return item, true, nil
	}

	transcript := codexConversationTranscript(prior)
	text := currentText
	if transcript != "" {
		text = "Conversation so far:\n" + transcript + "\n\nCurrent request: " + currentText
	}

	return codexInputItem{
		Type: "message",
		Role: "user",
		Content: []codexInputContent{
			{Type: "input_text", Text: text},
		},
	}, true, nil
}

func codexConversationTranscript(messages []Message) string {
	lines := make([]string, 0, len(messages))
	for _, msg := range messages {
		text := strings.TrimSpace(codexMessageText(msg))
		if text == "" {
			continue
		}
		switch msg.Role {
		case RoleUser:
			lines = append(lines, "User: "+text)
		case RoleAssistant:
			if len(msg.ToolCalls) == 0 {
				lines = append(lines, "Assistant: "+text)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func codexMessageText(msg Message) string {
	parts := make([]string, 0, len(msg.Content)+1)
	if text := strings.TrimSpace(msg.Text); text != "" {
		parts = append(parts, text)
	}
	for _, part := range msg.Content {
		if strings.EqualFold(strings.TrimSpace(part.Type), ContentTypeText) || strings.TrimSpace(part.Type) == "" {
			if text := strings.TrimSpace(part.Text); text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

func codexIncrementalMessages(messages []Message) []Message {
	if len(messages) == 0 {
		return nil
	}

	lastAssistantIndex := -1
	for index, msg := range messages {
		if msg.Role == RoleAssistant {
			lastAssistantIndex = index
		}
	}

	start := lastAssistantIndex + 1
	if lastAssistantIndex < 0 {
		start = 0
	}
	if start >= len(messages) {
		return nil
	}
	return CloneMessages(messages[start:])
}

func toCodexMessageInput(msg Message) (codexInputItem, bool, error) {
	content, err := toCodexInputContent(msg)
	if err != nil {
		return codexInputItem{}, false, err
	}
	if len(content) == 0 {
		return codexInputItem{}, false, nil
	}
	return codexInputItem{
		Type:    "message",
		Role:    string(msg.Role),
		Content: content,
	}, true, nil
}

func toCodexInputContent(msg Message) ([]codexInputContent, error) {
	parts := make([]codexInputContent, 0, len(msg.Content)+1)
	if text := strings.TrimSpace(msg.Text); text != "" {
		parts = append(parts, codexInputContent{
			Type: "input_text",
			Text: text,
		})
	}
	for _, part := range msg.Content {
		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case "", ContentTypeText:
			if text := strings.TrimSpace(part.Text); text != "" {
				parts = append(parts, codexInputContent{
					Type: "input_text",
					Text: text,
				})
			}
		case ContentTypeImage:
			if part.Image == nil {
				continue
			}
			imageURL, err := resolveOpenAIImageURL(part.Image)
			if err != nil {
				return nil, err
			}
			parts = append(parts, codexInputContent{
				Type:     "input_image",
				ImageURL: imageURL,
			})
		}
	}
	return parts, nil
}

func toCodexToolOutput(msg Message) string {
	baseText := strings.TrimSpace(msg.Text)
	if parsed, ok := parseCodexToolEnvelope(baseText); ok {
		switch {
		case parsed.Status == "error" && strings.TrimSpace(parsed.Error) != "":
			baseText = parsed.Error
		case strings.TrimSpace(parsed.Output) != "":
			baseText = parsed.Output
		default:
			baseText = ""
		}
	}

	if len(msg.Content) == 0 {
		return baseText
	}

	type toolContentPart struct {
		Type      string `json:"type"`
		Text      string `json:"text,omitempty"`
		ImageURL  string `json:"image_url,omitempty"`
		Path      string `json:"path,omitempty"`
		MimeType  string `json:"mime_type,omitempty"`
		SHA256    string `json:"sha256,omitempty"`
		ByteCount int    `json:"bytes,omitempty"`
	}

	payload := struct {
		Text    string            `json:"text,omitempty"`
		Content []toolContentPart `json:"content,omitempty"`
	}{
		Text: baseText,
	}
	for _, part := range msg.Content {
		entry := toolContentPart{
			Type: strings.TrimSpace(part.Type),
			Text: strings.TrimSpace(part.Text),
		}
		if part.Image != nil {
			entry.Path = strings.TrimSpace(part.Image.Path)
			entry.ImageURL = strings.TrimSpace(part.Image.URL)
			entry.MimeType = strings.TrimSpace(part.Image.MimeType)
			entry.SHA256 = strings.TrimSpace(part.Image.SHA256)
			entry.ByteCount = part.Image.Bytes
		}
		payload.Content = append(payload.Content, entry)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return baseText
	}
	return string(encoded)
}

func parseCodexToolEnvelope(raw string) (codexToolEnvelope, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return codexToolEnvelope{}, false
	}

	var envelope codexToolEnvelope
	if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
		return codexToolEnvelope{}, false
	}
	if strings.TrimSpace(envelope.Status) == "" {
		return codexToolEnvelope{}, false
	}
	return envelope, true
}
