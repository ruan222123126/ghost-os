package agent

import (
	"strings"

	"ghost-os/bridge/llm"
)

type History struct {
	messages []llm.ChatMessage
}

func NewHistory(systemPrompt string) *History {
	h := &History{
		messages: make([]llm.ChatMessage, 0, 16),
	}

	if prompt := strings.TrimSpace(systemPrompt); prompt != "" {
		h.Append(llm.ChatMessage{
			Role:    "system",
			Content: prompt,
		})
	}

	return h
}

func (h *History) Append(msg llm.ChatMessage) {
	h.messages = append(h.messages, msg)
}

func (h *History) Messages() []llm.ChatMessage {
	out := make([]llm.ChatMessage, len(h.messages))
	copy(out, h.messages)
	return out
}

func (h *History) Len() int {
	return len(h.messages)
}
