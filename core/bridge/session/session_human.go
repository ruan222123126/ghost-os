package session

import (
	"sort"
	"strings"
	"time"
)

const (
	HumanQuestionSelectionSingle   = "single"
	HumanQuestionSelectionMultiple = "multiple"
)

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
