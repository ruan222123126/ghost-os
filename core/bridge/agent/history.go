package agent

import (
	"strings"

	"ghost-os/bridge/llm"
)

// History 仅负责维护会话消息序列，不包含业务决策。
type History struct {
	messages          []llm.Message
	conversationState llm.ConversationState
	maxMessages       int // 0 表示不限制，正数表示最大消息条数（超出时裁剪，保留系统消息和最近的）
}

// DefaultMaxHistoryMessages 默认保留最近 100 条消息（约 50 轮对话）。
const DefaultMaxHistoryMessages = 100

// NewHistory 在会话头部注入 system prompt（若存在）。
func NewHistory(systemPrompt string) *History {
	return NewHistoryWithMaxMessages(systemPrompt, DefaultMaxHistoryMessages)
}

// NewHistoryWithMaxMessages 创建带最大消息数限制的历史记录。
func NewHistoryWithMaxMessages(systemPrompt string, maxMessages int) *History {
	if maxMessages <= 0 {
		maxMessages = DefaultMaxHistoryMessages
	}
	h := &History{
		messages:    make([]llm.Message, 0, maxMessages),
		maxMessages: maxMessages,
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

// Append 追加单条消息，若超出最大长度则自动裁剪（保留系统消息和最近的）。
func (h *History) Append(msg llm.Message) {
	h.messages = append(h.messages, msg)
	if h.maxMessages > 0 && len(h.messages) > h.maxMessages {
		h.trimToMax()
	}
}

// trimToMax 裁剪消息到最大长度，保留系统消息（如果存在）以及最近的 (maxMessages-1) 条消息。
func (h *History) trimToMax() {
	if h.maxMessages <= 0 || len(h.messages) <= h.maxMessages {
		return
	}
	// 检查是否有系统消息
	hasSystem := len(h.messages) > 0 && h.messages[0].Role == llm.RoleSystem
	systemMsg := llm.Message{}
	if hasSystem {
		systemMsg = h.messages[0]
	}

	keep := h.maxMessages
	if hasSystem {
		keep-- // 系统消息占一个位置，额外保留最近 keep 条
	}
	if keep <= 0 {
		// 极端情况：maxMessages=1 且存在系统消息，则只保留系统消息
		if hasSystem {
			h.messages = []llm.Message{systemMsg}
		} else {
			// 保留最后一条
			h.messages = h.messages[len(h.messages)-1:]
		}
		return
	}
	// 保留系统消息 + 最近 keep 条消息
	recent := h.messages[len(h.messages)-keep:]
	if hasSystem {
		h.messages = append([]llm.Message{systemMsg}, recent...)
	} else {
		h.messages = recent
	}
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
