package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ghost-os/bridge/tools/internal/toolartifacts"
	"ghost-os/bridge/tools/internal/tooljson"
)

const screenImageMimeType = "image/png"

func (t *ScreenActionTool) executeScreenshot(ctx context.Context, params map[string]any, traceID string) (string, error) {
	payload, err := t.captureScreen(ctx, params, traceID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.ImagePath) == "" {
		return "", fmt.Errorf("SCREEN_CAPTURE returned empty image_path")
	}

	artifact, err := writeScreenArtifact(ctx, traceID, payload)
	if err != nil {
		return "", err
	}
	return tooljson.Encode(screenActionResult{
		Action:    "screenshot",
		DisplayID: payload.DisplayID,
		Artifact:  artifact,
	})
}

func writeScreenArtifact(
	ctx context.Context,
	traceID string,
	shotPayload screenCapturePayload,
) (*screenActionArtifact, error) {
	baseDir, err := toolartifacts.ResolveScreenshotsRoot()
	if err != nil {
		return nil, err
	}

	sessionID := toolartifacts.SanitizePathComponent(screenSessionID(ctx), "no-session")
	traceToken := toolartifacts.SanitizePathComponent(traceID, "trace")
	toolToken := toolartifacts.SanitizePathComponent(ToolCallIDFromContext(ctx), "tool")
	filename := fmt.Sprintf(
		"screen-%s-%s-%s.png",
		toolartifacts.FormatUTCTimestamp(time.Now().UTC()),
		traceToken,
		toolToken,
	)
	sessionDir := filepath.Join(baseDir, sessionID)
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		return nil, fmt.Errorf("create screenshot directory: %w", err)
	}
	fullPath := filepath.Join(sessionDir, filename)
	written, sha256Value, err := copyAndHashScreenshot(shotPayload.ImagePath, fullPath)
	if err != nil {
		return nil, err
	}
	visionBytes, err := toIntBytes(written)
	if err != nil {
		return nil, err
	}

	return &screenActionArtifact{
		Type:        "image",
		VisionPath:  fullPath,
		VisionMime:  screenImageMimeType,
		Width:       shotPayload.ImageWidth,
		Height:      shotPayload.ImageHeight,
		SHA256:      sha256Value,
		VisionBytes: visionBytes,
	}, nil
}

func copyAndHashScreenshot(sourcePath string, targetPath string) (int64, string, error) {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return 0, "", fmt.Errorf("open screenshot file: %w", err)
	}
	defer sourceFile.Close()

	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, "", fmt.Errorf("create screenshot file: %w", err)
	}

	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(targetFile, hasher), sourceFile)
	closeErr := targetFile.Close()
	if copyErr != nil {
		_ = os.Remove(targetPath)
		return 0, "", fmt.Errorf("copy screenshot file: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(targetPath)
		return 0, "", fmt.Errorf("close screenshot file: %w", closeErr)
	}
	return written, hex.EncodeToString(hasher.Sum(nil)), nil
}

func toIntBytes(value int64) (int, error) {
	bytes := int(value)
	if int64(bytes) != value {
		return 0, fmt.Errorf("screenshot file is too large")
	}
	return bytes, nil
}

func screenSessionID(ctx context.Context) string {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return "no-session"
	}
	return sess.ID
}
