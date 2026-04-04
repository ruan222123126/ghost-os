package session

import "strings"

func (s *Session) handleRemovedPendingQuestion(questionID string, question PendingHumanQuestion) {
	if s == nil || strings.TrimSpace(question.ToolName) != "computer_use" {
		return
	}
	s.RemovePendingComputerUseRun(questionID)
}
