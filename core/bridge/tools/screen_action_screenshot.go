package tools

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
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
	if payload.ImageBase64 == "" {
		return "", fmt.Errorf("SCREEN_CAPTURE returned empty image payload")
	}

	imageBytes, err := base64.StdEncoding.DecodeString(payload.ImageBase64)
	if err != nil {
		return "", fmt.Errorf("decode screenshot image: %w", err)
	}

	artifact, err := writeScreenArtifact(ctx, traceID, imageBytes, payload)
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
	imageBytes []byte,
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
	if err := os.WriteFile(fullPath, imageBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write screenshot file: %w", err)
	}

	return &screenActionArtifact{
		Type:        "image",
		VisionPath:  fullPath,
		VisionMime:  screenImageMimeType,
		Width:       shotPayload.ImageWidth,
		Height:      shotPayload.ImageHeight,
		SHA256:      toolartifacts.SHA256Hex(imageBytes),
		VisionBytes: len(imageBytes),
	}, nil
}

func screenSessionID(ctx context.Context) string {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return "no-session"
	}
	return sess.ID
}
