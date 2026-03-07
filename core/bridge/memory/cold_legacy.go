package memory

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"ghost-os/bridge/llm"
)

const (
	coldFilePrefix = "session-"
	coldFileSuffix = ".json"
)

type coldArchiveFile struct {
	SessionID  string        `json:"session_id"`
	ArchivedAt time.Time     `json:"archived_at"`
	Messages   []llm.Message `json:"messages"`
	Summary    string        `json:"summary,omitempty"`
	Entities   []string      `json:"entities,omitempty"`
}

type LegacyArchiveRecord struct {
	Path     string
	MonthDir string
	Archive  coldArchiveFile
}

type LegacyColdStore struct {
	baseDir     string
	truth       *TruthWriter
	truthMapper *TruthMapper
	mu          sync.Mutex
}

func NewLegacyColdStore(baseDir string) *LegacyColdStore {
	return &LegacyColdStore{baseDir: resolveMemoryPath(baseDir)}
}

func (s *LegacyColdStore) SetTruthShadow(writer *TruthWriter, mapper *TruthMapper) {
	if s == nil {
		return
	}
	s.truth = writer
	s.truthMapper = mapper
}

func (s *LegacyColdStore) BaseDir() string {
	if s == nil {
		return ""
	}
	return s.baseDir
}

func (s *LegacyColdStore) Archive(sessionID string, messages []llm.Message) error {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return fmt.Errorf("session id is required")
	}
	if s.baseDir == "" {
		return fmt.Errorf("cold memory base dir is empty")
	}

	now := time.Now().UTC()
	payload := coldArchiveFile{
		SessionID:  sid,
		ArchivedAt: now,
		Messages:   llm.CloneMessages(messages),
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	targetDir := filepath.Join(s.baseDir, now.Format("2006-01"))
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return fmt.Errorf("create cold memory directory: %w", err)
	}

	targetPath := filepath.Join(targetDir, coldFilePrefix+sid+coldFileSuffix)
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cold archive: %w", err)
	}
	data = append(data, '\n')

	tmpPath := fmt.Sprintf("%s.tmp-%d", targetPath, time.Now().UTC().UnixNano())
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write cold archive temp file: %w", err)
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace cold archive file: %w", err)
	}

	if s.truth != nil && s.truth.DualWriteEnabled() && s.truthMapper != nil {
		for _, object := range s.truthMapper.MapArchiveMessages(sid, now, messages) {
			if err := writeTruthObjectShadow(s.truth, truthEventTypeArchiveMessage, object, ""); err != nil {
				if handleErr := handleTruthShadowWriteError(s.truth, "", object.ObjectID, err); handleErr != nil {
					return handleErr
				}
				log.Printf("[MEMORY] truth archive shadow write skipped after error: session_id=%s object_id=%s", sid, object.ObjectID)
			}
		}
	}
	return nil
}

