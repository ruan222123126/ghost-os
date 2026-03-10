package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrSessionNotFound  = errors.New("session not found")
	ErrSessionCorrupted = errors.New("session corrupted")
	ErrInvalidSessionID = errors.New("invalid session id")
)

var sessionIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)

// Store 提供会话文件持久化能力。
type Store struct {
	baseDir string
	mu      sync.Mutex
}

// SessionMetadata 表示列表场景需要的轻量会话信息。
type SessionMetadata struct {
	ID           string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	MessageCount int
	TokenCount   int
}

// NewStore 初始化存储目录，并返回文件会话存储实例。
func NewStore(baseDir string) (*Store, error) {
	resolved, err := resolveBaseDir(baseDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(resolved, 0o700); err != nil {
		return nil, fmt.Errorf("create session directory %q: %w", resolved, err)
	}

	return &Store{baseDir: resolved}, nil
}

// Load 从磁盘读取会话。
func (s *Store) Load(sessionID string) (*Session, error) {
	path, err := s.pathForSession(sessionID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, strings.TrimSpace(sessionID))
		}
		return nil, fmt.Errorf("read session %q: %w", strings.TrimSpace(sessionID), err)
	}

	var loaded Session
	if err := json.Unmarshal(data, &loaded); err != nil {
		return nil, fmt.Errorf("%w: id=%s: %v", ErrSessionCorrupted, strings.TrimSpace(sessionID), err)
	}

	if loaded.ID == "" {
		loaded.ID = strings.TrimSpace(sessionID)
	}
	if strings.TrimSpace(loaded.ID) != strings.TrimSpace(sessionID) {
		return nil, fmt.Errorf("%w: id mismatch file=%q payload=%q", ErrSessionCorrupted, strings.TrimSpace(sessionID), loaded.ID)
	}
	if loaded.CreatedAt.IsZero() {
		loaded.CreatedAt = time.Now().UTC()
	}
	if loaded.UpdatedAt.IsZero() {
		loaded.UpdatedAt = loaded.CreatedAt
	}
	loaded.RecalculateTokenCount()

	return &loaded, nil
}

// Save 将会话原子写入磁盘，防止部分写入导致文件损坏。
func (s *Store) Save(session *Session) error {
	if session == nil {
		return errors.New("session is nil")
	}

	path, err := s.pathForSession(session.ID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	session.UpdatedAt = now
	session.RecalculateTokenCount()

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session %q: %w", session.ID, err)
	}
	data = append(data, '\n')

	tempPath := fmt.Sprintf("%s.tmp-%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write temp session file %q: %w", tempPath, err)
	}

	if err := replaceFileAtomic(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace session file %q: %w", path, err)
	}
	return nil
}

// Delete 删除指定会话文件。
func (s *Store) Delete(sessionID string) error {
	path, err := s.pathForSession(sessionID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrSessionNotFound, strings.TrimSpace(sessionID))
		}
		return fmt.Errorf("delete session %q: %w", strings.TrimSpace(sessionID), err)
	}
	return nil
}

// List 返回当前存储目录下的全部会话 ID。
func (s *Store) List() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("read sessions directory %q: %w", s.baseDir, err)
	}

	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		if !isValidSessionID(id) {
			continue
		}
		ids = append(ids, id)
	}

	sort.Strings(ids)
	return ids, nil
}

// ListMetadata 返回全部会话的轻量元数据，避免上层逐个 Load 组装。
func (s *Store) ListMetadata() ([]SessionMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("read sessions directory %q: %w", s.baseDir, err)
	}

	now := time.Now().UTC()
	metadata := make([]SessionMetadata, 0, len(entries))
	skipped := 0
	var firstSkippedID string
	var firstSkippedErr error
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}

		id := strings.TrimSuffix(name, ".json")
		if !isValidSessionID(id) {
			continue
		}

		path := filepath.Join(s.baseDir, name)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) {
				continue
			}
			skipped++
			if firstSkippedErr == nil {
				firstSkippedID = id
				firstSkippedErr = readErr
			}
			continue
		}

		summary, decodeErr := decodeSessionMetadata(id, data, now)
		if decodeErr != nil {
			skipped++
			if firstSkippedErr == nil {
				firstSkippedID = id
				firstSkippedErr = decodeErr
			}
			continue
		}
		metadata = append(metadata, summary)
	}

	sort.Slice(metadata, func(i, j int) bool {
		return metadata[i].ID < metadata[j].ID
	})

	if skipped > 0 {
		if firstSkippedErr != nil {
			log.Printf("[SESSION] ListMetadata skipped invalid session files: skipped=%d example_id=%s err=%v", skipped, firstSkippedID, firstSkippedErr)
		} else {
			log.Printf("[SESSION] ListMetadata skipped invalid session files: skipped=%d", skipped)
		}
	}
	return metadata, nil
}

type sessionMetadataEnvelope struct {
	ID         string     `json:"id"`
	Messages   []struct{} `json:"messages"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	TokenCount int        `json:"token_count"`
}

func decodeSessionMetadata(expectedID string, data []byte, now time.Time) (SessionMetadata, error) {
	var envelope sessionMetadataEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return SessionMetadata{}, fmt.Errorf("%w: id=%s: %v", ErrSessionCorrupted, expectedID, err)
	}

	loadedID := strings.TrimSpace(envelope.ID)
	if loadedID == "" {
		loadedID = expectedID
	}
	if loadedID != expectedID {
		return SessionMetadata{}, fmt.Errorf("%w: id mismatch file=%q payload=%q", ErrSessionCorrupted, expectedID, envelope.ID)
	}

	createdAt := envelope.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = now
	}
	updatedAt := envelope.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	return SessionMetadata{
		ID:           expectedID,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		MessageCount: len(envelope.Messages),
		TokenCount:   envelope.TokenCount,
	}, nil
}

func (s *Store) pathForSession(sessionID string) (string, error) {
	id := strings.TrimSpace(sessionID)
	if !isValidSessionID(id) {
		return "", fmt.Errorf("%w: %q", ErrInvalidSessionID, sessionID)
	}
	return filepath.Join(s.baseDir, id+".json"), nil
}

func resolveBaseDir(pathValue string) (string, error) {
	trimmed := strings.TrimSpace(pathValue)
	if trimmed == "" {
		return "", errors.New("sessions path is empty")
	}

	if strings.HasPrefix(trimmed, "~/") || trimmed == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home directory: %w", err)
		}

		if trimmed == "~" {
			return homeDir, nil
		}
		return filepath.Join(homeDir, strings.TrimPrefix(trimmed, "~/")), nil
	}

	return filepath.Clean(trimmed), nil
}

func isValidSessionID(sessionID string) bool {
	return sessionIDPattern.MatchString(strings.TrimSpace(sessionID))
}

func replaceFileAtomic(tempPath, path string) error {
	if err := os.Rename(tempPath, path); err != nil {
		if runtime.GOOS != "windows" {
			return err
		}
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("remove existing session file %q: %w", path, removeErr)
		}
		if renameErr := os.Rename(tempPath, path); renameErr != nil {
			return renameErr
		}
	}
	return nil
}
