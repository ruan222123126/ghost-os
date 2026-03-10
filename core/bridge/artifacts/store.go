package artifacts

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultArtifactsPath = "~/.ghost-os/artifacts"
	sessionArtifactsDir  = "sessions"
)

// MaxIdentifierLength caps session/artifact identifiers used as path components.
// Keep this conservative so generated metadata filenames remain cross-platform safe.
const MaxIdentifierLength = 128

var ErrArtifactNotFound = errors.New("artifact not found")

type SessionFileArtifact struct {
	ArtifactID  string `json:"artifact_id"`
	SessionID   string `json:"session_id"`
	Name        string `json:"name"`
	MimeType    string `json:"mime_type,omitempty"`
	Bytes       int64  `json:"bytes,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	DownloadURL string `json:"download_url"`
	SourcePath  string `json:"source_path,omitempty"`
	StoredPath  string `json:"stored_path"`
	Note        string `json:"note,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

type SessionArtifactStore struct {
	baseDir string
}

func NewSessionArtifactStore(baseDir string) (*SessionArtifactStore, error) {
	resolved, err := ResolveArtifactsDir(baseDir)
	if err != nil {
		return nil, err
	}
	return &SessionArtifactStore{baseDir: resolved}, nil
}

func NewSessionArtifactStoreFromEnv() (*SessionArtifactStore, error) {
	return NewSessionArtifactStore(strings.TrimSpace(os.Getenv("GHOST_ARTIFACTS_PATH")))
}

func ResolveArtifactsDir(baseDir string) (string, error) {
	path := strings.TrimSpace(baseDir)
	if path == "" {
		path = defaultArtifactsPath
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home for artifact directory: %w", err)
		}
		if path == "~" {
			path = homeDir
		} else {
			path = filepath.Join(homeDir, strings.TrimPrefix(path, "~/"))
		}
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve artifact directory: %w", err)
	}
	return filepath.Clean(absolute), nil
}

func (s *SessionArtifactStore) BaseDir() string {
	if s == nil {
		return ""
	}
	return s.baseDir
}

func (s *SessionArtifactStore) SessionDir(sessionID string) (string, error) {
	if s == nil {
		return "", errors.New("artifact store is not configured")
	}
	normalized, err := NormalizeSessionID(sessionID)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.baseDir, sessionArtifactsDir, normalized), nil
}

func (s *SessionArtifactStore) WriteMetadata(artifact SessionFileArtifact) error {
	if s == nil {
		return errors.New("artifact store is not configured")
	}
	sessionID, err := NormalizeSessionID(artifact.SessionID)
	if err != nil {
		return err
	}
	artifactID, err := NormalizeArtifactID(artifact.ArtifactID)
	if err != nil {
		return err
	}

	artifact.SessionID = sessionID
	artifact.ArtifactID = artifactID
	artifact.Name = strings.TrimSpace(artifact.Name)
	artifact.DownloadURL = strings.TrimSpace(artifact.DownloadURL)
	artifact.StoredPath = strings.TrimSpace(artifact.StoredPath)
	artifact.SourcePath = strings.TrimSpace(artifact.SourcePath)
	artifact.Note = strings.TrimSpace(artifact.Note)
	artifact.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	if artifact.Name == "" || artifact.DownloadURL == "" || artifact.StoredPath == "" {
		return errors.New("artifact metadata is incomplete")
	}

	dir := filepath.Join(s.baseDir, sessionArtifactsDir, sessionID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create artifact directory: %w", err)
	}

	encoded, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return fmt.Errorf("encode artifact metadata: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, artifactID+".json"), encoded, 0o600); err != nil {
		return fmt.Errorf("write artifact metadata: %w", err)
	}
	return nil
}

func (s *SessionArtifactStore) Load(sessionID string, artifactID string) (*SessionFileArtifact, error) {
	if s == nil {
		return nil, errors.New("artifact store is not configured")
	}
	normalizedSessionID, err := NormalizeSessionID(sessionID)
	if err != nil {
		return nil, err
	}
	normalizedArtifactID, err := NormalizeArtifactID(artifactID)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(s.baseDir, sessionArtifactsDir, normalizedSessionID, normalizedArtifactID+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrArtifactNotFound
		}
		return nil, fmt.Errorf("read artifact metadata: %w", err)
	}

	var artifact SessionFileArtifact
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return nil, fmt.Errorf("decode artifact metadata: %w", err)
	}
	return &artifact, nil
}

func NormalizeSessionID(sessionID string) (string, error) {
	return normalizeIdentifier(sessionID, "session_id")
}

func NormalizeArtifactID(artifactID string) (string, error) {
	return normalizeIdentifier(artifactID, "artifact_id")
}

func normalizeIdentifier(value string, label string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	if len(trimmed) > MaxIdentifierLength {
		return "", fmt.Errorf("%s is too long", label)
	}
	if trimmed == "." || trimmed == ".." {
		return "", fmt.Errorf("invalid %s", label)
	}
	for _, ch := range trimmed {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch >= '0' && ch <= '9':
		case ch == '-' || ch == '_':
		default:
			return "", fmt.Errorf("invalid %s", label)
		}
	}
	// Keep legacy separator checks as an extra guardrail.
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, string(filepath.Separator)) {
		return "", fmt.Errorf("invalid %s", label)
	}
	return trimmed, nil
}
