package agent

type completionAttemptState struct {
	completionDeltaEmitted bool
}

func (s *completionAttemptState) reset() {
	if s == nil {
		return
	}
	s.completionDeltaEmitted = false
}

func (s *completionAttemptState) markCompletionDeltaEmitted() {
	if s == nil {
		return
	}
	s.completionDeltaEmitted = true
}

func (s *completionAttemptState) hasCompletionDeltaEmitted() bool {
	return s != nil && s.completionDeltaEmitted
}
