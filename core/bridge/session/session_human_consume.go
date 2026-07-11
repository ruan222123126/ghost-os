package session

import (
	"strings"
	"time"
)

func (s *Session) ConsumeAnsweredQuestion(questionID string) (AnsweredHumanQuestion, bool) {
	if s == nil || len(s.PendingQuestions) == 0 || len(s.HumanAnswers) == 0 {
		return AnsweredHumanQuestion{}, false
	}
	questionID = strings.TrimSpace(questionID)
	if questionID == "" {
		return AnsweredHumanQuestion{}, false
	}
	question, ok := s.PendingQuestions[questionID]
	if !ok {
		return AnsweredHumanQuestion{}, false
	}
	answer, ok := s.HumanAnswers[questionID]
	if !ok {
		return AnsweredHumanQuestion{}, false
	}
	delete(s.PendingQuestions, questionID)
	delete(s.HumanAnswers, questionID)
	s.UpdatedAt = time.Now().UTC()
	return AnsweredHumanQuestion{
		QuestionID: questionID,
		Question:   question,
		Answer:     answer,
	}, true
}
