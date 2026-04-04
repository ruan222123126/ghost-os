package guiagent

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ghost-os/bridge/artifacts"
)

const computerUseArtifactDir = "computer_use"

type ArtifactWriter struct {
	baseDir string
}

func NewArtifactWriter(
	store *artifacts.SessionArtifactStore,
	sessionID string,
	runID string,
) (*ArtifactWriter, error) {
	if store == nil {
		return nil, fmt.Errorf("artifact store is not configured")
	}
	sessionDir, err := store.SessionDir(sessionID)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Join(sessionDir, computerUseArtifactDir, sanitizePathComponent(runID, "run"))
	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, fmt.Errorf("create computer_use artifact directory: %w", err)
	}
	return &ArtifactWriter{baseDir: baseDir}, nil
}

func NewRunID() (string, error) {
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "gui-" + hex.EncodeToString(raw[:]), nil
}

func (w *ArtifactWriter) WritePNG(
	stepIndex int,
	phase string,
	traceID string,
	buf []byte,
	width int,
	height int,
) (*StoredArtifact, error) {
	return w.writeBytes(stepIndex, phase, traceID, "png", "image/png", buf, width, height)
}

func (w *ArtifactWriter) WriteText(
	stepIndex int,
	phase string,
	traceID string,
	text string,
) (*StoredArtifact, error) {
	return w.writeBytes(stepIndex, phase, traceID, "txt", "text/plain", []byte(text), 0, 0)
}

func (w *ArtifactWriter) WriteJSON(
	stepIndex int,
	phase string,
	traceID string,
	value any,
) (*StoredArtifact, error) {
	buf, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode %s artifact: %w", phase, err)
	}
	buf = append(buf, '\n')
	return w.writeBytes(stepIndex, phase, traceID, "json", "application/json", buf, 0, 0)
}

func (w *ArtifactWriter) writeBytes(
	stepIndex int,
	phase string,
	traceID string,
	extension string,
	mimeType string,
	buf []byte,
	width int,
	height int,
) (*StoredArtifact, error) {
	if w == nil {
		return nil, fmt.Errorf("artifact writer is not configured")
	}
	filename := fmt.Sprintf(
		"%02d-%s-%s.%s",
		stepIndex,
		sanitizePathComponent(phase, "artifact"),
		time.Now().UTC().Format("20060102T150405Z"),
		extension,
	)
	fullPath := filepath.Join(w.baseDir, filename)
	if err := os.WriteFile(fullPath, buf, 0o600); err != nil {
		return nil, fmt.Errorf("write artifact: %w", err)
	}
	return &StoredArtifact{
		Type:      extension,
		Path:      fullPath,
		MimeType:  mimeType,
		SHA256:    sha256Hex(buf),
		Bytes:     len(buf),
		Width:     width,
		Height:    height,
		TraceID:   strings.TrimSpace(traceID),
		StepIndex: stepIndex,
	}, nil
}

func sanitizePathComponent(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	builder := strings.Builder{}
	builder.Grow(len(trimmed))
	for _, ch := range trimmed {
		builder.WriteRune(sanitizePathRune(ch))
	}
	out := strings.Trim(builder.String(), "_")
	if out == "" {
		return fallback
	}
	return out
}

func sanitizePathRune(ch rune) rune {
	if isPathComponentRune(ch) {
		return ch
	}
	return '_'
}

func isPathComponentRune(ch rune) bool {
	if ch >= 'a' && ch <= 'z' {
		return true
	}
	if ch >= 'A' && ch <= 'Z' {
		return true
	}
	if ch >= '0' && ch <= '9' {
		return true
	}
	return ch == '-' || ch == '_'
}

func sha256Hex(buf []byte) string {
	sum := sha256.Sum256(buf)
	return fmt.Sprintf("%x", sum[:])
}
