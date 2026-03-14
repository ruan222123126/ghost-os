package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

const screenActionDescription = "Run desktop screen perception actions such as screenshot capture, OCR scan, visible-text click, or template icon matching. click_text always verifies the visible OCR label before clicking; optional x and y are used only to choose among verified matches."

const screenActionSchema = `{
	"type":"object",
	"properties":{
		"action":{"type":"string","enum":["screenshot","ocr_scan","click_text","find_icon","click_icon"]},
		"params":{
			"type":"object",
			"description":"Action-specific parameters. screenshot supports display_id. OCR/icon actions support display_id and optional region. click_text requires text and treats optional x/y only as a candidate hint. click_icon accepts template_path or optional x/y direct-click coordinates.",
			"properties":{
				"display_id":{"type":"number","minimum":0,"description":"Optional display ID."},
				"region":{
					"type":"object",
					"description":"Optional capture region.",
					"properties":{
						"x":{"type":"number","description":"Left coordinate."},
						"y":{"type":"number","description":"Top coordinate."},
						"width":{"type":"number","description":"Region width."},
						"height":{"type":"number","description":"Region height."}
					},
					"additionalProperties":false
				},
				"languages":{"type":"array","items":{"type":"string"},"description":"Optional OCR languages, e.g. zh or en."},
				"min_confidence":{"type":"number","minimum":0,"maximum":1,"description":"Optional OCR confidence threshold."},
				"text":{"type":"string","description":"Visible OCR text to match for click_text."},
				"allow_single_char":{"type":"boolean","description":"Allow click_text to target a single visible character."},
				"x":{"type":"number","description":"Optional X hint or direct click coordinate."},
				"y":{"type":"number","description":"Optional Y hint or direct click coordinate."},
				"scale_x":{"type":"number","minimum":0,"description":"Optional display scale factor for X coordinate."},
				"scale_y":{"type":"number","minimum":0,"description":"Optional display scale factor for Y coordinate."},
				"match_mode":{"type":"string","enum":["exact","contains","case_insensitive","normalized","fuzzy"],"description":"Text match mode for click_text."},
				"max_distance":{"type":"number","minimum":1,"description":"Optional max edit distance for fuzzy match_mode."},
				"occurrence":{"type":"number","minimum":1,"description":"Select the Nth matching OCR result."},
				"button":{"type":"string","enum":["left","right","middle"],"description":"Optional mouse button for click actions."},
				"ensure_active_window_title":{"type":"string","description":"Optional active window title substring to verify before clicking."},
				"ensure_active_window_class":{"type":"string","description":"Optional active window class substring to verify before clicking."},
				"reuse_cache":{"type":"boolean","description":"Optional reuse of recent OCR cache for click_text."},
				"cache_ttl_ms":{"type":"number","minimum":0,"description":"Optional OCR cache TTL in milliseconds (default 1000)."},
				"template_path":{"type":"string","description":"Template image path for find_icon or click_icon."},
				"threshold":{"type":"number","minimum":0,"maximum":1,"description":"Optional icon match threshold."},
				"max_results":{"type":"number","minimum":1,"description":"Optional maximum icon matches to return."},
				"scale_range":{
					"type":"object",
					"description":"Optional template scale range for icon matching.",
					"properties":{
						"min":{"type":"number","minimum":0,"description":"Minimum scale factor."},
						"max":{"type":"number","minimum":0,"description":"Maximum scale factor."},
						"step":{"type":"number","minimum":0,"description":"Scale step size."}
					},
					"additionalProperties":false
				}
			},
			"additionalProperties":false
		}
	},
	"required":["action"],
	"additionalProperties":false
}`

type ScreenActionTool struct {
	execution ExecutionClient
	ocrCache  *screenOCRCache
}

type screenActionArgs struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params,omitempty"`
}

type screenRegion struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type screenPoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type screenBoundingBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type screenOCRItem struct {
	Text       string            `json:"text"`
	Confidence float64           `json:"confidence"`
	BBox       screenBoundingBox `json:"bbox"`
	Center     screenPoint       `json:"center"`
}

