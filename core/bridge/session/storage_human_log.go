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

func removeSessionHumanLogFile(store *Store, sessionID string) error {
	if store == nil || store.humanLog == nil {
		return nil
	}
	_, err := store.humanLog.Delete(sessionID)
	return err
}

func deleteSessionHumanLogFile(store *Store, sessionID string) (bool, error) {
	if store == nil || store.humanLog == nil {
		return false, nil
	}
	return store.humanLog.Delete(sessionID)
}
