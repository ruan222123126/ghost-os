package browseraction

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools/internal/payloadutil"
)

type OutputProcessor struct {
	screenshots screenshotResultProcessor
}

type screenshotResultProcessor interface {
	process(payload map[string]any, traceID string) (string, error)
}

type screenshotArtifactProcessor struct {
	store screenshotArtifactStore
}

type ScreenshotArtifactPayload struct {
	Type        string `json:"type"`
	Path        string `json:"path"`
	VisionPath  string `json:"vision_path"`
	MimeType    string `json:"mime_type"`
	VisionMime  string `json:"vision_mime_type"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	DisplayID   any    `json:"display_id,omitempty"`
	SHA256      string `json:"sha256"`
	Bytes       int    `json:"bytes"`
	VisionBytes int    `json:"vision_bytes"`
}

type outputPayload struct {
	Action   string                     `json:"action"`
	Artifact *ScreenshotArtifactPayload `json:"artifact"`
}

func NewOutputProcessor() OutputProcessor {
	return OutputProcessor{screenshots: screenshotArtifactProcessor{store: defaultScreenshotArtifactStore{}}}
}

func (p OutputProcessor) PostProcess(output string, traceID string) (string, error) {
	payload, ok := decodePayload(output)
	if !ok || !isScreenshotPayload(payload) {
		return output, nil
	}
	if p.screenshots == nil {
		return output, nil
	}
	return p.screenshots.process(payload, traceID)
}

func (p OutputProcessor) Content(output string) []llm.ContentPart {
	content, ok := decodeScreenshotContent(output)
	if !ok {
		return nil
	}
	return content
}

func (p screenshotArtifactProcessor) process(payload map[string]any, traceID string) (string, error) {
	rawImage, _ := payload["image_base64"].(string)
	pngBytes, err := base64.StdEncoding.DecodeString(rawImage)
	if err != nil {
		return "", fmt.Errorf("decode screenshot base64: %w", err)
	}

	artifact, err := p.store.storeScreenshot(pngBytes, payload, traceID)
	if err != nil {
		return "", err
	}

	result := map[string]any{
		"action":   "screenshot",
		"artifact": artifact,
	}
	if width, ok := payloadutil.NumericToInt(payload["width"]); ok {
		result["width"] = width
	}
	if height, ok := payloadutil.NumericToInt(payload["height"]); ok {
		result["height"] = height
	}
	if displayID, exists := payload["display_id"]; exists {
		result["display_id"] = displayID
	}

	return encodePayload(result)
}

func decodePayload(output string) (map[string]any, bool) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func encodePayload(payload any) (string, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode payload: %w", err)
	}
	return string(encoded), nil
}

func isScreenshotPayload(payload map[string]any) bool {
	rawImage, ok := payload["image_base64"].(string)
	return ok && strings.TrimSpace(rawImage) != ""
}

func decodeScreenshotContent(output string) ([]llm.ContentPart, bool) {
	var payload outputPayload
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		return nil, false
	}
	if strings.TrimSpace(payload.Action) != "screenshot" || payload.Artifact == nil {
		return nil, false
	}
	if strings.TrimSpace(payload.Artifact.Type) != "image" {
		return nil, false
	}

	imagePath := strings.TrimSpace(payload.Artifact.VisionPath)
	imageMime := strings.TrimSpace(payload.Artifact.VisionMime)
	imageBytes := payload.Artifact.VisionBytes
	if imagePath == "" {
		imagePath = strings.TrimSpace(payload.Artifact.Path)
		imageMime = strings.TrimSpace(payload.Artifact.MimeType)
		imageBytes = payload.Artifact.Bytes
	}
	if imagePath == "" {
		return nil, false
	}
	if imageMime == "" {
		imageMime = "image/png"
	}

	return []llm.ContentPart{{
		Type: llm.ContentTypeImage,
		Image: &llm.ImageContent{
			Path:     imagePath,
			MimeType: imageMime,
			Width:    payload.Artifact.Width,
			Height:   payload.Artifact.Height,
			SHA256:   strings.TrimSpace(payload.Artifact.SHA256),
			Bytes:    imageBytes,
		},
	}}, true
}
