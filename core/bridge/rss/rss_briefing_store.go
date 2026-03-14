package rss

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrRSSBriefingNotFound = errors.New("rss briefing not found")

const defaultRSSBriefingRetention = 60

type RSSBriefingStore struct {
	path string
	now  func() time.Time
	mu   sync.Mutex
}

type rssBriefingSnapshot struct {
	Briefings []RSSBriefingResult `json:"briefings,omitempty"`
}

func NewRSSBriefingStore(path string) (*RSSBriefingStore, error) {
	resolved, err := resolveRSSBriefingPath(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o700); err != nil {
		return nil, fmt.Errorf("create rss briefing directory %q: %w", filepath.Dir(resolved), err)
	}
	return &RSSBriefingStore{path: resolved, now: time.Now}, nil
}

func (s *RSSBriefingStore) Save(briefing RSSBriefingResult) (RSSBriefingResult, error) {
	if s == nil {
		return RSSBriefingResult{}, errors.New("rss briefing store is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, err := s.loadLocked()
	if err != nil {
		return RSSBriefingResult{}, err
	}
	normalized := normalizeRSSBriefingResult(briefing, s.currentTime().UTC())
	replaced := false
	for i := range snapshot.Briefings {
		if snapshot.Briefings[i].ID != normalized.ID {
			continue
		}
		snapshot.Briefings[i] = normalized
		replaced = true
		break
	}
	if !replaced {
		snapshot.Briefings = append(snapshot.Briefings, normalized)
	}
	sortRSSBriefings(snapshot.Briefings)
	if len(snapshot.Briefings) > defaultRSSBriefingRetention {
		snapshot.Briefings = snapshot.Briefings[:defaultRSSBriefingRetention]
	}
	if err := s.saveLocked(snapshot); err != nil {
		return RSSBriefingResult{}, err
	}
	return normalized, nil
}

func (s *RSSBriefingStore) Latest() (RSSBriefingResult, error) {
	if s == nil {
		return RSSBriefingResult{}, errors.New("rss briefing store is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, err := s.loadLocked()
	if err != nil {
		return RSSBriefingResult{}, err
	}
	if len(snapshot.Briefings) == 0 {
		return RSSBriefingResult{}, ErrRSSBriefingNotFound
	}
	sortRSSBriefings(snapshot.Briefings)
	return cloneRSSBriefingResult(snapshot.Briefings[0]), nil
}

func (s *RSSBriefingStore) Get(id string) (RSSBriefingResult, error) {
	if s == nil {
		return RSSBriefingResult{}, errors.New("rss briefing store is nil")
	}
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return RSSBriefingResult{}, fmt.Errorf("rss briefing id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, err := s.loadLocked()
	if err != nil {
		return RSSBriefingResult{}, err
	}
	for _, briefing := range snapshot.Briefings {
		if briefing.ID == trimmedID {
			return cloneRSSBriefingResult(briefing), nil
		}
	}
	return RSSBriefingResult{}, fmt.Errorf("%w: %s", ErrRSSBriefingNotFound, trimmedID)
}

func (s *RSSBriefingStore) loadLocked() (rssBriefingSnapshot, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return rssBriefingSnapshot{}, nil
		}
		return rssBriefingSnapshot{}, fmt.Errorf("read rss briefing snapshot %q: %w", s.path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return rssBriefingSnapshot{}, nil
	}
	var snapshot rssBriefingSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return rssBriefingSnapshot{}, fmt.Errorf("decode rss briefing snapshot %q: %w", s.path, err)
	}
	return snapshot, nil
}

func (s *RSSBriefingStore) saveLocked(snapshot rssBriefingSnapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode rss briefing snapshot %q: %w", s.path, err)
	}
	data = append(data, '\n')
	tempPath := fmt.Sprintf("%s.tmp-%d", s.path, time.Now().UnixNano())
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write temp rss briefing snapshot %q: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace rss briefing snapshot %q: %w", s.path, err)
	}
	return nil
}

func (s *RSSBriefingStore) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func resolveRSSBriefingPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		trimmed = defaultRSSBriefingsPath
	}
	resolved, err := resolveUserPath(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve rss briefing path: %w", err)
	}
	return resolved, nil
}

func normalizeRSSBriefingResult(briefing RSSBriefingResult, fallbackSavedAt time.Time) RSSBriefingResult {
	briefing.ID = strings.TrimSpace(briefing.ID)
	briefing.Title = strings.TrimSpace(briefing.Title)
	briefing.Summary = strings.TrimSpace(briefing.Summary)
	briefing.TraceID = strings.TrimSpace(briefing.TraceID)
	briefing.TaskID = strings.TrimSpace(briefing.TaskID)
	briefing.ReportError = strings.TrimSpace(briefing.ReportError)
	if briefing.GeneratedAt.IsZero() {
		briefing.GeneratedAt = fallbackSavedAt
	}
	briefing.GeneratedAt = briefing.GeneratedAt.UTC()
	if briefing.SavedAt.IsZero() {
		briefing.SavedAt = fallbackSavedAt
	}
	briefing.SavedAt = briefing.SavedAt.UTC()
	if briefing.ID == "" {
		briefing.ID = newRSSBriefingID(briefing)
	}
	briefing.Highlights = cloneRSSBriefingHighlights(briefing.Highlights)
	if briefing.Report != nil {
		report := cloneRSSReportResult(*briefing.Report)
		briefing.Report = &report
	}
	briefing.HighlightCount = len(briefing.Highlights)
	return briefing
}

func newRSSBriefingID(briefing RSSBriefingResult) string {
	payload := []string{
		briefing.GeneratedAt.UTC().Format(time.RFC3339Nano),
		strings.TrimSpace(briefing.Title),
		strings.TrimSpace(briefing.TraceID),
	}
	hash := sha256.Sum256([]byte(strings.Join(payload, "\n")))
	return "rssb_" + hex.EncodeToString(hash[:12])
}

func sortRSSBriefings(items []RSSBriefingResult) {
	sort.SliceStable(items, func(i, j int) bool {
		if !items[i].SavedAt.Equal(items[j].SavedAt) {
			return items[i].SavedAt.After(items[j].SavedAt)
		}
		if !items[i].GeneratedAt.Equal(items[j].GeneratedAt) {
			return items[i].GeneratedAt.After(items[j].GeneratedAt)
		}
		return items[i].ID > items[j].ID
	})
}

func cloneRSSBriefingResult(briefing RSSBriefingResult) RSSBriefingResult {
	briefing.Highlights = cloneRSSBriefingHighlights(briefing.Highlights)
	if briefing.Report != nil {
		report := cloneRSSReportResult(*briefing.Report)
		briefing.Report = &report
	}
	return briefing
}

func cloneRSSBriefingHighlights(items []RSSBriefingHighlight) []RSSBriefingHighlight {
	if len(items) == 0 {
		return nil
	}
	out := make([]RSSBriefingHighlight, len(items))
	for i := range items {
		out[i] = items[i]
		out[i].Tags = append([]string(nil), items[i].Tags...)
	}
	return out
}
