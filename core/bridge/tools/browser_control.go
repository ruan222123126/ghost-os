package tools

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
)

type BrowserControlTool struct {
	execution ExecutionClient
	mu        sync.Mutex
	sessions  map[string]*browserSession
}

type browserControlArgs struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params,omitempty"`
}

type browserSession struct {
	id         string
	wsEndpoint string
	allocCtx   context.Context
	allocStop  context.CancelFunc
	ctx        context.Context
	stop       context.CancelFunc
	mu         sync.Mutex
	lastUsed   time.Time
}

type browserArtifact struct {
	Type        string `json:"type"`
	VisionPath  string `json:"vision_path"`
	VisionMime  string `json:"vision_mime_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	SHA256      string `json:"sha256"`
	VisionBytes int    `json:"vision_bytes"`
}

func NewBrowserControlTool(client ExecutionClient) Tool {
	return &BrowserControlTool{
		execution: client,
		sessions:  make(map[string]*browserSession),
	}
}

func (BrowserControlTool) Name() string {
	return "browser_control"
}

func (BrowserControlTool) Description() string {
	return "Control a Chrome DevTools Protocol (CDP) browser session for navigation, clicking, typing, DOM extraction, and screenshots. Use connect or launch first to create a session_id, then reuse it for subsequent actions."
}

func (BrowserControlTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["connect","launch","goto","click","type","press","evaluate","content","screenshot","info","close"]},
			"params":{
				"type":"object",
				"properties":{
					"session_id":{"type":"string","description":"Existing browser session id."},
					"endpoint":{"type":"string","description":"CDP websocket URL (ws://) or http(s)://host:port for /json/version discovery."},
					"command":{"type":"string","description":"Launch command that must background itself (launch action)."},
					"debug_port":{"type":"number","minimum":1,"maximum":65535,"description":"Remote debugging port for launch/connect."},
					"wait_timeout_ms":{"type":"number","minimum":0,"description":"Launch wait timeout in milliseconds."},
					"url":{"type":"string","description":"Navigation target URL."},
					"wait":{"type":"string","enum":["none","dom","load"],"description":"Navigation wait mode."},
					"wait_ms":{"type":"number","minimum":0,"description":"Extra wait time after navigation."},
					"selector":{"type":"string","description":"CSS selector for click/type/content."},
					"selector_type":{"type":"string","enum":["css","xpath","text"],"description":"Selector type for click."},
					"text":{"type":"string","description":"Text to match for click (selector_type=text)."},
					"clear":{"type":"boolean","description":"Clear existing value before typing."},
					"key":{"type":"string","description":"Key to press (e.g. Enter, Tab, Escape)."},
					"expression":{"type":"string","description":"JavaScript expression for evaluate."},
					"timeout_ms":{"type":"number","minimum":0,"description":"Per-action timeout in milliseconds."}
				},
				"additionalProperties":false
			}
		},
		"required":["action"],
		"additionalProperties":false
	}`)
}

func (t *BrowserControlTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	var args browserControlArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	action := strings.ToLower(strings.TrimSpace(args.Action))
	switch action {
	case "connect":
		return t.executeConnect(ctx, args.Params)
	case "launch":
		return t.executeLaunch(ctx, args.Params, traceID)
	case "goto":
		return t.executeGoto(ctx, args.Params)
	case "click":
		return t.executeClick(ctx, args.Params)
	case "type":
		return t.executeType(ctx, args.Params)
	case "press":
		return t.executePress(ctx, args.Params)
	case "evaluate":
		return t.executeEvaluate(ctx, args.Params)
	case "content":
		return t.executeContent(ctx, args.Params)
	case "screenshot":
		return t.executeScreenshot(ctx, args.Params, traceID)
	case "info":
		return t.executeInfo(ctx, args.Params)
	case "close":
		return t.executeClose(args.Params)
	default:
		return "", fmt.Errorf("unsupported browser action %q", args.Action)
	}
}

func (t *BrowserControlTool) executeConnect(ctx context.Context, params map[string]any) (string, error) {
	endpoint, err := browserRequiredStringParam(params, "endpoint")
	if err != nil {
		return "", err
	}
	sessionID := strings.TrimSpace(browserOptionalStringParam(params, "session_id"))
	if sessionID == "" {
		sessionID = newBrowserSessionID()
	}
	wsEndpoint, err := resolveWSEndpoint(endpoint, browserParseTimeout(params))
	if err != nil {
		return "", err
	}

	session, err := newRemoteBrowserSession(sessionID, wsEndpoint)
	if err != nil {
		return "", err
	}
	t.storeSession(session)

	result := map[string]any{
		"action":      "connect",
		"session_id":  sessionID,
		"ws_endpoint": wsEndpoint,
		"connected":   true,
	}
	return encodeBrowserJSON(result)
}

