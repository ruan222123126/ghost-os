package session

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

// Session 保存跨请求会话历史及其基础元数据。
type Session struct {
	ID         string        `json:"id"`
	Messages   []llm.Message `json:"messages"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
	TokenCount int           `json:"token_count"`
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

func newSessionID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fallbackSessionID()
	}

	// RFC 4122 UUID v4
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16])
}

func fallbackSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UTC().UnixNano())
}
