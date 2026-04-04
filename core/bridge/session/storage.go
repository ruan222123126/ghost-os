package session

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ghost-os/bridge/llm"

	_ "modernc.org/sqlite"
)

const (
	DefaultDetailPageLimit = 100
	MaxDetailPageLimit     = 200
	hotWindowMaxMessages   = 200
	hotWindowMaxTokens     = 24_000
)

var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionCorrupted     = errors.New("session corrupted")
	ErrInvalidSessionID     = errors.New("invalid session id")
	ErrSessionNotAppendOnly = errors.New("session history mutation must be append-only")
)

// Store 提供会话状态与消息历史的持久化能力。
type Store struct {
	baseDir string
	db      *sql.DB
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

// PageParams 描述一次会话历史分页读取请求。
type PageParams struct {
	Limit  int
	Before *int
}

// IndexedMessage 表示带稳定绝对序号的一条会话消息。
type IndexedMessage struct {
	Index   int
	Message llm.Message
}

// MessagePage 描述一次分页读取结果。
type MessagePage struct {
	Messages      []IndexedMessage
	Limit         int
	Before        *int
	StartIndex    *int
	EndIndex      *int
	HasMoreBefore bool
	NextBefore    *int
}

// NewStore 初始化 SQLite 会话存储。
func NewStore(baseDir string) (*Store, error) {
	resolved, err := resolveBaseDir(baseDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(resolved, 0o700); err != nil {
		return nil, fmt.Errorf("create session directory %q: %w", resolved, err)
	}

	db, err := sql.Open("sqlite", filepath.Join(resolved, sessionsDatabaseFilename))
	if err != nil {
		return nil, fmt.Errorf("open session database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := initSessionSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{
		baseDir: resolved,
		db:      db,
	}, nil
}

// Load 从磁盘读取会话轻量状态与热窗口消息。
func (s *Store) Load(sessionID string) (*Session, error) {
	id, err := normalizeSessionID(sessionID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.importLegacySessionLocked(id); err != nil {
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin load session: %w", err)
	}
	defer rollbackTx(tx)

	sess, err := loadSessionTx(tx, id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit load session: %w", err)
	}
	return sess, nil
}

// LoadPage 读取单会话详情和指定窗口的消息分页。
func (s *Store) LoadPage(sessionID string, params PageParams) (*Session, MessagePage, error) {
	id, err := normalizeSessionID(sessionID)
	if err != nil {
		return nil, MessagePage{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.importLegacySessionLocked(id); err != nil {
		return nil, MessagePage{}, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, MessagePage{}, fmt.Errorf("begin load session page: %w", err)
	}
	defer rollbackTx(tx)

	sess, err := loadSessionTx(tx, id)
	if err != nil {
		return nil, MessagePage{}, err
	}
	page, err := loadMessagePageTx(tx, id, sess.MessageCount, params)
	if err != nil {
		return nil, MessagePage{}, err
	}
	if err := tx.Commit(); err != nil {
		return nil, MessagePage{}, fmt.Errorf("commit load session page: %w", err)
	}
	return sess, page, nil
}

// Save 将会话状态更新写入磁盘，消息历史只允许尾部追加。
func (s *Store) Save(sess *Session) error {
	if sess == nil {
		return errors.New("session is nil")
	}
	id, err := normalizeSessionID(sess.ID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.importLegacySessionLocked(id); err != nil && !errors.Is(err, ErrSessionNotFound) {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin save session: %w", err)
	}
	defer rollbackTx(tx)
	if err := saveSessionTx(tx, id, sess); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit save session: %w", err)
	}
	return nil
}

func saveSessionTx(tx *sql.Tx, sessionID string, sess *Session) error {
	record, err := loadSessionRecordTx(tx, sessionID)
	if err != nil && !errors.Is(err, ErrSessionNotFound) {
		return err
	}
	appended, appendTokens, err := appendedMessages(sess, record)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = now
	}
	sess.UpdatedAt = now
	if err := upsertSessionTx(tx, sess, record, appended, appendTokens); err != nil {
		return err
	}

	reloaded, err := loadSessionTx(tx, sessionID)
	if err != nil {
		return err
	}
	replaceLoadedSession(sess, reloaded)
	return nil
}

// Delete 删除指定会话。
func (s *Store) Delete(sessionID string) error {
	id, err := normalizeSessionID(sessionID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin delete session: %w", err)
	}
	defer rollbackTx(tx)

	deleted, err := deleteSessionTx(tx, id)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete session: %w", err)
	}
	if deleted {
		_ = removeLegacySessionFile(s, id)
		return nil
	}
	if removed, err := deleteLegacySessionFile(s, id); err != nil {
		return err
	} else if removed {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrSessionNotFound, id)
}