func (t *BrowserControlTool) executeLaunch(ctx context.Context, params map[string]any, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}
	command, err := browserRequiredStringParam(params, "command")
	if err != nil {
		return "", err
	}

	_, err = t.execution.Call(ctx, "BASH_EXEC", map[string]any{"command": command}, traceID)
	if err != nil {
		return "", fmt.Errorf("launch command failed: %w", err)
	}

	endpoint := strings.TrimSpace(browserOptionalStringParam(params, "endpoint"))
	if endpoint == "" {
		port, ok := browserOptionalIntParam(params, "debug_port")
		if !ok {
			return "", fmt.Errorf("debug_port or endpoint is required for launch")
		}
		endpoint = fmt.Sprintf("http://127.0.0.1:%d", port)
	}

	waitTimeout := browserParseWaitTimeout(params, 8000*time.Millisecond)
	wsEndpoint, err := waitForWSEndpoint(endpoint, waitTimeout)
	if err != nil {
		return "", err
	}

	sessionID := strings.TrimSpace(browserOptionalStringParam(params, "session_id"))
	if sessionID == "" {
		sessionID = newBrowserSessionID()
	}
	if err := t.replaceSession(sessionID, wsEndpoint); err != nil {
		return "", err
	}

	result := map[string]any{
		"action":      "launch",
		"session_id":  sessionID,
		"ws_endpoint": wsEndpoint,
		"connected":   true,
	}
	return encodeBrowserJSON(result)
}

func (t *BrowserControlTool) executeGoto(ctx context.Context, params map[string]any) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	url, err := browserRequiredStringParam(params, "url")
	if err != nil {
		return "", err
	}
	waitMode := strings.ToLower(strings.TrimSpace(browserOptionalStringParam(params, "wait")))
	if waitMode == "" {
		waitMode = "dom"
	}
	extraWait := browserParseWaitDuration(params, "wait_ms")
	timeout := browserParseTimeout(params)

	var title string
	var location string
	var readyState string
	actions := []chromedp.Action{
		chromedp.Navigate(url),
	}
	switch waitMode {
	case "none":
	case "dom":
		actions = append(actions, chromedp.WaitReady("body", chromedp.ByQuery))
	case "load":
		actions = append(actions, chromedp.WaitReady("body", chromedp.ByQuery),
			chromedp.Evaluate(`document.readyState`, &readyState))
	default:
		return "", fmt.Errorf("wait must be one of: none, dom, load")
	}
	if extraWait > 0 {
		actions = append(actions, chromedp.Sleep(extraWait))
	}
	actions = append(actions, chromedp.Location(&location), chromedp.Title(&title))

	if err := session.run(timeout, actions...); err != nil {
		return "", err
	}

	result := map[string]any{
		"action":     "goto",
		"session_id": session.id,
		"url":        location,
		"title":      title,
	}
	return encodeBrowserJSON(result)
}

func (t *BrowserControlTool) executeClick(ctx context.Context, params map[string]any) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	selectorType := strings.ToLower(strings.TrimSpace(browserOptionalStringParam(params, "selector_type")))
	if selectorType == "" {
		selectorType = "css"
	}
	selector := strings.TrimSpace(browserOptionalStringParam(params, "selector"))
	text := strings.TrimSpace(browserOptionalStringParam(params, "text"))
	if selectorType == "text" {
		if text == "" {
			return "", fmt.Errorf("text is required for selector_type=text")
		}
		var clicked bool
		if err := session.run(browserParseTimeout(params), chromedp.Evaluate(clickByTextScript(text), &clicked)); err != nil {
			return "", err
		}
		if !clicked {
			return "", fmt.Errorf("no element matched text %q", text)
		}
		return encodeBrowserJSON(map[string]any{
			"action":     "click",
			"session_id": session.id,
			"selector":   "text",
			"text":       text,
		})
	}
	if selector == "" {
		return "", fmt.Errorf("selector is required for click")
	}
	if selectorType == "xpath" {
		var clicked bool
		if err := session.run(browserParseTimeout(params), chromedp.Evaluate(clickByXPathScript(selector), &clicked)); err != nil {
			return "", err
		}
		if !clicked {
			return "", fmt.Errorf("no element matched xpath %q", selector)
		}
		return encodeBrowserJSON(map[string]any{
			"action":        "click",
			"session_id":    session.id,
			"selector":      selector,
			"selector_type": "xpath",
		})
	}
	if selectorType != "css" {
		return "", fmt.Errorf("selector_type must be one of: css, xpath, text")
	}
	if err := session.run(browserParseTimeout(params), chromedp.Click(selector, chromedp.ByQuery)); err != nil {
		return "", err
	}
	return encodeBrowserJSON(map[string]any{
		"action":        "click",
		"session_id":    session.id,
		"selector":      selector,
		"selector_type": "css",
	})
}

