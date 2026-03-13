package tools

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"ghost-os/bridge/llm"
)

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
	return "Run desktop screen perception actions such as screenshot capture, OCR scan, visible-text click, or template icon matching. click_text always verifies the visible OCR label before clicking; optional x and y are used only to choose among verified matches."
}

func (ScreenActionTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
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
	}`)
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

func (t *ScreenActionTool) executeScreenshot(ctx context.Context, params map[string]any, traceID string) (string, error) {
	payload, err := t.execution.Call(ctx, "SCREEN_SHOT", cloneParams(params), traceID)
	if err != nil {
		return "", fmt.Errorf("execution SCREEN_SHOT failed: %w", err)
	}

	shotPayload, err := decodePayload[screenShotPayload](payload)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(shotPayload.ImageBase64) == "" {
		return "", fmt.Errorf("SCREEN_SHOT returned empty image payload")
	}

	imageBytes, err := base64.StdEncoding.DecodeString(shotPayload.ImageBase64)
	if err != nil {
		return "", fmt.Errorf("decode screenshot image: %w", err)
	}

	baseDir, err := resolveScreenshotsDir()
	if err != nil {
		return "", err
	}

	sessionID := "no-session"
	if sess := SessionFromContext(ctx); sess != nil {
		sessionID = sanitizePathComponent(sess.ID, "no-session")
	}
	traceToken := sanitizePathComponent(traceID, "trace")
	toolToken := sanitizePathComponent(ToolCallIDFromContext(ctx), "tool")
	timestamp := formatUTCTimestamp(time.Now().UTC())
	filename := fmt.Sprintf("screen-%s-%s-%s.png", timestamp, traceToken, toolToken)
	sessionDir := filepath.Join(baseDir, sessionID)
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		return "", fmt.Errorf("create screenshot directory: %w", err)
	}
	fullPath := filepath.Join(sessionDir, filename)
	if err := os.WriteFile(fullPath, imageBytes, 0o600); err != nil {
		return "", fmt.Errorf("write screenshot file: %w", err)
	}

	sha256Sum := sha256.Sum256(imageBytes)
	artifact := &screenActionArtifact{
		Type:        "image",
		VisionPath:  fullPath,
		VisionMime:  "image/png",
		Width:       shotPayload.Width,
		Height:      shotPayload.Height,
		SHA256:      fmt.Sprintf("%x", sha256Sum[:]),
		VisionBytes: len(imageBytes),
	}

	result := screenActionResult{
		Action:    "screenshot",
		DisplayID: shotPayload.DisplayID,
		Artifact:  artifact,
	}
	return encodeJSON(result)
}

func (t *ScreenActionTool) executeOCRScan(ctx context.Context, params map[string]any, traceID string) (string, error) {
	cacheKey, err := buildOCRCacheKey(params)
	if err != nil {
		return "", err
	}
	payload, err := t.execution.Call(ctx, "SCREEN_OCR", cloneParams(params), traceID)
	if err != nil {
		return "", fmt.Errorf("execution SCREEN_OCR failed: %w", err)
	}
	ocrPayload, err := decodePayload[screenOCRPayload](payload)
	if err != nil {
		return "", err
	}
	t.ensureOCRCache().store(cacheKey, ocrPayload)
	return encodeJSON(payload)
}

func (t *ScreenActionTool) executeClickText(ctx context.Context, params map[string]any, traceID string) (string, error) {
	query, err := requiredStringParam(params, "text")
	if err != nil {
		return "", err
	}
	if utf8.RuneCountInString(query) == 1 && !optionalBoolParam(params, "allow_single_char", false) {
		return "", fmt.Errorf("single-character text clicks require allow_single_char=true")
	}
	point, hasPoint, err := parseOptionalPoint(params)
	if err != nil {
		return "", err
	}
	log.Printf("trace_id=%s tool=screen_action action=click_text direct=false x=%v y=%v query=%q", strings.TrimSpace(traceID), params["x"], params["y"], query)
	matchMode := strings.ToLower(rawOptionalStringParam(params, "match_mode", "exact"))
	if !isMatchModeSupported(matchMode) {
		return "", fmt.Errorf("match_mode must be one of: exact, contains, case_insensitive, normalized, fuzzy")
	}
	occurrence := optionalPositiveIntParam(params, "occurrence", 1)
	maxDistance := optionalPositiveIntParam(params, "max_distance", defaultFuzzyDistance(query))

	cacheKey, err := buildOCRCacheKey(params)
	if err != nil {
		return "", err
	}
	cacheTTL := parseCacheTTL(params)
	reuseCache := optionalBoolParam(params, "reuse_cache", true)
	ocrPayload, cached := t.ensureOCRCache().load(cacheKey, cacheTTL, reuseCache)
	if !cached {
		payload, err := t.execution.Call(ctx, "SCREEN_OCR", cloneParams(params), traceID)
		if err != nil {
			return "", fmt.Errorf("execution SCREEN_OCR failed: %w", err)
		}
		ocrPayload, err = decodePayload[screenOCRPayload](payload)
		if err != nil {
			return "", err
		}
		t.ensureOCRCache().store(cacheKey, ocrPayload)
	}
	matches := filterOCRItems(ocrPayload.Items, query, matchMode, maxDistance)
	if hasPoint {
		sortOCRItemsByPoint(matches, point)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no OCR text match for %q", query)
	}
	if occurrence > len(matches) {
		return "", fmt.Errorf("requested occurrence %d but found %d OCR match(es) for %q", occurrence, len(matches), query)
	}
	selected := matches[occurrence-1]

	clickPayload := map[string]any{
		"x": selected.Center.X,
		"y": selected.Center.Y,
	}
	if button := strings.TrimSpace(rawOptionalStringParam(params, "button", "")); button != "" {
		clickPayload["button"] = button
	}
	appendDisplayScaleFromOCR(ocrPayload, clickPayload)
	appendActiveWindowConstraints(params, clickPayload)
	if _, err := t.execution.Call(ctx, "MOUSE_CLICK", clickPayload, traceID); err != nil {
		return "", fmt.Errorf("execution MOUSE_CLICK failed: %w", err)
	}

	return encodeJSON(map[string]any{
		"action":          "click_text",
		"query":           query,
		"match_mode":      matchMode,
		"occurrence":      occurrence,
		"candidate_count": len(matches),
		"hinted":          hasPoint,
		"cached":          cached,
		"clicked":         true,
		"selected":        selected,
	})
}

func (t *ScreenActionTool) executeFindIcon(ctx context.Context, params map[string]any, traceID string) (string, error) {
	payload, err := t.execution.Call(ctx, "ICON_MATCH", cloneParams(params), traceID)
	if err != nil {
		return "", fmt.Errorf("execution ICON_MATCH failed: %w", err)
	}
	return encodeJSON(payload)
}

func (t *ScreenActionTool) executeClickIcon(ctx context.Context, params map[string]any, traceID string) (string, error) {
	if point, ok, err := parseOptionalPoint(params); err != nil {
		return "", err
	} else if ok {
		log.Printf("trace_id=%s tool=screen_action action=click_icon direct=true x=%v y=%v template_path=%q", strings.TrimSpace(traceID), params["x"], params["y"], strings.TrimSpace(rawOptionalStringParam(params, "template_path", "")))
		extra := map[string]any{
			"template_path": strings.TrimSpace(rawOptionalStringParam(params, "template_path", "")),
		}
		appendDisplayScaleFromParams(params, extra)
		appendActiveWindowConstraints(params, extra)
		return t.clickDirectPoint(ctx, traceID, "click_icon", point, strings.TrimSpace(rawOptionalStringParam(params, "button", "")), extra)
	}
	log.Printf("trace_id=%s tool=screen_action action=click_icon direct=false x=%v y=%v template_path=%q", strings.TrimSpace(traceID), params["x"], params["y"], strings.TrimSpace(rawOptionalStringParam(params, "template_path", "")))
	payload, err := t.execution.Call(ctx, "ICON_MATCH", cloneParams(params), traceID)
	if err != nil {
		return "", fmt.Errorf("execution ICON_MATCH failed: %w", err)
	}

	iconPayload, err := decodePayload[iconMatchPayload](payload)
	if err != nil {
		return "", err
	}
	if len(iconPayload.Matches) == 0 {
		templatePath, _ := params["template_path"].(string)
		return "", fmt.Errorf("no icon match for template %q", strings.TrimSpace(templatePath))
	}
	selected := iconPayload.Matches[0]

	clickPayload := map[string]any{
		"x": selected.Center.X,
		"y": selected.Center.Y,
	}
	if button := strings.TrimSpace(rawOptionalStringParam(params, "button", "")); button != "" {
		clickPayload["button"] = button
	}
	appendDisplayScaleFromIcon(iconPayload, clickPayload)
	appendActiveWindowConstraints(params, clickPayload)
	if _, err := t.execution.Call(ctx, "MOUSE_CLICK", clickPayload, traceID); err != nil {
		return "", fmt.Errorf("execution MOUSE_CLICK failed: %w", err)
	}

	return encodeJSON(map[string]any{
		"action":          "click_icon",
		"template_path":   strings.TrimSpace(rawOptionalStringParam(params, "template_path", "")),
		"candidate_count": len(iconPayload.Matches),
		"clicked":         true,
		"selected":        selected,
	})
}

func (t *ScreenActionTool) clickDirectPoint(
	ctx context.Context,
	traceID string,
	action string,
	point screenPoint,
	button string,
	extra map[string]any,
) (string, error) {
	clickPayload := map[string]any{
		"x": point.X,
		"y": point.Y,
	}
	if button != "" {
		clickPayload["button"] = button
	}
	appendDisplayScaleFromParams(extra, clickPayload)
	appendActiveWindowConstraints(extra, clickPayload)
	if _, err := t.execution.Call(ctx, "MOUSE_CLICK", clickPayload, traceID); err != nil {
		return "", fmt.Errorf("execution MOUSE_CLICK failed: %w", err)
	}

	result := map[string]any{
		"action":  action,
		"clicked": true,
		"direct":  true,
		"selected": map[string]any{
			"center": point,
		},
	}
	for key, value := range extra {
		if _, exists := result[key]; !exists && value != nil && value != "" {
			result[key] = value
		}
	}
	return encodeJSON(result)
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
		Content: []llm.ContentPart{
			{
				Type: llm.ContentTypeImage,
				Image: &llm.ImageContent{
					Path:     path,
					MimeType: mimeType,
					Width:    result.Artifact.Width,
					Height:   result.Artifact.Height,
					SHA256:   strings.TrimSpace(result.Artifact.SHA256),
					Bytes:    result.Artifact.VisionBytes,
				},
			},
		},
	}
}

func filterOCRItems(items []screenOCRItem, query string, matchMode string, maxDistance int) []screenOCRItem {
	filtered := make([]screenOCRItem, 0, len(items))
	queryNormalized := normalizeForMatch(query, matchMode)
	for _, item := range items {
		switch matchMode {
		case "exact":
			if strings.TrimSpace(item.Text) != query {
				continue
			}
		case "contains":
			if !strings.Contains(item.Text, query) {
				continue
			}
		case "case_insensitive":
			if !strings.EqualFold(strings.TrimSpace(item.Text), query) {
				continue
			}
		case "normalized":
			if normalizeForMatch(item.Text, matchMode) != queryNormalized {
				continue
			}
		case "fuzzy":
			itemNormalized := normalizeForMatch(item.Text, matchMode)
			if itemNormalized == "" || queryNormalized == "" {
				continue
			}
			if levenshteinDistance(itemNormalized, queryNormalized) > maxDistance {
				continue
			}
		default:
			if !strings.Contains(item.Text, query) {
				continue
			}
		}
		filtered = append(filtered, item)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].BBox.Y == filtered[j].BBox.Y {
			return filtered[i].BBox.X < filtered[j].BBox.X
		}
		return filtered[i].BBox.Y < filtered[j].BBox.Y
	})
	return filtered
}

func isMatchModeSupported(matchMode string) bool {
	switch matchMode {
	case "exact", "contains", "case_insensitive", "normalized", "fuzzy":
		return true
	default:
		return false
	}
}

func defaultFuzzyDistance(query string) int {
	length := utf8.RuneCountInString(strings.TrimSpace(query))
	switch {
	case length <= 4:
		return 1
	case length <= 8:
		return 2
	default:
		return 3
	}
}

func normalizeForMatch(text string, matchMode string) string {
	switch matchMode {
	case "case_insensitive":
		return strings.ToLower(strings.TrimSpace(text))
	case "normalized", "fuzzy":
		return normalizeText(text)
	default:
		return strings.TrimSpace(text)
	}
}

func normalizeText(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))
	for _, r := range text {
		if r == 0x3000 {
			r = ' '
		} else if r >= 0xFF01 && r <= 0xFF5E {
			r = r - 0xFEE0
		}
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			continue
		}
		builder.WriteRune(unicode.ToLower(r))
	}
	return builder.String()
}

func levenshteinDistance(a string, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}

	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := 0; j <= len(rb); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 0
			if ra[i-1] != rb[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = minInt(del, minInt(ins, sub))
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func sortOCRItemsByPoint(items []screenOCRItem, point screenPoint) {
	sort.SliceStable(items, func(i, j int) bool {
		left := distanceSquared(items[i].Center, point)
		right := distanceSquared(items[j].Center, point)
		if left == right {
			if items[i].BBox.Y == items[j].BBox.Y {
				return items[i].BBox.X < items[j].BBox.X
			}
			return items[i].BBox.Y < items[j].BBox.Y
		}
		return left < right
	})
}

func requiredStringParam(params map[string]any, field string) (string, error) {
	value := strings.TrimSpace(rawOptionalStringParam(params, field, ""))
	if value == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return value, nil
}

func rawOptionalStringParam(params map[string]any, field string, defaultValue string) string {
	if len(params) == 0 {
		return defaultValue
	}
	text, _ := params[field].(string)
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultValue
	}
	return text
}

func parseOptionalPoint(params map[string]any) (screenPoint, bool, error) {
	if len(params) == 0 {
		return screenPoint{}, false, nil
	}

	rawX, hasX := numericInt(params["x"])
	rawY, hasY := numericInt(params["y"])
	if !hasX && !hasY {
		return screenPoint{}, false, nil
	}
	if !hasX || !hasY {
		return screenPoint{}, false, fmt.Errorf("x and y must both be provided for direct click")
	}
	return screenPoint{X: rawX, Y: rawY}, true, nil
}

func numericInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case float32:
		return int(typed), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func optionalFloatParam(params map[string]any, field string) (float64, bool) {
	if len(params) == 0 {
		return 0, false
	}
	switch typed := params[field].(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func distanceSquared(a screenPoint, b screenPoint) float64 {
	dx := float64(a.X - b.X)
	dy := float64(a.Y - b.Y)
	return math.Pow(dx, 2) + math.Pow(dy, 2)
}

func optionalPositiveIntParam(params map[string]any, field string, defaultValue int) int {
	if len(params) == 0 {
		return defaultValue
	}
	switch typed := params[field].(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case int32:
		if typed > 0 {
			return int(typed)
		}
	case int64:
		if typed > 0 {
			return int(typed)
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	}
	return defaultValue
}

func optionalBoolParam(params map[string]any, field string, defaultValue bool) bool {
	if len(params) == 0 {
		return defaultValue
	}
	value, ok := params[field].(bool)
	if !ok {
		return defaultValue
	}
	return value
}

func appendActiveWindowConstraints(params map[string]any, payload map[string]any) {
	if len(params) == 0 {
		return
	}
	if title := strings.TrimSpace(rawOptionalStringParam(params, "ensure_active_window_title", "")); title != "" {
		payload["ensure_active_window_title"] = title
	}
	if className := strings.TrimSpace(rawOptionalStringParam(params, "ensure_active_window_class", "")); className != "" {
		payload["ensure_active_window_class"] = className
	}
}

func appendDisplayScaleFromOCR(payload screenOCRPayload, clickPayload map[string]any) {
	clickPayload["display_id"] = payload.DisplayID
	if payload.ScaleX > 0 {
		clickPayload["scale_x"] = payload.ScaleX
	}
	if payload.ScaleY > 0 {
		clickPayload["scale_y"] = payload.ScaleY
	}
}

func appendDisplayScaleFromIcon(payload iconMatchPayload, clickPayload map[string]any) {
	clickPayload["display_id"] = payload.DisplayID
	if payload.ScaleX > 0 {
		clickPayload["scale_x"] = payload.ScaleX
	}
	if payload.ScaleY > 0 {
		clickPayload["scale_y"] = payload.ScaleY
	}
}

func appendDisplayScaleFromParams(params map[string]any, payload map[string]any) {
	if len(params) == 0 {
		return
	}
	if displayID, ok := numericInt(params["display_id"]); ok {
		payload["display_id"] = displayID
	}
	if scaleX, ok := optionalFloatParam(params, "scale_x"); ok && scaleX > 0 {
		payload["scale_x"] = scaleX
	}
	if scaleY, ok := optionalFloatParam(params, "scale_y"); ok && scaleY > 0 {
		payload["scale_y"] = scaleY
	}
}

func resolveScreenshotsDir() (string, error) {
	const defaultPath = "~/.ghost-os/screenshots"
	baseDir := strings.TrimSpace(os.Getenv("GHOST_SCREENSHOTS_PATH"))
	if baseDir == "" {
		baseDir = defaultPath
	}
	if baseDir == "~" || strings.HasPrefix(baseDir, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home for screenshots directory: %w", err)
		}
		if baseDir == "~" {
			baseDir = homeDir
		} else {
			baseDir = filepath.Join(homeDir, strings.TrimPrefix(baseDir, "~/"))
		}
	}
	absolute, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("resolve screenshots directory: %w", err)
	}
	return filepath.Clean(absolute), nil
}

func sanitizePathComponent(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	var builder strings.Builder
	builder.Grow(len(trimmed))
	for _, ch := range trimmed {
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
			builder.WriteByte('_')
		}
	}
	out := strings.Trim(builder.String(), "_")
	if out == "" {
		return fallback
	}
	return out
}

func formatUTCTimestamp(now time.Time) string {
	return fmt.Sprintf("%s%09dZ", now.Format("20060102T150405"), now.Nanosecond())
}

type screenOCRCache struct {
	mu    sync.Mutex
	entry *screenOCRCacheEntry
}

type screenOCRCacheEntry struct {
	key        screenOCRCacheKey
	payload    screenOCRPayload
	capturedAt time.Time
}

type screenOCRCacheKey struct {
	hasDisplayID  bool
	displayID     int
	hasRegion     bool
	region        screenRegion
	languages     []string
	minConfidence float64
}

func (c *screenOCRCache) load(key screenOCRCacheKey, ttl time.Duration, reuse bool) (screenOCRPayload, bool) {
	if !reuse || ttl <= 0 {
		return screenOCRPayload{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entry == nil || !sameOCRCacheKey(c.entry.key, key) {
		return screenOCRPayload{}, false
	}
	if time.Since(c.entry.capturedAt) > ttl {
		return screenOCRPayload{}, false
	}
	return c.entry.payload, true
}

func (c *screenOCRCache) store(key screenOCRCacheKey, payload screenOCRPayload) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entry = &screenOCRCacheEntry{
		key:        key,
		payload:    payload,
		capturedAt: time.Now(),
	}
}

func sameOCRCacheKey(a screenOCRCacheKey, b screenOCRCacheKey) bool {
	if a.hasDisplayID != b.hasDisplayID || a.displayID != b.displayID {
		return false
	}
	if a.hasRegion != b.hasRegion {
		return false
	}
	if a.hasRegion && a.region != b.region {
		return false
	}
	if a.minConfidence != b.minConfidence {
		return false
	}
	if len(a.languages) != len(b.languages) {
		return false
	}
	for i := range a.languages {
		if a.languages[i] != b.languages[i] {
			return false
		}
	}
	return true
}

func buildOCRCacheKey(params map[string]any) (screenOCRCacheKey, error) {
	var key screenOCRCacheKey
	if len(params) == 0 {
		key.languages = defaultOCRLanguages()
		key.minConfidence = defaultOCRConfidence()
		return key, nil
	}
	if displayID, ok := numericInt(params["display_id"]); ok {
		key.hasDisplayID = true
		key.displayID = displayID
	}
	if region, ok, err := parseOptionalRegion(params); err != nil {
		return key, err
	} else if ok {
		key.hasRegion = true
		key.region = region
	}
	languages, err := parseLanguages(params)
	if err != nil {
		return key, err
	}
	key.languages = languages
	if confidence, ok := optionalFloatParam(params, "min_confidence"); ok && confidence > 0 {
		key.minConfidence = confidence
	} else {
		key.minConfidence = defaultOCRConfidence()
	}
	return key, nil
}

func defaultOCRLanguages() []string {
	return []string{"en", "zh"}
}

func defaultOCRConfidence() float64 {
	return 0.75
}

func parseLanguages(params map[string]any) ([]string, error) {
	raw, ok := params["languages"]
	if !ok {
		return defaultOCRLanguages(), nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("languages must be an array")
	}
	normalized := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("languages must contain only strings")
		}
		text = strings.TrimSpace(strings.ToLower(text))
		if text == "" {
			continue
		}
		normalized = append(normalized, text)
	}
	if len(normalized) == 0 {
		return defaultOCRLanguages(), nil
	}
	sort.Strings(normalized)
	normalized = compactStrings(normalized)
	return normalized, nil
}

func compactStrings(items []string) []string {
	if len(items) < 2 {
		return items
	}
	out := items[:1]
	for _, item := range items[1:] {
		if item != out[len(out)-1] {
			out = append(out, item)
		}
	}
	return out
}

func parseOptionalRegion(params map[string]any) (screenRegion, bool, error) {
	if len(params) == 0 {
		return screenRegion{}, false, nil
	}
	raw, ok := params["region"]
	if !ok || raw == nil {
		return screenRegion{}, false, nil
	}
	obj, ok := raw.(map[string]any)
	if !ok {
		return screenRegion{}, false, fmt.Errorf("region must be an object")
	}
	x, ok := numericInt(obj["x"])
	if !ok {
		return screenRegion{}, false, fmt.Errorf("region.x must be a number")
	}
	y, ok := numericInt(obj["y"])
	if !ok {
		return screenRegion{}, false, fmt.Errorf("region.y must be a number")
	}
	width, ok := numericInt(obj["width"])
	if !ok || width <= 0 {
		return screenRegion{}, false, fmt.Errorf("region.width must be a positive number")
	}
	height, ok := numericInt(obj["height"])
	if !ok || height <= 0 {
		return screenRegion{}, false, fmt.Errorf("region.height must be a positive number")
	}
	return screenRegion{X: x, Y: y, Width: width, Height: height}, true, nil
}

func parseCacheTTL(params map[string]any) time.Duration {
	if len(params) == 0 {
		return time.Second
	}
	if value, ok := optionalFloatParam(params, "cache_ttl_ms"); ok && value >= 0 {
		return time.Duration(value) * time.Millisecond
	}
	return time.Second
}

func cloneParams(params map[string]any) map[string]any {
	if len(params) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(params))
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
}

func decodePayload[T any](payload map[string]any) (T, error) {
	var out T
	encoded, err := json.Marshal(payload)
	if err != nil {
		return out, fmt.Errorf("encode payload: %w", err)
	}
	if err := json.Unmarshal(encoded, &out); err != nil {
		return out, fmt.Errorf("decode payload: %w", err)
	}
	return out, nil
}

func encodeJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode payload: %w", err)
	}
	return string(encoded), nil
}
