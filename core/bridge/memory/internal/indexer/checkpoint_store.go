package indexer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FileCheckpointStore struct {
	baseDir string
}

func NewFileCheckpointStore(baseDir string) *FileCheckpointStore {
	trimmed := strings.TrimSpace(baseDir)
	if trimmed == "" {
		return nil
	}
	return &FileCheckpointStore{baseDir: filepath.Clean(trimmed)}
}

func (s *FileCheckpointStore) Load(projector string, bucket Bucket) (Checkpoint, error) {
	if s == nil || s.baseDir == "" {
		return Checkpoint{}, nil
	}
	path := s.path(projector, bucket)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Checkpoint{}, nil
		}
		return Checkpoint{}, fmt.Errorf("read index checkpoint: %w", err)
	}
	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return Checkpoint{}, fmt.Errorf("decode index checkpoint: %w", err)
	}
	return normalizeCheckpoint(checkpoint), nil
}

func (s *FileCheckpointStore) Save(projector string, bucket Bucket, checkpoint Checkpoint) error {
	if s == nil || s.baseDir == "" {
		return nil
	}
	path := s.path(projector, bucket)
	checkpoint = normalizeCheckpoint(checkpoint)
	checkpoint.UpdatedAt = time.Now().UTC()
	if checkpoint.Segment == "" {
		checkpoint.Segment = normalizeBucket(bucket).SegmentPath
	}
	data, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("encode index checkpoint: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create index checkpoint dir: %w", err)
	}
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write index checkpoint temp: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("rename index checkpoint temp: %w", err)
	}
	return nil
}

func (s *FileCheckpointStore) path(projector string, bucket Bucket) string {
	bucket = normalizeBucket(bucket)
	workspace := bucket.Workspace
	if workspace == "" {
		workspace = "_"
	}
	return filepath.Join(
		s.baseDir,
		strings.TrimSpace(projector),
		bucket.Namespace,
		workspace,
		bucket.Month,
		checkpointFileName,
	)
}