func (t *BrowserControlTool) executeType(ctx context.Context, params map[string]any) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	selector, err := browserRequiredStringParam(params, "selector")
	if err != nil {
		return "", err
	}
	text, err := browserRequiredStringParam(params, "text")
	if err != nil {
		return "", err
	}
	clear := browserOptionalBoolParam(params, "clear", false)
	var actions []chromedp.Action
	if clear {
		actions = append(actions, chromedp.SetValue(selector, "", chromedp.ByQuery))
	}
	actions = append(actions, chromedp.Focus(selector, chromedp.ByQuery), chromedp.SendKeys(selector, text, chromedp.ByQuery))
	if err := session.run(browserParseTimeout(params), actions...); err != nil {
		return "", err
	}
	return encodeBrowserJSON(map[string]any{
		"action":     "type",
		"session_id": session.id,
		"selector":   selector,
		"length":     len(text),
	})
}

func (t *BrowserControlTool) executePress(ctx context.Context, params map[string]any) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	key, err := browserRequiredStringParam(params, "key")
	if err != nil {
		return "", err
	}
	selector := strings.TrimSpace(browserOptionalStringParam(params, "selector"))
	if selector == "" {
		selector = "body"
	}
	payload := normalizeKeyPress(key)
	if err := session.run(browserParseTimeout(params), chromedp.SendKeys(selector, payload, chromedp.ByQuery)); err != nil {
		return "", err
	}
	return encodeBrowserJSON(map[string]any{
		"action":     "press",
		"session_id": session.id,
		"key":        key,
		"selector":   selector,
	})
}

func (t *BrowserControlTool) executeEvaluate(ctx context.Context, params map[string]any) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	expr, err := browserRequiredStringParam(params, "expression")
	if err != nil {
		return "", err
	}
	var value any
	if err := session.run(browserParseTimeout(params), chromedp.Evaluate(expr, &value, chromedp.EvalAsValue)); err != nil {
		return "", err
	}
	return encodeBrowserJSON(map[string]any{
		"action":     "evaluate",
		"session_id": session.id,
		"value":      value,
	})
}

func (t *BrowserControlTool) executeContent(ctx context.Context, params map[string]any) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	selector := strings.TrimSpace(browserOptionalStringParam(params, "selector"))
	if selector == "" {
		selector = "html"
	}
	var html string
	if err := session.run(browserParseTimeout(params), chromedp.OuterHTML(selector, &html, chromedp.ByQuery)); err != nil {
		return "", err
	}
	return encodeBrowserJSON(map[string]any{
		"action":     "content",
		"session_id": session.id,
		"selector":   selector,
		"content":    html,
	})
}

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
	return encodeBrowserJSON(map[string]any{
		"action":     "screenshot",
		"session_id": session.id,
		"artifact":   artifact,
	})
}

func (t *BrowserControlTool) executeInfo(ctx context.Context, params map[string]any) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	var title string
	var location string
	if err := session.run(browserParseTimeout(params), chromedp.Location(&location), chromedp.Title(&title)); err != nil {
		return "", err
	}
	return encodeBrowserJSON(map[string]any{
		"action":     "info",
		"session_id": session.id,
		"url":        location,
		"title":      title,
	})
}

func (t *BrowserControlTool) executeClose(params map[string]any) (string, error) {
	sessionID := strings.TrimSpace(browserOptionalStringParam(params, "session_id"))
	if sessionID == "" {
		return "", fmt.Errorf("session_id is required")
	}
	session := t.popSession(sessionID)
	if session == nil {
		return "", fmt.Errorf("browser session %q not found", sessionID)
	}
	session.close()
	return encodeBrowserJSON(map[string]any{
		"action":     "close",
		"session_id": sessionID,
		"closed":     true,
	})
}

