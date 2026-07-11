package tasks

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type storedRunLogFile struct {
	name     string
	path     string
	sortTime time.Time
}

func (s *Store) AppendRunLog(run RunLog) error {
	prepared, err := prepareStoredRunLog(run)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.writeRunLogLocked(prepared); err != nil {
		return err
	}
	if err := s.pruneRunLogsLocked(prepared.TaskID, DefaultRunLogRetention); err != nil {
		log.Printf("task store: prune task logs task_id=%q keep=%d err=%v", prepared.TaskID, DefaultRunLogRetention, err)
	}
	return nil
}

func (s *Store) ListRunLogs(taskID string, limit int) ([]RunLog, error) {
	id := strings.TrimSpace(taskID)
	if !IsValidTaskID(id) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidTaskID, taskID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	runs, err := listStoredRunLogs(filepath.Join(s.logsDir, id), id)
	if err != nil {
		return nil, err
	}
	sortRunLogsNewestFirst(runs)
	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

func prepareStoredRunLog(run RunLog) (RunLog, error) {
	NormalizeRunLog(&run)
	if !IsValidTaskID(run.TaskID) {
		return RunLog{}, fmt.Errorf("%w: %q", ErrInvalidTaskID, run.TaskID)
	}
	if run.RunID == "" {
		run.RunID = NewRunID()
	}
	return run, nil
}

func (s *Store) writeRunLogLocked(run RunLog) error {
	logDir := filepath.Join(s.logsDir, run.TaskID)
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return fmt.Errorf("create task log directory %q: %w", logDir, err)
	}

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal task log %q/%q: %w", run.TaskID, run.RunID, err)
	}
	data = append(data, '\n')

	path := filepath.Join(logDir, run.RunID+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write task log %q/%q: %w", run.TaskID, run.RunID, err)
	}
	return nil
}

func listStoredRunLogs(logDir string, taskID string) ([]RunLog, error) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []RunLog{}, nil
		}
		return nil, fmt.Errorf("read task log directory %q: %w", logDir, err)
	}

	runs := make([]RunLog, 0, len(entries))
	for _, entry := range entries {
		run, ok, err := readStoredRunLog(logDir, taskID, entry)
		if err != nil {
			return nil, err
		}
		if ok {
			runs = append(runs, run)
		}
	}
	return runs, nil
}

func readStoredRunLog(logDir string, taskID string, entry os.DirEntry) (RunLog, bool, error) {
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
		return RunLog{}, false, nil
	}

	path := filepath.Join(logDir, entry.Name())
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return RunLog{}, false, nil
		}
		return RunLog{}, false, fmt.Errorf("read task log %q/%q: %w", taskID, entry.Name(), err)
	}

	var run RunLog
	if err := json.Unmarshal(data, &run); err != nil {
		return RunLog{}, false, fmt.Errorf("decode task log %q/%q: %w", taskID, entry.Name(), err)
	}
	NormalizeRunLog(&run)
	return run, true, nil
}

func sortRunLogsNewestFirst(runs []RunLog) {
	sort.Slice(runs, func(i, j int) bool {
		return RunLogSortTime(runs[i]).After(RunLogSortTime(runs[j]))
	})
}

func (s *Store) pruneRunLogsLocked(taskID string, keep int) error {
	id := strings.TrimSpace(taskID)
	if keep <= 0 {
		return nil
	}

	files, err := s.listRunLogFilesLocked(id)
	if err != nil || len(files) <= keep {
		return err
	}
	for _, item := range files[:len(files)-keep] {
		if err := os.Remove(item.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete task log %q/%q: %w", id, item.name, err)
		}
	}
	return nil
}

func (s *Store) listRunLogFilesLocked(taskID string) ([]storedRunLogFile, error) {
	logDir := filepath.Join(s.logsDir, taskID)
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read task log directory %q: %w", logDir, err)
	}

	files := make([]storedRunLogFile, 0, len(entries))
	for _, entry := range entries {
		file, ok, err := readStoredRunLogFile(logDir, taskID, entry)
		if err != nil {
			return nil, err
		}
		if ok {
			files = append(files, file)
		}
	}
	sort.Slice(files, func(i, j int) bool {
		if !files[i].sortTime.Equal(files[j].sortTime) {
			return files[i].sortTime.Before(files[j].sortTime)
		}
		return files[i].name < files[j].name
	})
	return files, nil
}

func readStoredRunLogFile(
	logDir string,
	taskID string,
	entry os.DirEntry,
) (storedRunLogFile, bool, error) {
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
		return storedRunLogFile{}, false, nil
	}

	path := filepath.Join(logDir, entry.Name())
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return storedRunLogFile{}, false, nil
		}
		return storedRunLogFile{}, false, fmt.Errorf("read task log %q/%q: %w", taskID, entry.Name(), err)
	}

	file := storedRunLogFile{name: entry.Name(), path: path}
	var run RunLog
	if err := json.Unmarshal(data, &run); err == nil {
		NormalizeRunLog(&run)
		file.sortTime = RunLogSortTime(run)
	}
	return file, true, nil
}
