package llm

import (
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
			input = appendCodexFunctionCallItems(input, msg.ToolCalls)
		case RoleTool:
			input = appendCodexFunctionCallOutputItem(input, msg)
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
			input = appendCodexFunctionCallItems(input, msg.ToolCalls)
		case RoleTool:
			input = appendCodexFunctionCallOutputItem(input, msg)
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
		if msg.Role != RoleAssistant {
			continue
		}
		if codexSkipsIncrementalBoundary(msg) {
			continue
		}
		lastAssistantIndex = index
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

func codexSkipsIncrementalBoundary(msg Message) bool {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return false
	}
	return strings.HasPrefix(text, "[TOOL_TAG_RESULT]")
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

func appendCodexFunctionCallItems(input []codexInputItem, calls []ToolCall) []codexInputItem {
	for _, call := range calls {
		input = append(input, codexInputItem{
			Type:      "function_call",
			CallID:    strings.TrimSpace(call.ID),
			Name:      strings.TrimSpace(call.Name),
			Arguments: string(normalizeJSONObject(call.Arguments)),
		})
	}
	return input
}

func appendCodexFunctionCallOutputItem(input []codexInputItem, msg Message) []codexInputItem {
	return append(input, codexInputItem{
		Type:   "function_call_output",
		CallID: strings.TrimSpace(msg.ToolCallID),
		Output: toCodexToolOutput(msg),
	})
}

func toCodexInputContent(msg Message) ([]codexInputContent, error) {
	parts := make([]codexInputContent, 0, len(msg.Content)+1)
	textContentType := codexTextContentType(msg.Role)
	if text := strings.TrimSpace(msg.Text); text != "" {
		parts = append(parts, codexInputContent{
			Type: textContentType,
			Text: text,
		})
	}
	for _, part := range msg.Content {
		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case "", ContentTypeText:
			if text := strings.TrimSpace(part.Text); text != "" {
				parts = append(parts, codexInputContent{
					Type: textContentType,
					Text: text,
				})
			}
		case ContentTypeImage:
			if msg.Role != RoleUser {
				return nil, fmt.Errorf("unsupported image content role for codex provider: %q", msg.Role)
			}
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

func codexTextContentType(role Role) string {
	if role == RoleAssistant {
		return "output_text"
	}
	return "input_text"
}
