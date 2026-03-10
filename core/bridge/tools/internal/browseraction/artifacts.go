package browseraction

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ghost-os/bridge/artifacts"
	"ghost-os/bridge/tools/internal/payloadutil"
)

const (
	visionImageTargetMaxBytes = 450 * 1024
)

type screenshotArtifactStore interface {
	storeScreenshot(pngBytes []byte, payload map[string]any, traceID string) (*ScreenshotArtifactPayload, error)
}

type defaultScreenshotArtifactStore struct{}

func (defaultScreenshotArtifactStore) storeScreenshot(
	pngBytes []byte,
	payload map[string]any,
	traceID string,
) (*ScreenshotArtifactPayload, error) {
	baseDir, err := artifacts.ResolveArtifactsDir(os.Getenv("GHOST_ARTIFACTS_PATH"))
	if err != nil {
		return nil, err
	}
	screenshotsDir := filepath.Join(baseDir, "screenshots")
	if err := os.MkdirAll(screenshotsDir, 0o700); err != nil {
		return nil, fmt.Errorf("create screenshot artifact directory: %w", err)
	}

	sum := sha256.Sum256(pngBytes)
	hash := hex.EncodeToString(sum[:])
	name := screenshotArtifactName(traceID, hash)
	pngPath := filepath.Join(screenshotsDir, name+".png")
	if err := os.WriteFile(pngPath, pngBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write screenshot artifact: %w", err)
	}

	visionBytes, visionMime := buildVisionImage(pngBytes)
	visionPath := filepath.Join(screenshotsDir, name+".vision.jpg")
	if err := os.WriteFile(visionPath, visionBytes, 0o600); err != nil {
		return nil, fmt.Errorf("write screenshot vision artifact: %w", err)
	}

	artifact := &ScreenshotArtifactPayload{
		Type:        "image",
		Path:        pngPath,
		VisionPath:  visionPath,
		MimeType:    "image/png",
		VisionMime:  visionMime,
		SHA256:      hash,
		Bytes:       len(pngBytes),
		VisionBytes: len(visionBytes),
	}
	if width, ok := payloadutil.NumericToInt(payload["width"]); ok {
		artifact.Width = width
	}
	if height, ok := payloadutil.NumericToInt(payload["height"]); ok {
		artifact.Height = height
	}
	if displayID, exists := payload["display_id"]; exists {
		artifact.DisplayID = displayID
	}

	return artifact, nil
}

func buildVisionImage(source []byte) ([]byte, string) {
	img, _, err := image.Decode(bytes.NewReader(source))
	if err != nil {
		return source, "image/png"
	}

	qualities := []int{70, 55, 40, 30}
	best := source
	for _, quality := range qualities {
		var buffer bytes.Buffer
		if err := jpeg.Encode(&buffer, img, &jpeg.Options{Quality: quality}); err != nil {
			continue
		}
		candidate := append([]byte(nil), buffer.Bytes()...)
		best = candidate
		if len(candidate) <= visionImageTargetMaxBytes {
			return candidate, "image/jpeg"
		}
	}
	return best, "image/jpeg"
}

func screenshotArtifactName(traceID string, hash string) string {
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	safeTrace := sanitizeTraceID(traceID)
	if safeTrace == "" {
		safeTrace = "trace"
	}
	hashPrefix := "unknown"
	if len(hash) >= 10 {
		hashPrefix = hash[:10]
	}
	return fmt.Sprintf("%s-%s-%s", stamp, safeTrace, hashPrefix)
}

func sanitizeTraceID(traceID string) string {
	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(traceID))
	for _, ch := range traceID {
		switch {
		case ch >= 'a' && ch <= 'z':
			builder.WriteRune(ch)
		case ch >= 'A' && ch <= 'Z':
			builder.WriteRune(ch)
		case ch >= '0' && ch <= '9':
			builder.WriteRune(ch)
		case ch == '-' || ch == '_':
			builder.WriteRune(ch)
		default:
			builder.WriteRune('-')
		}
	}
	return strings.Trim(builder.String(), "-")
}
