package artifacts

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStoredPathRejectsOutsideBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewSessionArtifactStore(baseDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("nope"), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	if _, err := store.ResolveStoredPath(outsideFile); !errors.Is(err, ErrInvalidStoredPath) {
		t.Fatalf("expected invalid stored path, got %v", err)
	}
}

func TestResolveStoredPathRejectsSymlinkEscape(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewSessionArtifactStore(baseDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("nope"), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	symlinkPath := filepath.Join(baseDir, "escape.txt")
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	if _, err := store.ResolveStoredPath(symlinkPath); !errors.Is(err, ErrInvalidStoredPath) {
		t.Fatalf("expected invalid stored path, got %v", err)
	}
}

func TestOpenStoredFileReturnsFileAndInfo(t *testing.T) {
	baseDir := t.TempDir()
	store, err := NewSessionArtifactStore(baseDir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sessionDir, err := store.SessionDir("session-1")
	if err != nil {
		t.Fatalf("session dir: %v", err)
	}
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatalf("mkdir session dir: %v", err)
	}

	storedFile := filepath.Join(sessionDir, "artifact-1.txt")
	if err := os.WriteFile(storedFile, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write stored file: %v", err)
	}

	if err := store.WriteMetadata(SessionFileArtifact{
		ArtifactID:  "artifact-1",
		SessionID:   "session-1",
		Name:        "report.txt",
		DownloadURL: "/api/sessions/session-1/artifacts/artifact-1",
		StoredPath:  storedFile,
	}); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	file, info, artifact, err := store.OpenStoredFile("session-1", "artifact-1")
	if err != nil {
		t.Fatalf("open stored file: %v", err)
	}
	defer file.Close()

	if artifact == nil || artifact.ArtifactID != "artifact-1" {
		t.Fatalf("unexpected artifact metadata: %+v", artifact)
	}
	if info == nil || info.Size() != int64(len("hello")) {
		t.Fatalf("unexpected file info: %+v", info)
	}
}
