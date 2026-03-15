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
	ID                            string                                  `json:"id"`
	Messages                      []llm.Message                           `json:"messages"`
	CreatedAt                     time.Time                               `json:"created_at"`
	UpdatedAt                     time.Time                               `json:"updated_at"`
	EndedAt                       time.Time                               `json:"ended_at,omitempty"`
	TurnIndex                     int                                     `json:"turn_index,omitempty"`
	TokenCount                    int                                     `json:"token_count"`
	ConversationState             llm.ConversationState                   `json:"conversation_state,omitempty"`
	IterationRuntime              *IterationRuntime                       `json:"iteration_runtime,omitempty"`
	PendingQuestions              map[string]PendingHumanQuestion         `json:"pending_questions,omitempty"`
	HumanAnswers                  map[string]string                       `json:"human_answers,omitempty"`
	PendingGraphQLMutationIntents map[string]PendingGraphQLMutationIntent `json:"pending_graphql_mutation_intents,omitempty"`
	DynamicToolLoads              map[string]DynamicToolLoad              `json:"dynamic_tool_loads,omitempty"`
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
	s.TokenCount += EstimateTokens(cloned[0])
	s.UpdatedAt = time.Now().UTC()
}

// GetMessages 返回裁剪后的上下文消息。
func (s *Session) GetMessages(maxTokens int) []llm.Message {
	if s == nil {
		return nil
	}
	return PruneMessages(s.Messages, maxTokens)
}

// RecalculateTokenCount 重新计算会话 token 估算值，用于持久化校准。
func (s *Session) RecalculateTokenCount() {
	if s == nil {
		return
	}

	total := 0
	for _, msg := range s.Messages {
		total += EstimateTokens(msg)
	}
	s.TokenCount = total
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
