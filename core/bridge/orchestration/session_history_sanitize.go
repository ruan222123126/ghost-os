package orchestration

import (
	"strings"

	"ghost-os/bridge/llm"
)

type toolProtocolSanitizer struct {
	pending map[string]int
}

func sanitizeToolProtocolMessages(messages []llm.Message) []llm.Message {
	if len(messages) == 0 {
		return nil
	}

	sanitizer := toolProtocolSanitizer{
		pending: make(map[string]int),
	}
	out := make([]llm.Message, 0, len(messages))
	for _, message := range messages {
		sanitized, keep := sanitizer.sanitizeMessage(message)
		if keep {
			out = append(out, sanitized)
		}
	}
	return llm.CloneMessages(out)
}

func (s toolProtocolSanitizer) sanitizeMessage(message llm.Message) (llm.Message, bool) {
	switch message.Role {
	case llm.RoleAssistant:
		return s.sanitizeAssistantMessage(message)
	case llm.RoleTool:
		return s.sanitizeToolMessage(message)
	default:
		return message, true
	}
}

func (s toolProtocolSanitizer) sanitizeAssistantMessage(message llm.Message) (llm.Message, bool) {
	if len(message.ToolCalls) == 0 {
		return message, true
	}

	validCalls := make([]llm.ToolCall, 0, len(message.ToolCalls))
	for _, call := range message.ToolCalls {
		callID := strings.TrimSpace(call.ID)
		if callID == "" {
			continue
		}
		if _, exists := s.pending[callID]; exists {
			continue
		}
		s.pending[callID] = len(validCalls)
		validCalls = append(validCalls, call)
	}
	if len(validCalls) == 0 {
		message.ToolCalls = nil
		return message, !assistantMessageEmpty(message)
	}
	message.ToolCalls = validCalls
	return message, true
}

func (s toolProtocolSanitizer) sanitizeToolMessage(message llm.Message) (llm.Message, bool) {
	toolCallID := strings.TrimSpace(message.ToolCallID)
	if toolCallID == "" {
		return llm.Message{}, false
	}
	if _, ok := s.pending[toolCallID]; !ok {
		return llm.Message{}, false
	}
	delete(s.pending, toolCallID)
	return message, true
}

func assistantMessageEmpty(message llm.Message) bool {
	return strings.TrimSpace(message.Text) == "" &&
		len(message.Content) == 0 &&
		len(message.ToolCalls) == 0 &&
		len(strings.TrimSpace(string(message.ReasoningContent))) == 0
}
