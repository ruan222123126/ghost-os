package session

import (
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

// Session 保存跨请求会话历史及其基础元数据。
//
// 注意：Session 非并发安全（包含 slice/map），同一个会话必须由上层保证串行访问。
type Session struct {
	ID                string                          `json:"id"`
	Messages          []llm.Message                   `json:"messages"`
	CreatedAt         time.Time                       `json:"created_at"`
	UpdatedAt         time.Time                       `json:"updated_at"`
	EndedAt           time.Time                       `json:"ended_at,omitempty"`
	TokenCount        int                             `json:"token_count"`
	ConversationState llm.ConversationState           `json:"conversation_state,omitempty"`
	IterationRuntime  *IterationRuntime               `json:"iteration_runtime,omitempty"`
	PendingQuestions  map[string]PendingHumanQuestion `json:"pending_questions,omitempty"`
	HumanAnswers      map[string]string               `json:"human_answers,omitempty"`
}

var debugNilSessionReceiver = envBool("GHOST_BRIDGE_DEBUG") || envBool("GHOST_DEBUG") || envBool("DEBUG")

const (
	HumanQuestionSelectionSingle   = "single"
	HumanQuestionSelectionMultiple = "multiple"
)

// IterationRecord captures the minimal handoff state between fresh-memory pro/prox agents.
type IterationRecord struct {
	Iteration      int       `json:"iteration"`
	Did            string    `json:"did"`
	Remaining      string    `json:"remaining"`
	Completed      bool      `json:"completed,omitempty"`
	TraceID        string    `json:"trace_id,omitempty"`
	RecordedAt     time.Time `json:"recorded_at,omitempty"`
	FinalChangeLog string    `json:"final_change_log,omitempty"`
}

// IterationRuntime stores the latest pro/prox orchestration state without polluting chat history.
type IterationRuntime struct {
	Mode           string            `json:"mode,omitempty"`
	OriginalTask   string            `json:"original_task,omitempty"`
	MaxIterations  int               `json:"max_iterations,omitempty"`
	Unlimited      bool              `json:"unlimited,omitempty"`
	IterationCount int               `json:"iteration_count,omitempty"`
	Status         string            `json:"status,omitempty"`
	StoppedBy      string            `json:"stopped_by,omitempty"`
	FinalMessage   string            `json:"final_message,omitempty"`
	FinalChangeLog string            `json:"final_change_log,omitempty"`
	StartedAt      time.Time         `json:"started_at,omitempty"`
	UpdatedAt      time.Time         `json:"updated_at,omitempty"`
	Records        []IterationRecord `json:"records,omitempty"`
}

// PendingHumanQuestion 记录 ask_human 的待回答问题。
type PendingHumanQuestion struct {
	Prompt        string                `json:"prompt"`
	SelectionMode string                `json:"selection_mode,omitempty"`
	Options       []HumanQuestionOption `json:"options,omitempty"`
	ToolCallID    string                `json:"tool_call_id"`
	TraceID       string                `json:"trace_id"`
	CreatedAt     time.Time             `json:"created_at"`
}

// HumanQuestionOption 描述 ask_human 暴露给用户的单个选项。
type HumanQuestionOption struct {
	Label       string `json:"label"`
	AllowCustom bool   `json:"allow_custom,omitempty"`
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

// StartIterationRuntime resets the hidden pro/prox iteration state for a new run.
func (s *Session) StartIterationRuntime(mode string, originalTask string, maxIterations int, unlimited bool) {
	if s == nil {
		return
	}
	now := time.Now().UTC()
	s.IterationRuntime = &IterationRuntime{
		Mode:          strings.TrimSpace(mode),
		OriginalTask:  strings.TrimSpace(originalTask),
		MaxIterations: maxIterations,
		Unlimited:     unlimited,
		Status:        "running",
		StartedAt:     now,
		UpdatedAt:     now,
	}
	s.UpdatedAt = now
}

// AppendIterationRecord records one fresh-memory handoff step.
func (s *Session) AppendIterationRecord(record IterationRecord) {
	if s == nil {
		return
	}
	if s.IterationRuntime == nil {
		s.IterationRuntime = &IterationRuntime{}
	}
	record.Did = strings.TrimSpace(record.Did)
	record.Remaining = strings.TrimSpace(record.Remaining)
	record.TraceID = strings.TrimSpace(record.TraceID)
	record.FinalChangeLog = strings.TrimSpace(record.FinalChangeLog)
	if record.RecordedAt.IsZero() {
		record.RecordedAt = time.Now().UTC()
	}
	s.IterationRuntime.Records = append(s.IterationRuntime.Records, record)
	s.IterationRuntime.IterationCount = len(s.IterationRuntime.Records)
	s.IterationRuntime.UpdatedAt = record.RecordedAt
	s.IterationRuntime.Status = "running"
	s.UpdatedAt = record.RecordedAt
}

// FinishIterationRuntime marks the latest pro/prox run as completed, cancelled, or errored.
func (s *Session) FinishIterationRuntime(status string, stoppedBy string, finalMessage string, finalChangeLog string) {
	if s == nil || s.IterationRuntime == nil {
		return
	}
	now := time.Now().UTC()
	s.IterationRuntime.Status = strings.TrimSpace(status)
	s.IterationRuntime.StoppedBy = strings.TrimSpace(stoppedBy)
	s.IterationRuntime.FinalMessage = strings.TrimSpace(finalMessage)
	s.IterationRuntime.FinalChangeLog = strings.TrimSpace(finalChangeLog)
	s.IterationRuntime.IterationCount = len(s.IterationRuntime.Records)
	s.IterationRuntime.UpdatedAt = now
	s.UpdatedAt = now
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
	question.SelectionMode = strings.TrimSpace(question.SelectionMode)
	question.Options = cloneHumanQuestionOptions(question.Options)
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

// RemovePendingQuestion 删除未回答问题，常用于 ask_human 取消场景。
func (s *Session) RemovePendingQuestion(questionID string) (PendingHumanQuestion, bool) {
	if s == nil || len(s.PendingQuestions) == 0 {
		return PendingHumanQuestion{}, false
	}

	questionID = strings.TrimSpace(questionID)
	if questionID == "" {
		return PendingHumanQuestion{}, false
	}

	question, ok := s.PendingQuestions[questionID]
	if !ok {
		return PendingHumanQuestion{}, false
	}

	delete(s.PendingQuestions, questionID)
	delete(s.HumanAnswers, questionID)
	s.UpdatedAt = time.Now().UTC()
	return question, true
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

func cloneHumanQuestionOptions(options []HumanQuestionOption) []HumanQuestionOption {
	if len(options) == 0 {
		return nil
	}
	cloned := make([]HumanQuestionOption, 0, len(options))
	for _, option := range options {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			continue
		}
		cloned = append(cloned, HumanQuestionOption{
			Label:       label,
			AllowCustom: option.AllowCustom,
		})
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func debugNilReceiver(method string) {
	if !debugNilSessionReceiver {
		return
	}
	log.Printf("[SESSION] nil Session receiver (caller bug?): method=%s", strings.TrimSpace(method))
}

func envBool(key string) bool {
	value := strings.TrimSpace(os.Getenv(strings.TrimSpace(key)))
	if value == "" {
		return false
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
