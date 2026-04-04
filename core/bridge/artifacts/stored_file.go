package artifacts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrInvalidStoredPath = errors.New("invalid stored path")

func (s *SessionArtifactStore) ResolveStoredPath(storedPath string) (string, error) {
	if s == nil {
		return "", errors.New("artifact store is not configured")
	}

	trimmed := strings.TrimSpace(storedPath)
	if trimmed == "" {
		return "", ErrInvalidStoredPath
	}

	cleanBase := filepath.Clean(s.baseDir)
	cleanStored := filepath.Clean(trimmed)
	if !filepath.IsAbs(cleanStored) {
		return "", ErrInvalidStoredPath
	}
	if !isWithinBaseDir(cleanBase, cleanStored) {
		return "", ErrInvalidStoredPath
	}

	resolvedBase := cleanBase
	if resolved, err := filepath.EvalSymlinks(cleanBase); err == nil {
		resolvedBase = filepath.Clean(resolved)
	}

	resolvedStored, err := filepath.EvalSymlinks(cleanStored)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("resolve stored path: %w", err)
		}
		return "", fmt.Errorf("%w: resolve symlinks: %v", ErrInvalidStoredPath, err)
	}
	resolvedStored = filepath.Clean(resolvedStored)
	if !filepath.IsAbs(resolvedStored) || !isWithinBaseDir(resolvedBase, resolvedStored) {
		return "", ErrInvalidStoredPath
	}
	return resolvedStored, nil
}

func (s *SessionArtifactStore) OpenStoredFile(sessionID string, artifactID string) (*os.File, os.FileInfo, *SessionFileArtifact, error) {
	if s == nil {
		return nil, nil, nil, errors.New("artifact store is not configured")
	}

	artifact, err := s.Load(sessionID, artifactID)
	if err != nil {
		return nil, nil, nil, err
	}

	path, err := s.ResolveStoredPath(artifact.StoredPath)
	if err != nil {
		return nil, nil, nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("open artifact: %w", err)
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, nil, fmt.Errorf("stat artifact: %w", err)
	}
	return file, info, artifact, nil
}

func isWithinBaseDir(baseDir string, target string) bool {
	rel, err := filepath.Rel(baseDir, target)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	if rel == "." || rel == ".." {
		return false
	}
	prefix := ".." + string(filepath.Separator)
	return !strings.HasPrefix(rel, prefix)
}
