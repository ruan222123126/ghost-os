package externalagent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const rpcRequestTimeout = 30 * time.Second

type appServerClient struct {
	cfg ClientConfig

	mu              sync.Mutex
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	pending         map[int]chan rpcResponse
	nextID          int64
	connected       bool
	eventHandler    func(CodexEvent)
	approvalHandler func(context.Context, ApprovalRequest) (string, error)
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc,omitempty"`
	ID      int    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type rpcEnvelope struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      *int            `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

func NewAppServerClient(cfg ClientConfig) CodexClient {
	return &appServerClient{
		cfg:     cfg,
		pending: make(map[int]chan rpcResponse),
	}
}

func (c *appServerClient) SetEventHandler(handler func(CodexEvent)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.eventHandler = handler
}

func (c *appServerClient) SetApprovalHandler(handler func(context.Context, ApprovalRequest) (string, error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.approvalHandler = handler
}

func (c *appServerClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return nil
	}
	launchSpec, err := resolveCodexLaunchSpec(c.cfg.CodexPath, c.cfg.NodePath)
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("prepare codex app-server launch: %w", err)
	}
	cmd := exec.Command(launchSpec.executablePath, "app-server", "--stdio")
	cmd.Dir = strings.TrimSpace(c.cfg.CWD)
	if cmd.Dir == "" {
		cmd.Dir = "."
	}
	cmd.Env = codexEnv(launchSpec.pathOverride)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("open codex stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("open codex stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("open codex stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		c.mu.Unlock()
		return fmt.Errorf("start codex app-server: %w", err)
	}
	c.cmd = cmd
	c.stdin = stdin
	c.connected = true
	c.mu.Unlock()

	go c.readStdout(stdout)
	go c.readStderr(stderr)
	go c.waitProcess(cmd)

	initParams := map[string]any{
		"clientInfo": map[string]any{
			"name":    "ghost-os-codex",
			"title":   "Ghost-OS Codex Client",
			"version": "0.1.0",
		},
		"capabilities": map[string]any{"experimentalApi": true},
	}
	if _, err := c.request(ctx, "initialize", initParams); err != nil {
		_ = c.Close()
		return err
	}
	return c.notify("initialized", nil)
}

func (c *appServerClient) StartThread(ctx context.Context, opts ThreadOptions) (ThreadResult, error) {
	params := threadParams(opts, "")
	raw, err := c.request(ctx, "thread/start", params)
	if err != nil {
		return ThreadResult{}, err
	}
	return decodeThreadResult(raw)
}

func (c *appServerClient) ResumeThread(ctx context.Context, opts ThreadOptions) (ThreadResult, error) {
	params := threadParams(opts, opts.ThreadID)
	raw, err := c.request(ctx, "thread/resume", params)
	if err != nil {
		return ThreadResult{}, err
	}
	return decodeThreadResult(raw)
}

func threadParams(opts ThreadOptions, threadID string) map[string]any {
	if strings.TrimSpace(threadID) != "" {
		params := map[string]any{
			"threadId":               strings.TrimSpace(threadID),
			"model":                  nil,
			"modelProvider":          nil,
			"cwd":                    defaultString(opts.CWD, "."),
			"approvalPolicy":         nil,
			"sandbox":                nil,
			"config":                 nil,
			"baseInstructions":       nil,
			"developerInstructions":  nil,
			"persistExtendedHistory": true,
		}
		applyThreadOverrides(params, opts)
		return params
	}
	params := map[string]any{
		"model":                  nil,
		"modelProvider":          nil,
		"profile":                nil,
		"cwd":                    defaultString(opts.CWD, "."),
		"approvalPolicy":         nil,
		"sandbox":                nil,
		"config":                 nil,
		"baseInstructions":       nil,
		"developerInstructions":  nil,
		"compactPrompt":          nil,
		"includeApplyPatchTool":  nil,
		"experimentalRawEvents":  false,
		"persistExtendedHistory": true,
	}
	applyThreadOverrides(params, opts)
	return params
}

func applyThreadOverrides(params map[string]any, opts ThreadOptions) {
	if model := strings.TrimSpace(opts.Model); model != "" {
		params["model"] = model
	}
	if policy := strings.TrimSpace(opts.ApprovalPolicy); policy != "" {
		params["approvalPolicy"] = policy
	}
	if sandbox := strings.TrimSpace(opts.Sandbox); sandbox != "" {
		params["sandbox"] = sandbox
	}
}

func decodeThreadResult(raw json.RawMessage) (ThreadResult, error) {
	var decoded struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return ThreadResult{}, fmt.Errorf("decode thread result: %w", err)
	}
	if strings.TrimSpace(decoded.Thread.ID) == "" {
		return ThreadResult{}, fmt.Errorf("thread result missing thread.id")
	}
	return ThreadResult{ThreadID: strings.TrimSpace(decoded.Thread.ID), Model: strings.TrimSpace(decoded.Model)}, nil
}

func (c *appServerClient) SetCollaborationMode(ctx context.Context, opts CollaborationModeOptions) error {
	mode := strings.TrimSpace(opts.Mode)
	if mode == "" {
		return nil
	}

	params := map[string]any{
		"threadId": strings.TrimSpace(opts.ThreadID),
		"collaborationMode": map[string]any{
			"mode": mode,
			"settings": map[string]any{
				"model":                  strings.TrimSpace(opts.Model),
				"reasoning_effort":       collaborationModeEffort(opts.Effort),
				"developer_instructions": nil,
			},
		},
	}
	_, err := c.request(ctx, "thread/settings/update", params)
	return err
}

func collaborationModeEffort(effort string) any {
	trimmed := strings.TrimSpace(effort)
	if trimmed != "" {
		return trimmed
	}
	return nil
}

func (c *appServerClient) StartTurn(ctx context.Context, opts TurnOptions) (string, error) {
	params := map[string]any{
		"threadId": strings.TrimSpace(opts.ThreadID),
		"input": []map[string]string{{
			"type": "text",
			"text": opts.Message,
		}},
	}
	if cwd := strings.TrimSpace(opts.CWD); cwd != "" {
		params["cwd"] = cwd
	}
	if policy := strings.TrimSpace(opts.ApprovalPolicy); policy != "" {
		params["approvalPolicy"] = policy
	}
	if model := strings.TrimSpace(opts.Model); model != "" {
		params["model"] = model
	}
	if effort := strings.TrimSpace(opts.Effort); effort != "" {
		params["effort"] = effort
	}
	if sandbox := strings.TrimSpace(opts.Sandbox); sandbox != "" {
		params["sandboxPolicy"] = sandboxPolicy(sandbox)
	}
	raw, err := c.request(ctx, "turn/start", params)
	if err != nil {
		return "", err
	}
	var decoded struct {
		Turn struct {
			ID string `json:"id"`
		} `json:"turn"`
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &decoded)
	}
	return strings.TrimSpace(decoded.Turn.ID), nil
}
