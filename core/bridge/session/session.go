package session

import (
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

// Session 保存跨请求会话历史及其基础元数据。
//
// 注意：Session 非并发安全（包含 slice/map），同一个会话必须由上层保证串行访问。
type Session struct {
	ID                string                          `json:"id"`
	Title             string                          `json:"title,omitempty"`
	Messages          []llm.Message                   `json:"messages"`
	CreatedAt         time.Time                       `json:"created_at"`
	UpdatedAt         time.Time                       `json:"updated_at"`
	EndedAt           time.Time                       `json:"ended_at,omitempty"`
	TurnIndex         int                             `json:"turn_index,omitempty"`
	TokenCount        int                             `json:"token_count"`
	MessageCount      int                             `json:"message_count"`
	WindowStart       int                             `json:"window_start,omitempty"`
	WindowTokenCount  int                             `json:"window_token_count,omitempty"`
	ConversationState llm.ConversationState           `json:"conversation_state,omitempty"`
	RelayRuntime      *RelayRuntime                   `json:"relay_runtime,omitempty"`
	PendingQuestions  map[string]PendingHumanQuestion `json:"pending_questions,omitempty"`
	HumanAnswers      map[string]string               `json:"human_answers,omitempty"`
	DynamicToolLoads  map[string]DynamicToolLoad      `json:"dynamic_tool_loads,omitempty"`
	DynamicSkillLoads map[string]DynamicSkillLoad     `json:"dynamic_skill_loads,omitempty"`
	AssistantDraft    *AssistantDraft                 `json:"assistant_draft,omitempty"`
	TurnDraft         *TurnDraft                      `json:"turn_draft,omitempty"`

	persistedMessageCount int
	persistedMessages     []llm.Message
}

// NewSession 创建带唯一 ID 的会话，并在首条消息写入 system prompt（若非空）。
func NewSession(systemPrompt string) *Session {
	now := time.Now().UTC()
	s := &Session{
		ID:        newSessionID(),
		Messages:  make([]llm.Message, 0, 16),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if prompt := strings.TrimSpace(systemPrompt); prompt != "" {
		systemMsg := llm.Message{
			Role: llm.RoleSystem,
			Text: prompt,
		}
		s.Messages = append(s.Messages, systemMsg)
		s.TokenCount = EstimateTokens(systemMsg)
		s.MessageCount = 1
		s.WindowTokenCount = s.TokenCount
	}

	return s
}

// AddMessage 追加消息，并维护更新时间与 token 估算值。
func (s *Session) AddMessage(msg llm.Message) {
	if s == nil {
		debugNilReceiver("AddMessage")
		return
	}

	cloned := llm.CloneMessages([]llm.Message{msg})
	if len(cloned) == 0 {
		return
	}

	s.Messages = append(s.Messages, cloned[0])
	messageTokens := EstimateTokens(cloned[0])
	s.TokenCount += messageTokens
	s.MessageCount++
	s.WindowTokenCount += messageTokens
	s.UpdatedAt = time.Now().UTC()
}

// GetMessages 返回裁剪后的上下文消息。
func (s *Session) GetMessages(maxTokens int) []llm.Message {
	if s == nil {
		return nil
	}
	return PruneMessages(s.Messages, maxTokens)
}

// RecalculateTokenCount 重新计算当前窗口 token 估算值。
//
// 当 Messages 表示完整历史时，该方法也会同步校正全会话 token/message 统计；
// 当 Messages 只表示热窗口时，调用方必须自行维护全会话 TokenCount / MessageCount。
func (s *Session) RecalculateTokenCount() {
	if s == nil {
		return
	}

	total := 0
	for _, msg := range s.Messages {
		total += EstimateTokens(msg)
	}
	s.WindowTokenCount = total
	if s.MessageCount == 0 || s.MessageCount == len(s.Messages) {
		s.MessageCount = len(s.Messages)
		s.TokenCount = total
		s.WindowStart = 0
	}
}

// MarkEnded 标记会话已结束，后续不应再继续使用相同 session id 续跑。
func (s *Session) MarkEnded(at time.Time) {
	if s == nil {
		debugNilReceiver("MarkEnded")
		return
	}
	when := at.UTC()
	if when.IsZero() {
		when = time.Now().UTC()
	}
	s.EndedAt = when
	s.UpdatedAt = when
}

// IsEnded 返回会话是否已结束。
func (s *Session) IsEnded() bool {
	if s == nil {
		return false
	}
	return !s.EndedAt.IsZero()
}

func (s *Session) setPersistedSnapshot() {
	if s == nil {
		return
	}
	s.persistedMessageCount = s.MessageCount
	s.persistedMessages = llm.CloneMessages(s.Messages)
}
