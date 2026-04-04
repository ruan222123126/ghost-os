package session

import (
	"time"

	"ghost-os/bridge/guiagent"
)

func (s *Session) SetPendingComputerUseRun(questionID string, state guiagent.State) {
	if s == nil || questionID == "" {
		return
	}
	if s.PendingComputerUseRuns == nil {
		s.PendingComputerUseRuns = make(map[string]guiagent.State, 2)
	}
	s.PendingComputerUseRuns[questionID] = guiagent.CloneState(state)
	s.UpdatedAt = time.Now().UTC()
}

func (s *Session) PendingComputerUseRun(questionID string) (guiagent.State, bool) {
	if s == nil || len(s.PendingComputerUseRuns) == 0 {
		return guiagent.State{}, false
	}
	state, ok := s.PendingComputerUseRuns[questionID]
	if !ok {
		return guiagent.State{}, false
	}
	return guiagent.CloneState(state), true
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
