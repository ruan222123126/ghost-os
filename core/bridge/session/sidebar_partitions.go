package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

const (
	sessionSidebarPartitionVersion        = 1
	sessionSidebarUnclassifiedPartitionID = "__unclassified__"
	sessionSidebarUIDirname               = "ui"
	sessionSidebarStateFilename           = "session_sidebar_partitions.json"
)

type SessionSidebarPartition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SessionSidebarPartitionState struct {
	Version     int                       `json:"version"`
	Partitions  []SessionSidebarPartition `json:"partitions"`
	Assignments map[string]string         `json:"assignments"`
}

func (s *Store) LoadSidebarPartitionState() (SessionSidebarPartitionState, error) {
	state := emptySidebarPartitionState()
	err := s.withStoreLock(func() error {
		loaded, changed, err := s.loadSidebarPartitionStateLocked()
		if err != nil {
			return err
		}
		state = loaded
		if !changed {
			return nil
		}
		return s.writeSidebarPartitionStateLocked(state)
	})
	return state, err
}

func (s *Store) SaveSidebarPartitionState(input SessionSidebarPartitionState) (SessionSidebarPartitionState, error) {
	state := emptySidebarPartitionState()
	err := s.withStoreLock(func() error {
		knownSessionIDs, err := s.listKnownSessionIDsLocked()
		if err != nil {
			return err
		}
		state = normalizeSidebarPartitionState(input, knownSessionIDs)
		return s.writeSidebarPartitionStateLocked(state)
	})
	return state, err
}

func (s *Store) cleanupSidebarPartitionStateLocked(sessionID string) error {
	state, _, err := s.loadSidebarPartitionStateLocked()
	if err != nil || !changedAssignment(state.Assignments, sessionID) {
		return err
	}

	delete(state.Assignments, sessionID)
	return s.writeSidebarPartitionStateLocked(state)
}

func (s *Store) loadSidebarPartitionStateLocked() (SessionSidebarPartitionState, bool, error) {
	path := s.sidebarPartitionStatePath()
	decoded, exists, err := readSidebarPartitionStateFile(path)
	if err != nil || !exists {
		return decoded, false, err
	}

	knownSessionIDs, err := s.listKnownSessionIDsLocked()
	if err != nil {
		return SessionSidebarPartitionState{}, false, err
	}
	normalized := normalizeSidebarPartitionState(decoded, knownSessionIDs)
	return normalized, !reflect.DeepEqual(decoded, normalized), nil
}

func (s *Store) writeSidebarPartitionStateLocked(state SessionSidebarPartitionState) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode session sidebar partition state: %w", err)
	}
	return writeFileAtomic(s.sidebarPartitionStatePath(), encoded)
}

func (s *Store) sidebarPartitionStatePath() string {
	return filepath.Join(s.baseDir, sessionSidebarUIDirname, sessionSidebarStateFilename)
}

func readSidebarPartitionStateFile(path string) (SessionSidebarPartitionState, bool, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return emptySidebarPartitionState(), false, nil
		}
		return SessionSidebarPartitionState{}, false, fmt.Errorf("read session sidebar partition state: %w", err)
	}

	var decoded SessionSidebarPartitionState
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return SessionSidebarPartitionState{}, false, fmt.Errorf("decode session sidebar partition state: %w", err)
	}
	return decoded, true, nil
}

func normalizeSidebarPartitionState(
	input SessionSidebarPartitionState,
	knownSessionIDs map[string]struct{},
) SessionSidebarPartitionState {
	partitions := normalizeSidebarPartitions(input.Partitions)
	knownPartitionIDs := partitionIDSet(partitions)
	assignments := normalizeSidebarAssignments(input.Assignments, knownSessionIDs, knownPartitionIDs)
	return SessionSidebarPartitionState{
		Version:     sessionSidebarPartitionVersion,
		Partitions:  partitions,
		Assignments: assignments,
	}
}

func emptySidebarPartitionState() SessionSidebarPartitionState {
	return SessionSidebarPartitionState{
		Version:     sessionSidebarPartitionVersion,
		Partitions:  []SessionSidebarPartition{},
		Assignments: map[string]string{},
	}
}

func normalizeSidebarPartitions(partitions []SessionSidebarPartition) []SessionSidebarPartition {
	seen := make(map[string]struct{}, len(partitions))
	normalized := make([]SessionSidebarPartition, 0, len(partitions))
	for _, partition := range partitions {
		id := strings.TrimSpace(partition.ID)
		name := strings.TrimSpace(partition.Name)
		if skipSidebarPartition(id, name, seen) {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, SessionSidebarPartition{ID: id, Name: name})
	}
	return normalized
}

func skipSidebarPartition(id string, name string, seen map[string]struct{}) bool {
	if id == "" || name == "" || id == sessionSidebarUnclassifiedPartitionID {
		return true
	}
	_, duplicated := seen[id]
	return duplicated
}

func partitionIDSet(partitions []SessionSidebarPartition) map[string]struct{} {
	ids := make(map[string]struct{}, len(partitions))
	for _, partition := range partitions {
		ids[partition.ID] = struct{}{}
	}
	return ids
}

func normalizeSidebarAssignments(
	assignments map[string]string,
	knownSessionIDs map[string]struct{},
	knownPartitionIDs map[string]struct{},
) map[string]string {
	normalized := make(map[string]string, len(assignments))
	for sessionID, partitionID := range assignments {
		id := strings.TrimSpace(sessionID)
		target := strings.TrimSpace(partitionID)
		if !keepSidebarAssignment(id, target, knownSessionIDs, knownPartitionIDs) {
			continue
		}
		normalized[id] = target
	}
	return normalized
}

func keepSidebarAssignment(
	sessionID string,
	partitionID string,
	knownSessionIDs map[string]struct{},
	knownPartitionIDs map[string]struct{},
) bool {
	if sessionID == "" || partitionID == "" || partitionID == sessionSidebarUnclassifiedPartitionID {
		return false
	}
	if _, ok := knownSessionIDs[sessionID]; !ok {
		return false
	}
	_, ok := knownPartitionIDs[partitionID]
	return ok
}

func changedAssignment(assignments map[string]string, sessionID string) bool {
	if assignments == nil {
		return false
	}
	_, ok := assignments[sessionID]
	return ok
}

func (s *Store) listKnownSessionIDsLocked() (map[string]struct{}, error) {
	s.importLegacySessionsForListingLocked()
	rows, err := s.db.Query(`SELECT id FROM sessions ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list session ids: %w", err)
	}
	defer rows.Close()

	known := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan session id: %w", err)
		}
		known[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list session ids: %w", err)
	}
	return known, nil
}

func writeFileAtomic(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create directory %q: %w", dir, err)
	}

	tempFile, err := os.CreateTemp(dir, ".session-sidebar-partitions-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file for %q: %w", path, err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if err := writeTempFile(tempFile, content); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("rename %q to %q: %w", tempPath, path, err)
	}
	return nil
}

func writeTempFile(file *os.File, content []byte) error {
	if _, err := file.Write(content); err != nil {
		file.Close()
		return fmt.Errorf("write temp session sidebar partition state: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temp session sidebar partition state: %w", err)
	}
	return nil
}
