package session

import "fmt"

func (s *Store) syncSessionHumanLogLocked(sessionID string) error {
	if s == nil || s.humanLog == nil {
		return nil
	}
	if err := s.humanLog.Sync(s.db, sessionID); err != nil {
		return fmt.Errorf("sync session human log: %w", err)
	}
	return nil
}