type screenOCRPayload struct {
	DisplayID   int             `json:"display_id"`
	ImageWidth  int             `json:"image_width"`
	ImageHeight int             `json:"image_height"`
	ScaleX      float64         `json:"scale_x"`
	ScaleY      float64         `json:"scale_y"`
	Region      screenRegion    `json:"region"`
	Items       []screenOCRItem `json:"items"`
}

type iconMatch struct {
	Score  float64           `json:"score"`
	BBox   screenBoundingBox `json:"bbox"`
	Center screenPoint       `json:"center"`
	Scale  float64           `json:"scale,omitempty"`
}

type iconMatchPayload struct {
	DisplayID   int          `json:"display_id"`
	ImageWidth  int          `json:"image_width"`
	ImageHeight int          `json:"image_height"`
	ScaleX      float64      `json:"scale_x"`
	ScaleY      float64      `json:"scale_y"`
	Region      screenRegion `json:"region"`
	Matches     []iconMatch  `json:"matches"`
}

type screenShotPayload struct {
	ImageBase64 string `json:"image_base64"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	DisplayID   int    `json:"display_id"`
}

type screenActionArtifact struct {
	Type        string `json:"type"`
	VisionPath  string `json:"vision_path"`
	VisionMime  string `json:"vision_mime_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	SHA256      string `json:"sha256"`
	VisionBytes int    `json:"vision_bytes"`
}

type screenActionResult struct {
	Action    string                `json:"action"`
	DisplayID int                   `json:"display_id,omitempty"`
	Artifact  *screenActionArtifact `json:"artifact,omitempty"`
}

func NewScreenActionTool(client ExecutionClient) Tool {
	return &ScreenActionTool{
		execution: client,
		ocrCache:  &screenOCRCache{},
	}
}

func (ScreenActionTool) Name() string {
	return "screen_action"
}

func (ScreenActionTool) Description() string {
	return screenActionDescription
}

func (ScreenActionTool) Parameters() json.RawMessage {
	return json.RawMessage(screenActionSchema)
}

func (t *ScreenActionTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}

	var args screenActionArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	switch strings.ToLower(strings.TrimSpace(args.Action)) {
	case "screenshot":
		return t.executeScreenshot(ctx, args.Params, traceID)
	case "ocr_scan":
		return t.executeOCRScan(ctx, args.Params, traceID)
	case "click_text":
		return t.executeClickText(ctx, args.Params, traceID)
	case "find_icon":
		return t.executeFindIcon(ctx, args.Params, traceID)
	case "click_icon":
		return t.executeClickIcon(ctx, args.Params, traceID)
	default:
		return "", fmt.Errorf("unsupported screen action %q", args.Action)
	}
}

func (t *ScreenActionTool) ensureOCRCache() *screenOCRCache {
	if t.ocrCache == nil {
		t.ocrCache = &screenOCRCache{}
	}
	return t.ocrCache
}

func (ScreenActionTool) InterpretResult(output string) ExecuteMeta {
	var result screenActionResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return ExecuteMeta{}
	}
	if strings.TrimSpace(result.Action) != "screenshot" || result.Artifact == nil {
		return ExecuteMeta{}
	}
	if strings.TrimSpace(result.Artifact.Type) != "image" {
		return ExecuteMeta{}
	}
	path := strings.TrimSpace(result.Artifact.VisionPath)
	if path == "" {
		return ExecuteMeta{}
	}
	mimeType := strings.TrimSpace(result.Artifact.VisionMime)
	if mimeType == "" {
		mimeType = "image/png"
	}

	return ExecuteMeta{
		Content: []llm.ContentPart{{
			Type: llm.ContentTypeImage,
			Image: &llm.ImageContent{
				Path:     path,
				MimeType: mimeType,
				Width:    result.Artifact.Width,
				Height:   result.Artifact.Height,
				SHA256:   strings.TrimSpace(result.Artifact.SHA256),
				Bytes:    result.Artifact.VisionBytes,
			},
		}},
	}
}
