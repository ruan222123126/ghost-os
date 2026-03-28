package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

const browserControlDescription = "Control a Chrome DevTools Protocol (CDP) browser session for navigation, clicking, typing, DOM extraction, and screenshots. Use connect or launch first to create a session_id, then reuse it for subsequent actions."

const browserControlSchema = `{
	"type":"object",
	"properties":{
		"action":{"type":"string","enum":["connect","launch","goto","click","type","press","evaluate","content","screenshot","info","close"]},
		"params":{
			"type":"object",
			"properties":{
				"session_id":{"type":"string","description":"Existing browser session id."},
				"endpoint":{"type":"string","description":"CDP websocket URL (ws://) or http(s)://host:port for /json/version discovery."},
				"command":{"type":"string","description":"Optional launch command for the launch action. If omitted, bridge auto-detects a Chrome-compatible binary in execution PATH and launches it."},
				"debug_port":{"type":"number","minimum":1,"maximum":65535,"description":"Remote debugging port for launch/connect. Required when endpoint is omitted, and also required for auto-launch when endpoint has no explicit port."},
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
}`

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
	return browserControlDescription
}

func (BrowserControlTool) Parameters() json.RawMessage {
	return json.RawMessage(browserControlSchema)
}

func (t *BrowserControlTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	var args browserControlArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	switch strings.ToLower(strings.TrimSpace(args.Action)) {
	case "connect":
		return t.executeConnect(args.Params)
	case "launch":
		return t.executeLaunch(ctx, args.Params, traceID)
	case "goto":
		return t.executeGoto(args.Params)
	case "click":
		return t.executeClick(args.Params)
	case "type":
		return t.executeType(args.Params)
	case "press":
		return t.executePress(args.Params)
	case "evaluate":
		return t.executeEvaluate(args.Params)
	case "content":
		return t.executeContent(args.Params)
	case "screenshot":
		return t.executeScreenshot(ctx, args.Params, traceID)
	case "info":
		return t.executeInfo(args.Params)
	case "close":
		return t.executeClose(args.Params)
	default:
		return "", fmt.Errorf("unsupported browser action %q", args.Action)
	}
}
