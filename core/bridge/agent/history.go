package agent

import (
	"strings"

	"ghost-os/bridge/llm"
)

// History 仅负责维护会话消息序列，不包含业务决策。
type History struct {
	messages          []llm.Message
	conversationState llm.ConversationState
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

// NewHistoryFromMessages 使用已有消息初始化历史。
func NewHistoryFromMessages(messages []llm.Message) *History {
	return &History{
		messages: llm.CloneMessages(messages),
	}
}

// Append 追加单条消息。
func (h *History) Append(msg llm.Message) {
	h.messages = append(h.messages, msg)
}

// UpdateSystemPrompt 覆盖或注入首条 system prompt，供运行期动态调整工具上下文。
func (h *History) UpdateSystemPrompt(newPrompt string) {
	if h == nil {
		return
	}

	prompt := strings.TrimSpace(newPrompt)
	if prompt == "" {
		return
	}

	desired := llm.Message{
		Role: llm.RoleSystem,
		Text: prompt,
	}

	if len(h.messages) == 0 {
		h.messages = append(h.messages, desired)
		return
	}
	if h.messages[0].Role == llm.RoleSystem {
		h.messages[0] = desired
		return
	}

	h.messages = append([]llm.Message{desired}, h.messages...)
}

// Messages 返回深拷贝，避免调用方意外修改内部状态。
func (h *History) Messages() []llm.Message {
	return llm.CloneMessages(h.messages)
}

// Len 返回当前消息数量。
func (h *History) Len() int {
	return len(h.messages)
}

// Clone 返回一份可独立修改的历史副本，供单回合暂存使用。
func (h *History) Clone() *History {
	if h == nil {
		return NewHistory("")
	}

	return &History{
		messages:          llm.CloneMessages(h.messages),
		conversationState: h.conversationState,
	}
}

func (h *History) ConversationState() llm.ConversationState {
	if h == nil {
		return llm.ConversationState{}
	}
	return h.conversationState
}

func (h *History) SetConversationState(state llm.ConversationState) {
	if h == nil {
		return
	}
	h.conversationState = state
}
