package session

import (
	"crypto/rand"
	"fmt"
	"sort"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

// Session 保存跨请求会话历史及其基础元数据。
type Session struct {
	ID               string                          `json:"id"`
	Messages         []llm.Message                   `json:"messages"`
	CreatedAt        time.Time                       `json:"created_at"`
	UpdatedAt        time.Time                       `json:"updated_at"`
	TokenCount       int                             `json:"token_count"`
	MemoryMetadata   MemoryMetadata                  `json:"memory_metadata,omitempty"`
	PendingQuestions map[string]PendingHumanQuestion `json:"pending_questions,omitempty"`
	HumanAnswers     map[string]string               `json:"human_answers,omitempty"`
}

// MemoryMetadata 记录会话在分层记忆中的状态。
type MemoryMetadata struct {
	ArchivedAt   time.Time `json:"archived_at,omitempty"`
	AccessCount  int       `json:"access_count"`
	LastAccessAt time.Time `json:"last_access_at,omitempty"`
}

func (m MemoryMetadata) IsZero() bool {
	return m.ArchivedAt.IsZero() && m.AccessCount == 0 && m.LastAccessAt.IsZero()
}

// PendingHumanQuestion 记录 ask_human 的待回答问题。
type PendingHumanQuestion struct {
	Prompt     string    `json:"prompt"`
	ToolCallID string    `json:"tool_call_id"`
	TraceID    string    `json:"trace_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// AnsweredHumanQuestion 表示可注入回会话的问答结果。
type AnsweredHumanQuestion struct {
	QuestionID string
	Question   PendingHumanQuestion
	Answer     string
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

// MarkMemoryArchived 标记会话已归档到冷存储。
func (s *Session) MarkMemoryArchived(at time.Time) {
	if s == nil {
		return
	}
	when := at.UTC()
	if when.IsZero() {
		when = time.Now().UTC()
	}
	s.MemoryMetadata.ArchivedAt = when
	s.UpdatedAt = when
}

// MarkMemoryAccess 更新会话被记忆系统访问的统计信息。
func (s *Session) MarkMemoryAccess(at time.Time) {
	if s == nil {
		return
	}
	when := at.UTC()
	if when.IsZero() {
		when = time.Now().UTC()
	}
	s.MemoryMetadata.AccessCount++
	s.MemoryMetadata.LastAccessAt = when
	s.UpdatedAt = when
}

// AddPendingQuestion 注册待用户回复的问题。
func (s *Session) AddPendingQuestion(questionID string, question PendingHumanQuestion) {
	if s == nil {
		return
	}

	questionID = strings.TrimSpace(questionID)
	if questionID == "" {
		return
	}

	if s.PendingQuestions == nil {
		s.PendingQuestions = make(map[string]PendingHumanQuestion, 2)
	}
	if s.HumanAnswers == nil {
		s.HumanAnswers = make(map[string]string, 2)
	}
	if question.CreatedAt.IsZero() {
		question.CreatedAt = time.Now().UTC()
	}
	s.PendingQuestions[questionID] = question
	delete(s.HumanAnswers, questionID)
	s.UpdatedAt = time.Now().UTC()
}

// HasPendingQuestion 判断指定问题是否仍处于待回答状态。
func (s *Session) HasPendingQuestion(questionID string) bool {
	if s == nil || len(s.PendingQuestions) == 0 {
		return false
	}
	_, ok := s.PendingQuestions[strings.TrimSpace(questionID)]
	return ok
}

// SetHumanAnswer 记录用户回答；仅对 pending question 生效。
func (s *Session) SetHumanAnswer(questionID string, answer string) bool {
	if s == nil || len(s.PendingQuestions) == 0 {
		return false
	}

	questionID = strings.TrimSpace(questionID)
	if questionID == "" {
		return false
	}
	if _, ok := s.PendingQuestions[questionID]; !ok {
		return false
	}
	if s.HumanAnswers == nil {
		s.HumanAnswers = make(map[string]string, 2)
	}
	s.HumanAnswers[questionID] = answer
	s.UpdatedAt = time.Now().UTC()
	return true
}

// PopAnsweredQuestions 返回已回答问题，并从 pending/answer 中移除。
func (s *Session) PopAnsweredQuestions() []AnsweredHumanQuestion {
	if s == nil || len(s.PendingQuestions) == 0 || len(s.HumanAnswers) == 0 {
		return nil
	}

	resolved := make([]AnsweredHumanQuestion, 0, len(s.HumanAnswers))
	for questionID, answer := range s.HumanAnswers {
		question, ok := s.PendingQuestions[questionID]
		if !ok {
			continue
		}
		resolved = append(resolved, AnsweredHumanQuestion{
			QuestionID: questionID,
			Question:   question,
			Answer:     answer,
		})
		delete(s.PendingQuestions, questionID)
		delete(s.HumanAnswers, questionID)
	}
	if len(resolved) == 0 {
		return nil
	}

	sort.Slice(resolved, func(i, j int) bool {
		return resolved[i].Question.CreatedAt.Before(resolved[j].Question.CreatedAt)
	})
	s.UpdatedAt = time.Now().UTC()
	return resolved
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
