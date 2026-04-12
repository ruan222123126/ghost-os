package session

import (
	"time"
)

func (s *Session) SetPendingComputerUseRun(questionID string, state PendingComputerUseRunState) {
	if s == nil || questionID == "" {
		return
	}
	if s.PendingComputerUseRuns == nil {
		s.PendingComputerUseRuns = make(map[string]PendingComputerUseRunState, 2)
	}
	s.PendingComputerUseRuns[questionID] = clonePendingComputerUseRunState(state)
	s.UpdatedAt = time.Now().UTC()
}

func (s *Session) PendingComputerUseRun(questionID string) (PendingComputerUseRunState, bool) {
	if s == nil || len(s.PendingComputerUseRuns) == 0 {
		return PendingComputerUseRunState{}, false
	}
	state, ok := s.PendingComputerUseRuns[questionID]
	if !ok {
		return PendingComputerUseRunState{}, false
	}
	return clonePendingComputerUseRunState(state), true
}

func (s *Session) RemovePendingComputerUseRun(questionID string) {
	if s == nil || len(s.PendingComputerUseRuns) == 0 {
		return
	}
	delete(s.PendingComputerUseRuns, questionID)
	if len(s.PendingComputerUseRuns) == 0 {
		s.PendingComputerUseRuns = nil
	}
	s.UpdatedAt = time.Now().UTC()
}