func (t *BrowserControlTool) sessionFromParams(params map[string]any) (*browserSession, error) {
	sessionID := strings.TrimSpace(browserOptionalStringParam(params, "session_id"))
	if sessionID == "" {
		return nil, fmt.Errorf("session_id is required")
	}
	session := t.getSession(sessionID)
	if session == nil {
		return nil, fmt.Errorf("browser session %q not found", sessionID)
	}
	return session, nil
}

func (t *BrowserControlTool) getSession(sessionID string) *browserSession {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.sessions[sessionID]
}

func (t *BrowserControlTool) popSession(sessionID string) *browserSession {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.sessions == nil {
		return nil
	}
	session := t.sessions[sessionID]
	delete(t.sessions, sessionID)
	return session
}

func (t *BrowserControlTool) storeSession(session *browserSession) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if existing := t.sessions[session.id]; existing != nil {
		existing.close()
	}
	t.sessions[session.id] = session
}

func (t *BrowserControlTool) replaceSession(sessionID string, wsEndpoint string) error {
	session, err := newRemoteBrowserSession(sessionID, wsEndpoint)
	if err != nil {
		return err
	}
	t.storeSession(session)
	return nil
}

func newRemoteBrowserSession(sessionID string, wsEndpoint string) (*browserSession, error) {
	allocCtx, allocStop := chromedp.NewRemoteAllocator(context.Background(), wsEndpoint)
	ctx, stop := chromedp.NewContext(allocCtx)
	return &browserSession{
		id:         sessionID,
		wsEndpoint: wsEndpoint,
		allocCtx:   allocCtx,
		allocStop:  allocStop,
		ctx:        ctx,
		stop:       stop,
		lastUsed:   time.Now(),
	}, nil
}

func (s *browserSession) run(timeout time.Duration, actions ...chromedp.Action) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastUsed = time.Now()
	ctx := s.ctx
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return chromedp.Run(ctx, actions...)
}

func (s *browserSession) close() {
	if s.stop != nil {
		s.stop()
	}
	if s.allocStop != nil {
		s.allocStop()
	}
}

func newBrowserSessionID() string {
	var raw [6]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("browser-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("browser-%x", raw)
}

func resolveWSEndpoint(endpoint string, timeout time.Duration) (string, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return "", fmt.Errorf("endpoint is required")
	}
	if strings.HasPrefix(trimmed, "ws://") || strings.HasPrefix(trimmed, "wss://") {
		return trimmed, nil
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "http://" + trimmed
	}
	return fetchWebSocketURL(trimmed, timeout)
}

func waitForWSEndpoint(endpoint string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		ws, err := resolveWSEndpoint(endpoint, timeout)
		if err == nil {
			return ws, nil
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("timed out waiting for endpoint")
	}
	return "", lastErr
}

func fetchWebSocketURL(base string, timeout time.Duration) (string, error) {
	url := strings.TrimRight(base, "/")
	if !strings.HasSuffix(url, "/json/version") {
		url = url + "/json/version"
	}
	client := &http.Client{Timeout: pickTimeout(timeout, 3*time.Second)}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch %s failed: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch %s failed: status %d", url, resp.StatusCode)
	}
	var payload struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode %s failed: %w", url, err)
	}
	if strings.TrimSpace(payload.WebSocketDebuggerURL) == "" {
		return "", fmt.Errorf("missing webSocketDebuggerUrl in %s", url)
	}
	return payload.WebSocketDebuggerURL, nil
}

func pickTimeout(primary time.Duration, fallback time.Duration) time.Duration {
	if primary > 0 {
		return primary
	}
	return fallback
}

func clickByTextScript(text string) string {
	encoded, _ := json.Marshal(text)
	return fmt.Sprintf(`(() => {
  const query = %s;
  const candidates = Array.from(document.querySelectorAll('a,button,input,textarea,[role="button"],[role="link"]'));
  const exact = candidates.find(el => (el.innerText || el.value || '').trim() === query);
  if (exact) { exact.click(); return true; }
  const fallback = Array.from(document.querySelectorAll('*')).find(el => (el.innerText || '').trim() === query);
  if (fallback) { fallback.click(); return true; }
  return false;
})()`, string(encoded))
}

func clickByXPathScript(xpath string) string {
	encoded, _ := json.Marshal(xpath)
	return fmt.Sprintf(`(() => {
  const path = %s;
  const result = document.evaluate(path, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null);
  const node = result.singleNodeValue;
  if (node && node.click) { node.click(); return true; }
  return false;
})()`, string(encoded))
}

