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

// ColdMemory 是 L3 冷数据层，负责长期归档与按需检索。
type ColdMemory struct {
	baseDir string
	mu      sync.Mutex
}

func NewColdMemory(baseDir string) *ColdMemory {
	return &ColdMemory{
		baseDir: resolveMemoryPath(baseDir),
	}
}

// Archive 把会话完整消息落盘到按月目录。
func (c *ColdMemory) Archive(sessionID string, messages []llm.Message) error {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return fmt.Errorf("session id is required")
	}
	if c.baseDir == "" {
		return fmt.Errorf("cold memory base dir is empty")
	}

	now := time.Now().UTC()
	payload := coldArchiveFile{
		SessionID:  sid,
		ArchivedAt: now,
		Messages:   llm.CloneMessages(messages),
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	targetDir := filepath.Join(c.baseDir, now.Format("2006-01"))
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return fmt.Errorf("create cold memory directory: %w", err)
	}

	targetPath := filepath.Join(targetDir, coldFilePrefix+sid+coldFileSuffix)
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cold archive: %w", err)
	}
	data = append(data, '\n')

	tmpPath := fmt.Sprintf("%s.tmp-%d", targetPath, time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write cold archive temp file: %w", err)
	}
	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace cold archive file: %w", err)
	}
	return nil
}

func (c *ColdMemory) Store(entry MemoryEntry) error {
	return fmt.Errorf("cold memory does not support generic store, use Archive")
}

// Retrieve 按 query 过滤归档消息，返回统一条目结构。
func (c *ColdMemory) Retrieve(query MemoryQuery) ([]MemoryEntry, error) {
	if c.baseDir == "" {
		return nil, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	files, err := c.collectArchiveFilesLocked(query.TimeRange)
	if err != nil {
		return nil, err
	}

	results := make([]MemoryEntry, 0, 64)
	for _, path := range files {
		archive, err := c.readArchiveFileLocked(path)
		if err != nil {
			continue
		}
		if query.TimeRange != nil && !query.TimeRange.Contains(archive.ArchivedAt) {
			continue
		}
		for i, msg := range archive.Messages {
			entry := MemoryEntry{
				ID:        fmt.Sprintf("%s:%06d", archive.SessionID, i),
				Content:   messageToContent(msg),
				Type:      MemoryTypeMessage,
				Timestamp: archive.ArchivedAt,
				Metadata: map[string]any{
					"layer":        "cold",
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

func (c *ColdMemory) Delete(id string) error {
	sid := strings.TrimSpace(id)
	if sid == "" {
		return fmt.Errorf("session id is required")
	}
	if c.baseDir == "" {
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	removed := false
	monthDirs, err := os.ReadDir(c.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read cold memory directory: %w", err)
	}
	for _, monthDir := range monthDirs {
		if !monthDir.IsDir() {
			continue
		}
		target := filepath.Join(c.baseDir, monthDir.Name(), coldFilePrefix+sid+coldFileSuffix)
		err := os.Remove(target)
		if err == nil {
			removed = true
			continue
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("delete cold archive %q: %w", target, err)
		}
	}
	if !removed {
		return nil
	}
	return nil
}

func (c *ColdMemory) Clear() error {
	if c.baseDir == "" {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := os.RemoveAll(c.baseDir); err != nil {
		return fmt.Errorf("clear cold memory directory: %w", err)
	}
	return nil
}

// ListSessions 返回时间范围内存在归档的会话 ID。
func (c *ColdMemory) ListSessions(timeRange TimeRange) ([]string, error) {
	if c.baseDir == "" {
		return nil, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	files, err := c.collectArchiveFilesLocked(&timeRange)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(files))
	out := make([]string, 0, len(files))
	for _, path := range files {
		archive, err := c.readArchiveFileLocked(path)
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

func (c *ColdMemory) collectArchiveFilesLocked(timeRange *TimeRange) ([]string, error) {
	monthDirs, err := os.ReadDir(c.baseDir)
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

		entries, err := os.ReadDir(filepath.Join(c.baseDir, monthName))
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
			files = append(files, filepath.Join(c.baseDir, monthName, name))
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

// UpdateSummary 更新归档文件的摘要和实体标签。
func (c *ColdMemory) UpdateSummary(sessionID string, summary string, entities []string) error {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return fmt.Errorf("session id is required")
	}
	if c.baseDir == "" {
		return fmt.Errorf("cold memory base dir is empty")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 查找归档文件
	monthDirs, err := os.ReadDir(c.baseDir)
	if err != nil {
		return fmt.Errorf("read cold memory directory: %w", err)
	}

	for _, monthDir := range monthDirs {
		if !monthDir.IsDir() {
			continue
		}
		targetPath := filepath.Join(c.baseDir, monthDir.Name(), coldFilePrefix+sid+coldFileSuffix)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			continue
		}

		// 读取现有归档
		archive, err := c.readArchiveFileLocked(targetPath)
		if err != nil {
			return err
		}

		// 更新摘要和实体
		archive.Summary = strings.TrimSpace(summary)
		archive.Entities = entities

		// 写回文件
		data, err := json.MarshalIndent(archive, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal cold archive: %w", err)
		}
		data = append(data, '\n')

		tmpPath := fmt.Sprintf("%s.tmp-%d", targetPath, time.Now().UnixNano())
		if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
			return fmt.Errorf("write cold archive temp file: %w", err)
		}
		if err := os.Rename(tmpPath, targetPath); err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("replace cold archive file: %w", err)
		}
		return nil
	}

	return fmt.Errorf("session %s not found in cold storage", sid)
}

func (c *ColdMemory) readArchiveFileLocked(path string) (coldArchiveFile, error) {
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
	return archive, nil
}

func messageToContent(msg llm.Message) string {
	if text := strings.TrimSpace(msg.Text); text != "" {
		return text
	}
	if len(msg.ToolCalls) == 0 {
		return ""
	}
	encoded, err := json.Marshal(msg.ToolCalls)
	if err != nil {
		return ""
	}
	return string(encoded)
}
