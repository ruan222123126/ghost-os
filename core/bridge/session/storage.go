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
	baseDir  string
	db       *sql.DB
	humanLog *sessionHumanLogWriter
	mu       sync.Mutex
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
func NewStore(baseDir string, options ...StoreOptions) (*Store, error) {
	resolved, err := resolveBaseDir(baseDir)
	if err != nil {
		return nil, err
	}
	resolvedOptions, err := resolveStoreOptions(options)
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
	humanLog, err := newSessionHumanLogWriter(resolved, resolvedOptions)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{
		baseDir:  resolved,
		db:       db,
		humanLog: humanLog,
	}, nil
}

// Load 从磁盘读取会话轻量状态与热窗口消息。
func (s *Store) Load(sessionID string) (*Session, error) {
	id, err := normalizeSessionID(sessionID)
	if err != nil {
		return nil, err
	}

	var sess *Session
	if err := s.withSessionLock(id, false, func() error {
		return s.withTx("load session", func(tx *sql.Tx) error {
			loaded, loadErr := loadSessionTx(tx, id)
			if loadErr != nil {
				return loadErr
			}
			sess = loaded
			return nil
		})
	}); err != nil {
		return nil, err
	}
	return sess, nil
}

// LoadPage 读取单会话详情和指定窗口的消息分页。
func (s *Store) LoadPage(sessionID string, params PageParams) (*Session, MessagePage, error) {
	id, err := normalizeSessionID(sessionID)
	if err != nil {
		return nil, MessagePage{}, err
	}

	var (
		sess *Session
		page MessagePage
	)
	if err := s.withSessionLock(id, false, func() error {
		return s.withTx("load session page", func(tx *sql.Tx) error {
			loaded, loadErr := loadSessionTx(tx, id)
			if loadErr != nil {
				return loadErr
			}
			loadedPage, pageErr := loadMessagePageTx(tx, id, loaded.MessageCount, params)
			if pageErr != nil {
				return pageErr
			}
			sess = loaded
			page = loadedPage
			return nil
		})
	}); err != nil {
		return nil, MessagePage{}, err
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

	return s.withSessionLock(id, true, func() error {
		if err := s.withTx("save session", func(tx *sql.Tx) error {
			return saveSessionTx(tx, id, sess)
		}); err != nil {
			return err
		}
		return s.syncSessionHumanLogLocked(id)
	})
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

	return s.withStoreLock(func() error {
		deleted := false
		if err := s.withTx("delete session", func(tx *sql.Tx) error {
			removed, deleteErr := deleteSessionTx(tx, id)
			if deleteErr != nil {
				return deleteErr
			}
			deleted = removed
			return nil
		}); err != nil {
			return err
		}
		if deleted {
			if err := removeLegacySessionFile(s, id); err != nil {
				return err
			}
			if err := removeSessionHumanLogFile(s, id); err != nil {
				return err
			}
			return s.cleanupSidebarPartitionStateLocked(id)
		}

		legacyRemoved, deleteLegacyErr := deleteLegacySessionFile(s, id)
		if deleteLegacyErr != nil {
			return deleteLegacyErr
		}
		humanLogRemoved, deleteHumanLogErr := deleteSessionHumanLogFile(s, id)
		if deleteHumanLogErr != nil {
			return deleteHumanLogErr
		}
		if legacyRemoved || humanLogRemoved {
			return s.cleanupSidebarPartitionStateLocked(id)
		}
		return fmt.Errorf("%w: %s", ErrSessionNotFound, id)
	})
}