func (s *LegacyColdStore) Retrieve(query MemoryQuery) ([]MemoryEntry, error) {
	if s == nil || s.baseDir == "" {
		return nil, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	files, err := s.collectArchiveFilesLocked(query.TimeRange)
	if err != nil {
		return nil, err
	}

	results := make([]MemoryEntry, 0, 64)
	for _, path := range files {
		archive, err := s.readArchiveFileLocked(path)
		if err != nil {
			continue
		}
		if query.TimeRange != nil && !query.TimeRange.Contains(archive.ArchivedAt) {
			continue
		}
		for i, msg := range archive.Messages {
			entry := MemoryEntry{
				ID:         fmt.Sprintf("%s:%06d", archive.SessionID, i),
				Content:    messageToContent(msg),
				Type:       MemoryTypeMessage,
				Timestamp:  archive.ArchivedAt,
				Source:     "archive",
				Summary:    summarizeLine(messageToContent(msg), 220),
				Confidence: 1,
				Metadata: map[string]any{
					"layer":        "cold",
					"source":       "archive",
					"session_id":   archive.SessionID,
					"role":         string(msg.Role),
					"tool_call_id": strings.TrimSpace(msg.ToolCallID),
				},
			}
			entry = normalizeEntry(entry)
			if !entryMatchesQuery(entry, query) {
				continue
			}
			results = append(results, entry)
		}
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}
	return cloneEntries(results), nil
}

func (s *LegacyColdStore) ListSessions(timeRange TimeRange) ([]string, error) {
	if s == nil || s.baseDir == "" {
		return nil, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	files, err := s.collectArchiveFilesLocked(&timeRange)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(files))
	out := make([]string, 0, len(files))
	for _, path := range files {
		archive, err := s.readArchiveFileLocked(path)
		if err != nil {
			continue
		}
		if !timeRange.Contains(archive.ArchivedAt) {
			continue
		}
		if _, ok := seen[archive.SessionID]; ok {
			continue
		}
		seen[archive.SessionID] = struct{}{}
		out = append(out, archive.SessionID)
	}
	sort.Strings(out)
	return out, nil
}

func (s *LegacyColdStore) ListArchives(timeRange *TimeRange) ([]ColdArchive, error) {
	if s == nil || s.baseDir == "" {
		return nil, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	files, err := s.collectArchiveFilesLocked(timeRange)
	if err != nil {
		return nil, err
	}

	out := make([]ColdArchive, 0, len(files))
	for _, path := range files {
		archive, err := s.readArchiveFileLocked(path)
		if err != nil {
			continue
		}
		if timeRange != nil && !timeRange.Contains(archive.ArchivedAt) {
			continue
		}
		out = append(out, ColdArchive{
			SessionID:  archive.SessionID,
			ArchivedAt: archive.ArchivedAt.UTC(),
			Messages:   llm.CloneMessages(archive.Messages),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ArchivedAt.Equal(out[j].ArchivedAt) {
			return out[i].SessionID < out[j].SessionID
		}
		return out[i].ArchivedAt.After(out[j].ArchivedAt)
	})
	return out, nil
}

func (s *LegacyColdStore) ListArchiveRecords(timeRange *TimeRange) ([]LegacyArchiveRecord, error) {
	if s == nil || s.baseDir == "" {
		return nil, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	files, err := s.collectArchiveFilesLocked(timeRange)
	if err != nil {
		return nil, err
	}
	records := make([]LegacyArchiveRecord, 0, len(files))
	for _, path := range files {
		archive, err := s.readArchiveFileLocked(path)
		if err != nil {
			continue
		}
		if timeRange != nil && !timeRange.Contains(archive.ArchivedAt) {
			continue
		}
		records = append(records, LegacyArchiveRecord{
			Path:     path,
			MonthDir: filepath.Base(filepath.Dir(path)),
			Archive:  archive,
		})
	}
	return records, nil
}

func (s *LegacyColdStore) collectArchiveFilesLocked(timeRange *TimeRange) ([]string, error) {
	monthDirs, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read cold memory root: %w", err)
	}

	files := make([]string, 0, 64)
	for _, monthDir := range monthDirs {
		if !monthDir.IsDir() {
			continue
		}
		monthName := monthDir.Name()
		if !monthDirMatchesRange(monthName, timeRange) {
			continue
		}

		entries, err := os.ReadDir(filepath.Join(s.baseDir, monthName))
		if err != nil {
			return nil, fmt.Errorf("read cold month directory %q: %w", monthName, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasPrefix(name, coldFilePrefix) || !strings.HasSuffix(name, coldFileSuffix) {
				continue
			}
			files = append(files, filepath.Join(s.baseDir, monthName, name))
		}
	}
	sort.Strings(files)
	return files, nil
}

func monthDirMatchesRange(name string, timeRange *TimeRange) bool {
	if timeRange == nil {
		return true
	}
	monthStart, err := time.Parse("2006-01", name)
	if err != nil {
		return true
	}
	monthStart = monthStart.UTC()
	monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Nanosecond)
	if !timeRange.Start.IsZero() && monthEnd.Before(timeRange.Start.UTC()) {
		return false
	}
	if !timeRange.End.IsZero() && monthStart.After(timeRange.End.UTC()) {
		return false
	}
	return true
}

func (s *LegacyColdStore) readArchiveFileLocked(path string) (coldArchiveFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return coldArchiveFile{}, err
	}

	var archive coldArchiveFile
	if err := json.Unmarshal(data, &archive); err != nil {
		return coldArchiveFile{}, err
	}
	if archive.ArchivedAt.IsZero() {
		info, statErr := os.Stat(path)
		if statErr == nil {
			archive.ArchivedAt = info.ModTime().UTC()
		} else {
			archive.ArchivedAt = time.Now().UTC()
		}
	} else {
		archive.ArchivedAt = archive.ArchivedAt.UTC()
	}
	archive.SessionID = strings.TrimSpace(archive.SessionID)
	archive.Messages = llm.CloneMessages(archive.Messages)
	return archive, nil
}
