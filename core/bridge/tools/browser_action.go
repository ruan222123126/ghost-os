package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

type BrowserActionTool struct {
	execution ExecutionClient
}

type browserActionArgs struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params,omitempty"`
}

const (
	defaultArtifactsPath      = "~/.ghost-os/artifacts"
	visionImageTargetMaxBytes = 450 * 1024
)

// NewBrowserActionTool 创建 browser_action 工具并绑定 execution 客户端。
func NewBrowserActionTool(client ExecutionClient) Tool {
	return &BrowserActionTool{execution: client}
}

func (BrowserActionTool) Name() string {
	return "browser_action"
}

func (BrowserActionTool) Description() string {
	return "Run browser/vision actions via native execution layer: screenshot, click, or query."
}

func (BrowserActionTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["screenshot","click","query"]},
			"params":{"type":"object","description":"Action-specific parameters passed directly to native execution."}
		},
		"required":["action"],
		"additionalProperties":false
	}`)
}

// InterpretResult 解析截图结果并返回可用于多模态推理的 image content。
func (BrowserActionTool) InterpretResult(output string) ExecuteMeta {
	content, ok := decodeBrowserScreenshotContent(output)
	if !ok {
		return ExecuteMeta{}
	}
	return ExecuteMeta{Content: content}
}

// Execute 将语义化 action 映射为 native action，并统一编码返回结果。
func (t *BrowserActionTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}

	var args browserActionArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	nativeAction, err := toNativeBrowserAction(args.Action)
	if err != nil {
		return "", err
	}

	payload, err := t.execution.Call(ctx, nativeAction, args.Params, traceID)
	if err != nil {
		return "", fmt.Errorf("execution %s failed: %w", nativeAction, err)
	}
	if nativeAction == "SCREEN_SHOT" {
		return formatScreenshotPayload(payload, traceID)
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode payload: %w", err)
	}
	return string(encoded), nil
}

// formatScreenshotPayload 在截图场景下输出 artifact 引用，避免把 base64 图像直接塞进历史。
func formatScreenshotPayload(payload map[string]any, traceID string) (string, error) {
	rawImage, ok := payload["image_base64"].(string)
	if !ok || strings.TrimSpace(rawImage) == "" {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("encode payload: %w", err)
		}
		return string(encoded), nil
	}

	pngBytes, err := base64.StdEncoding.DecodeString(rawImage)
	if err != nil {
		return "", fmt.Errorf("decode screenshot base64: %w", err)
	}

	artifact, err := persistScreenshotArtifacts(pngBytes, payload, traceID)
	if err != nil {
		return "", err
	}

	result := map[string]any{
		"action":   "screenshot",
		"artifact": artifact,
	}
	if width, ok := numericToInt(payload["width"]); ok {
		result["width"] = width
	}
	if height, ok := numericToInt(payload["height"]); ok {
		result["height"] = height
	}
	if displayID, exists := payload["display_id"]; exists {
		result["display_id"] = displayID
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode screenshot payload: %w", err)
	}
	return string(encoded), nil
}

// persistScreenshotArtifacts 持久化原图与 vision 压缩图，并补充哈希/尺寸元数据。
func persistScreenshotArtifacts(pngBytes []byte, payload map[string]any, traceID string) (map[string]any, error) {
	baseDir, err := resolveArtifactsDir()
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

	artifact := map[string]any{
		"type":             "image",
		"path":             pngPath,
		"vision_path":      visionPath,
		"mime_type":        "image/png",
		"vision_mime_type": visionMime,
		"sha256":           hash,
		"bytes":            len(pngBytes),
		"vision_bytes":     len(visionBytes),
	}
	if width, ok := numericToInt(payload["width"]); ok {
		artifact["width"] = width
	}
	if height, ok := numericToInt(payload["height"]); ok {
		artifact["height"] = height
	}
	if displayID, exists := payload["display_id"]; exists {
		artifact["display_id"] = displayID
	}

	return artifact, nil
}

// buildVisionImage 生成较小的 JPEG 版本，控制视觉输入体积并降低 token 成本。
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

// resolveArtifactsDir 解析并规范化 artifact 根目录，支持 ~ 展开。
func resolveArtifactsDir() (string, error) {
	path := strings.TrimSpace(os.Getenv("GHOST_ARTIFACTS_PATH"))
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

// screenshotArtifactName 生成可读且低冲突的截图 artifact 文件名。
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

// sanitizeTraceID 仅保留文件名安全字符，避免 trace_id 造成路径污染。
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

// numericToInt 把 JSON 常见数字类型归一为 int，便于统一输出字段。
func numericToInt(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float32:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

type screenshotArtifactPayload struct {
	Type        string `json:"type"`
	Path        string `json:"path"`
	VisionPath  string `json:"vision_path"`
	MimeType    string `json:"mime_type"`
	VisionMime  string `json:"vision_mime_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	SHA256      string `json:"sha256"`
	Bytes       int    `json:"bytes"`
	VisionBytes int    `json:"vision_bytes"`
}

type browserActionOutputPayload struct {
	Action   string                     `json:"action"`
	Artifact *screenshotArtifactPayload `json:"artifact"`
}

func decodeBrowserScreenshotContent(output string) ([]llm.ContentPart, bool) {
	var payload browserActionOutputPayload
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

	return []llm.ContentPart{
		{
			Type: llm.ContentTypeImage,
			Image: &llm.ImageContent{
				Path:     imagePath,
				MimeType: imageMime,
				Width:    payload.Artifact.Width,
				Height:   payload.Artifact.Height,
				SHA256:   strings.TrimSpace(payload.Artifact.SHA256),
				Bytes:    imageBytes,
			},
		},
	}, true
}

// toNativeBrowserAction 把业务动作名映射到 native driver 的 action 常量。
func toNativeBrowserAction(action string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "screenshot":
		return "SCREEN_SHOT", nil
	case "click":
		return "MOUSE_CLICK", nil
	case "query":
		return "BROWSER_QUERY", nil
	default:
		return "", fmt.Errorf("unsupported browser action %q", action)
	}
}
