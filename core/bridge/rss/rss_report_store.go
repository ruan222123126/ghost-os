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

var ErrRSSReportNotFound = errors.New("rss report not found")

const defaultRSSReportRetention = 60

type RSSReportStore struct {
	path string
	now  func() time.Time
	mu   sync.Mutex
}

type rssReportSnapshot struct {
	Reports []RSSReportResult `json:"reports,omitempty"`
}

func NewRSSReportStore(path string) (*RSSReportStore, error) {
	resolved, err := resolveRSSReportPath(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o700); err != nil {
		return nil, fmt.Errorf("create rss report directory %q: %w", filepath.Dir(resolved), err)
	}
	return &RSSReportStore{path: resolved, now: time.Now}, nil
}

func (s *RSSReportStore) Save(report RSSReportResult, markdown string) (RSSReportResult, error) {
	if s == nil {
		return RSSReportResult{}, errors.New("rss report store is nil")
	}
	if strings.TrimSpace(markdown) == "" {
		return RSSReportResult{}, fmt.Errorf("rss report markdown is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, err := s.loadLocked()
	if err != nil {
		return RSSReportResult{}, err
	}
	normalized, err := normalizeRSSReportResult(report, s.currentTime().UTC(), filepath.Dir(s.path))
	if err != nil {
		return RSSReportResult{}, err
	}
	if err := writeRSSReportMarkdown(normalized.MarkdownPath, markdown); err != nil {
		return RSSReportResult{}, err
	}

	replaced := false
	for i := range snapshot.Reports {
		if snapshot.Reports[i].ID != normalized.ID {
			continue
		}
		snapshot.Reports[i] = normalized
		replaced = true
		break
	}
	if !replaced {
		snapshot.Reports = append(snapshot.Reports, normalized)
	}
	sortRSSReports(snapshot.Reports)
	if len(snapshot.Reports) > defaultRSSReportRetention {
		snapshot.Reports = snapshot.Reports[:defaultRSSReportRetention]
	}
	if err := s.saveLocked(snapshot); err != nil {
		return RSSReportResult{}, err
	}
	return normalized, nil
}

func (s *RSSReportStore) Latest() (RSSReportResult, error) {
	if s == nil {
		return RSSReportResult{}, errors.New("rss report store is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snapshot, err := s.loadLocked()
	if err != nil {
		return RSSReportResult{}, err
	}
	if len(snapshot.Reports) == 0 {
		return RSSReportResult{}, ErrRSSReportNotFound
	}
	sortRSSReports(snapshot.Reports)
	return cloneRSSReportResult(snapshot.Reports[0]), nil
}

func (s *RSSReportStore) loadLocked() (rssReportSnapshot, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return rssReportSnapshot{}, nil
		}
		return rssReportSnapshot{}, fmt.Errorf("read rss report snapshot %q: %w", s.path, err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return rssReportSnapshot{}, nil
	}
	var snapshot rssReportSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return rssReportSnapshot{}, fmt.Errorf("decode rss report snapshot %q: %w", s.path, err)
	}
	return snapshot, nil
}

func (s *RSSReportStore) saveLocked(snapshot rssReportSnapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode rss report snapshot %q: %w", s.path, err)
	}
	data = append(data, '\n')
	tempPath := fmt.Sprintf("%s.tmp-%d", s.path, time.Now().UnixNano())
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write temp rss report snapshot %q: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, s.path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace rss report snapshot %q: %w", s.path, err)
	}
	return nil
}

func (s *RSSReportStore) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *RSSReportStore) RootDir() string {
	if s == nil {
		return ""
	}
	return filepath.Dir(s.path)
}

func resolveRSSReportPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		trimmed = defaultRSSReportsPath
	}
	resolved, err := resolveUserPath(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve rss report path: %w", err)
	}
	return resolved, nil
}

func normalizeRSSReportResult(report RSSReportResult, fallbackSavedAt time.Time, rootDir string) (RSSReportResult, error) {
	report.ID = strings.TrimSpace(report.ID)
	report.BriefingID = strings.TrimSpace(report.BriefingID)
	report.Title = strings.TrimSpace(report.Title)
	report.Summary = strings.TrimSpace(report.Summary)
	report.TraceID = strings.TrimSpace(report.TraceID)
	report.TaskID = strings.TrimSpace(report.TaskID)
	report.SourceGroupIDs = append([]string(nil), report.SourceGroupIDs...)
	if report.GeneratedAt.IsZero() {
		report.GeneratedAt = fallbackSavedAt
	}
	report.GeneratedAt = report.GeneratedAt.UTC()
	if report.SavedAt.IsZero() {
		report.SavedAt = fallbackSavedAt
	}
	report.SavedAt = report.SavedAt.UTC()
	if report.ID == "" {
		report.ID = newRSSReportID(report)
	}
	if report.Title == "" {
		report.Title = "RSS Report"
	}
	var err error
	report.MarkdownPath, err = normalizeRSSReportMarkdownPath(report, rootDir)
	if err != nil {
		return RSSReportResult{}, err
	}
	report.HighlightCount = firstPositive(report.HighlightCount)
	report.GroupCount = firstPositive(report.GroupCount)
	return report, nil
}

func normalizeRSSReportMarkdownPath(report RSSReportResult, rootDir string) (string, error) {
	path := strings.TrimSpace(report.MarkdownPath)
	if path != "" {
		return resolveUserPath(path)
	}
	monthDir := report.GeneratedAt.UTC().Format("2006/01")
	return filepath.Join(rootDir, filepath.FromSlash(monthDir), report.ID+".md"), nil
}

func writeRSSReportMarkdown(path string, markdown string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create rss report markdown directory %q: %w", filepath.Dir(path), err)
	}
	body := strings.TrimSpace(markdown) + "\n"
	tempPath := fmt.Sprintf("%s.tmp-%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tempPath, []byte(body), 0o600); err != nil {
		return fmt.Errorf("write temp rss report markdown %q: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace rss report markdown %q: %w", path, err)
	}
	return nil
}

func newRSSReportID(report RSSReportResult) string {
	payload := []string{
		report.GeneratedAt.UTC().Format(time.RFC3339Nano),
		strings.TrimSpace(report.BriefingID),
		strings.TrimSpace(report.Title),
		strings.TrimSpace(report.TraceID),
	}
	hash := sha256.Sum256([]byte(strings.Join(payload, "\n")))
	return "rssr_" + hex.EncodeToString(hash[:12])
}

func sortRSSReports(items []RSSReportResult) {
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

func cloneRSSReportResult(report RSSReportResult) RSSReportResult {
	report.SourceGroupIDs = append([]string(nil), report.SourceGroupIDs...)
	return report
}
