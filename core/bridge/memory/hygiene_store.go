package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type HygieneStore struct {
	baseDir     string
	recordsPath string

	mu      sync.RWMutex
	records map[string]HygieneRecord
}

type hygieneRecordsSnapshot struct {
	Records []HygieneRecord `json:"records"`
}

func NewHygieneStore(baseDir string) *HygieneStore {
	resolved := resolveMemoryPath(baseDir)
	store := &HygieneStore{
		baseDir: resolved,
		records: make(map[string]HygieneRecord),
	}
	if resolved != "" {
		store.recordsPath = filepath.Join(resolved, defaultHygieneRecordsPathName)
	}
	return store
}

func (s *HygieneStore) BaseDir() string {
	if s == nil {
		return ""
	}
	return s.baseDir
}

func (s *HygieneStore) Load() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *HygieneStore) Get(target HygieneTarget) (HygieneRecord, bool) {
	if s == nil {
		return HygieneRecord{}, false
	}
	key, ok := normalizeHygieneTarget(target).Key()
	if !ok {
		return HygieneRecord{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, exists := s.records[key]
	if !exists {
		return HygieneRecord{}, false
	}
	return record, true
}

func (s *HygieneStore) BatchGet(targets []HygieneTarget) map[string]HygieneRecord {
	if s == nil || len(targets) == 0 {
		return nil
	}
	results := make(map[string]HygieneRecord, len(targets))
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, target := range targets {
		key, ok := normalizeHygieneTarget(target).Key()
		if !ok {
			continue
		}
		if record, exists := s.records[key]; exists {
			results[key] = record
		}
	}
	if len(results) == 0 {
		return nil
	}
	return results
}

func (s *HygieneStore) ListQuarantined(level string, limit int) []HygieneRecord {
	if s == nil {
		return nil
	}
	resolvedLevel := normalizeQuarantineLevel(level)
	requireExactLevel := strings.TrimSpace(level) != ""
	if requireExactLevel && resolvedLevel == QuarantineLevelNone {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HygieneRecord, 0, len(s.records))
	for _, record := range s.records {
		if record.QuarantineLevel == QuarantineLevelNone {
			continue
		}
		if requireExactLevel && record.QuarantineLevel != resolvedLevel {
			continue
		}
		out = append(out, record)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].GarbageScore != out[j].GarbageScore {
			return out[i].GarbageScore > out[j].GarbageScore
		}
		if out[i].GarbageVotes != out[j].GarbageVotes {
			return out[i].GarbageVotes > out[j].GarbageVotes
		}
		if !out[i].LastScoredAt.Equal(out[j].LastScoredAt) {
			return out[i].LastScoredAt.After(out[j].LastScoredAt)
		}
		return out[i].Key < out[j].Key
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (s *HygieneStore) Stats() HygieneStats {
	stats := HygieneStats{
		Levels: map[string]int{
			QuarantineLevelNone:   0,
			QuarantineLevelWeak:   0,
			QuarantineLevelStrong: 0,
			QuarantineLevelHard:   0,
		},
	}
	if s == nil {
		return stats
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, record := range s.records {
		stats.TotalRecords++
		stats.TotalVotes += record.GarbageVotes
		stats.Levels[record.QuarantineLevel]++
		if record.QuarantineLevel != QuarantineLevelNone {
			stats.QuarantinedRecords++
		}
		if record.LastScoredAt.After(stats.LastScoredAt) {
			stats.LastScoredAt = record.LastScoredAt
		}
	}
	return stats
}

func (s *HygieneStore) Upsert(record HygieneRecord) error {
	if s == nil {
		return nil
	}
	normalized, ok := normalizeHygieneRecord(record)
	if !ok {
		return fmt.Errorf("invalid hygiene record")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[normalized.Key] = normalized
	return s.persistLocked()
}

func (s *HygieneStore) mutate(target HygieneTarget, apply func(current HygieneRecord, exists bool) (HygieneRecord, error)) error {
	if s == nil {
		return nil
	}
	resolvedTarget := normalizeHygieneTarget(target)
	key, ok := resolvedTarget.Key()
	if !ok {
		return fmt.Errorf("invalid hygiene target")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, exists := s.records[key]
	next, err := apply(current, exists)
	if err != nil {
		return err
	}
	normalized, valid := normalizeHygieneRecord(next)
	if !valid {
		return fmt.Errorf("invalid hygiene record")
	}
	s.records[normalized.Key] = normalized
	return s.persistLocked()
}

func (s *HygieneStore) loadLocked() error {
	s.records = make(map[string]HygieneRecord)
	if s.baseDir == "" || s.recordsPath == "" {
		return nil
	}
	records, err := readHygieneSnapshot(s.recordsPath)
	if err != nil {
		return err
	}
	for _, record := range records {
		normalized, ok := normalizeHygieneRecord(record)
		if !ok {
			continue
		}
		s.records[normalized.Key] = normalized
	}
	return nil
}

func (s *HygieneStore) persistLocked() error {
	if s.baseDir == "" || s.recordsPath == "" {
		return nil
	}
	records := make([]HygieneRecord, 0, len(s.records))
	for _, record := range s.records {
		records = append(records, record)
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].Key < records[j].Key
	})
	return writeHygieneSnapshot(s.recordsPath, records)
}

func readHygieneSnapshot(path string) ([]HygieneRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read hygiene snapshot: %w", err)
	}
	var payload hygieneRecordsSnapshot
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode hygiene snapshot: %w", err)
	}
	return payload.Records, nil
}

func writeHygieneSnapshot(path string, records []HygieneRecord) error {
	data, err := json.MarshalIndent(hygieneRecordsSnapshot{Records: records}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal hygiene snapshot: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create hygiene dir: %w", err)
	}
	tmpPath := fmt.Sprintf("%s.tmp-%d", path, time.Now().UTC().UnixNano())
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write hygiene temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace hygiene snapshot: %w", err)
	}
	return nil
}
