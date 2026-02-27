package agent

import (
	"strings"

	"ghost-os/bridge/llm"
)

// History 仅负责维护会话消息序列，不包含业务决策。
type History struct {
	messages []llm.Message
}

// NewHistory 在会话头部注入 system prompt（若存在）。
func NewHistory(systemPrompt string) *History {
	h := &History{
		messages: make([]llm.Message, 0, 16),
	}

	if prompt := strings.TrimSpace(systemPrompt); prompt != "" {
		h.Append(llm.Message{
			Role: llm.RoleSystem,
			Text: prompt,
		})
	}

	return h
}

// Append 追加单条消息。
func (h *History) Append(msg llm.Message) {
	h.messages = append(h.messages, msg)
}

// Messages 返回深拷贝，避免调用方意外修改内部状态。
func (h *History) Messages() []llm.Message {
	return llm.CloneMessages(h.messages)
}

// Len 返回当前消息数量。
func (h *History) Len() int {
	return len(h.messages)
}
