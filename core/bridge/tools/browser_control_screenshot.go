package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ghost-os/bridge/tools/internal/toolartifacts"
	"ghost-os/bridge/tools/internal/tooljson"

	"github.com/chromedp/chromedp"
)

const browserScreenshotDir = "browser"

func (t *BrowserControlTool) executeScreenshot(ctx context.Context, params map[string]any, traceID string) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	var buf []byte
	if err := session.run(browserParseTimeout(params), chromedp.CaptureScreenshot(&buf)); err != nil {
		return "", err
	}
	artifact, err := writeBrowserScreenshot(ctx, session.id, traceID, buf)
	if err != nil {
		return "", err
	}
	return tooljson.Encode(map[string]any{
		"action":     "screenshot",
		"session_id": session.id,
		"artifact":   artifact,
	})
}

func writeBrowserScreenshot(ctx context.Context, sessionID string, traceID string, buf []byte) (*browserArtifact, error) {
	baseDir, err := toolartifacts.ResolveScreenshotsSubdir(browserScreenshotDir)
	if err != nil {
		return nil, err
	}
	safeSession := toolartifacts.SanitizePathComponent(sessionID, "no-session")
	safeTrace := toolartifacts.SanitizePathComponent(traceID, "trace")
	safeTool := toolartifacts.SanitizePathComponent(ToolCallIDFromContext(ctx), "tool")
	filename := fmt.Sprintf(
		"browser-%s-%s-%s.png",
		toolartifacts.FormatUTCTimestamp(time.Now().UTC()),
		safeTrace,
		safeTool,
	)
	sessionDir := filepath.Join(baseDir, safeSession)
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		return nil, fmt.Errorf("create screenshot directory: %w", err)
	}
	fullPath := filepath.Join(sessionDir, filename)
	if err := os.WriteFile(fullPath, buf, 0o600); err != nil {
		return nil, fmt.Errorf("write screenshot file: %w", err)
	}
	config, err := toolartifacts.DecodePNGConfig(buf)
	if err != nil {
		return nil, err
	}
	return &browserArtifact{
		Type:        "image",
		VisionPath:  fullPath,
		VisionMime:  "image/png",
		Width:       config.Width,
		Height:      config.Height,
		SHA256:      toolartifacts.SHA256Hex(buf),
		VisionBytes: len(buf),
	}, nil
}