func normalizeKeyPress(key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "enter":
		return "\n"
	case "tab":
		return "\t"
	case "escape", "esc":
		return "\u001b"
	case "backspace":
		return "\b"
	case "space":
		return " "
	default:
		return key
	}
}

func browserRequiredStringParam(params map[string]any, field string) (string, error) {
	if params == nil {
		return "", fmt.Errorf("%s is required", field)
	}
	raw, ok := params[field]
	if !ok {
		return "", fmt.Errorf("%s is required", field)
	}
	value, ok := raw.(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return strings.TrimSpace(value), nil
}

func browserOptionalStringParam(params map[string]any, field string) string {
	if params == nil {
		return ""
	}
	raw, ok := params[field]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func browserOptionalBoolParam(params map[string]any, field string, fallback bool) bool {
	if params == nil {
		return fallback
	}
	raw, ok := params[field]
	if !ok {
		return fallback
	}
	value, ok := raw.(bool)
	if !ok {
		return fallback
	}
	return value
}

func browserOptionalIntParam(params map[string]any, field string) (int, bool) {
	if params == nil {
		return 0, false
	}
	raw, ok := params[field]
	if !ok {
		return 0, false
	}
	switch value := raw.(type) {
	case float64:
		return int(value), true
	case int:
		return value, true
	case int64:
		return int(value), true
	case json.Number:
		parsed, err := value.Int64()
		if err != nil {
			return 0, false
		}
		return int(parsed), true
	default:
		return 0, false
	}
}

func browserParseTimeout(params map[string]any) time.Duration {
	value, ok := browserOptionalIntParam(params, "timeout_ms")
	if !ok || value <= 0 {
		return 0
	}
	return time.Duration(value) * time.Millisecond
}

func browserParseWaitDuration(params map[string]any, field string) time.Duration {
	value, ok := browserOptionalIntParam(params, field)
	if !ok || value <= 0 {
		return 0
	}
	return time.Duration(value) * time.Millisecond
}

func browserParseWaitTimeout(params map[string]any, fallback time.Duration) time.Duration {
	if params == nil {
		return fallback
	}
	value, ok := browserOptionalIntParam(params, "wait_timeout_ms")
	if !ok || value <= 0 {
		return fallback
	}
	return time.Duration(value) * time.Millisecond
}

func writeBrowserScreenshot(ctx context.Context, sessionID string, traceID string, buf []byte) (*browserArtifact, error) {
	baseDir, err := resolveBrowserSnapshotsDir()
	if err != nil {
		return nil, err
	}
	safeSession := sanitizeBrowserPathComponent(sessionID, "no-session")
	safeTrace := sanitizeBrowserPathComponent(traceID, "trace")
	safeTool := sanitizeBrowserPathComponent(ToolCallIDFromContext(ctx), "tool")
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	filename := fmt.Sprintf("browser-%s-%s-%s.png", timestamp, safeTrace, safeTool)
	sessionDir := filepath.Join(baseDir, safeSession)
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		return nil, fmt.Errorf("create screenshot directory: %w", err)
	}
	fullPath := filepath.Join(sessionDir, filename)
	if err := os.WriteFile(fullPath, buf, 0o600); err != nil {
		return nil, fmt.Errorf("write screenshot file: %w", err)
	}

	config, err := decodePNGConfig(buf)
	if err != nil {
		return nil, err
	}
	sha := sha256SumHex(buf)
	return &browserArtifact{
		Type:        "image",
		VisionPath:  fullPath,
		VisionMime:  "image/png",
		Width:       config.Width,
		Height:      config.Height,
		SHA256:      sha,
		VisionBytes: len(buf),
	}, nil
}

func resolveBrowserSnapshotsDir() (string, error) {
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
	return filepath.Join(filepath.Clean(absolute), "browser"), nil
}

func sanitizeBrowserPathComponent(value string, fallback string) string {
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

func decodePNGConfig(buf []byte) (image.Config, error) {
	config, err := png.DecodeConfig(bytes.NewReader(buf))
	if err != nil {
		return image.Config{}, fmt.Errorf("decode png config: %w", err)
	}
	return config, nil
}

func sha256SumHex(buf []byte) string {
	sum := sha256.Sum256(buf)
	return fmt.Sprintf("%x", sum[:])
}

func encodeBrowserJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode payload: %w", err)
	}
	return string(encoded), nil
}
