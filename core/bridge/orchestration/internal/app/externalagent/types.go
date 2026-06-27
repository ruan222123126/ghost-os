package externalagent

import (
	"context"
	"encoding/json"
	"errors"
)

const (
	ProviderCodex  = "codex"
	ProviderClaude = "claude"
)

const (
	StatusIdle             = "idle"
	StatusRunning          = "running"
	StatusAwaitingApproval = "awaiting_approval"
	StatusError            = "error"
)

const (
	DecisionApproved           = "approved"
	DecisionApprovedForSession = "approved_for_session"
	DecisionDenied             = "denied"
	DecisionAbort              = "abort"
)

var (
	ErrNotImplemented     = errors.New("external agent provider is not implemented")
	ErrSessionRequired    = errors.New("session_id is required")
	ErrApprovalNotFound   = errors.New("external approval not found")
	ErrExternalRunActive  = errors.New("external agent session already has an active turn")
	ErrExternalRunMissing = errors.New("external agent session has no active turn")
)

type ClientFactory func(ClientConfig) CodexClient

type ClientConfig struct {
	CodexPath string
	CWD       string
}

type ThreadOptions struct {
	ThreadID       string
	Model          string
	CWD            string
	ApprovalPolicy string
	Sandbox        string
}

type ThreadResult struct {
	ThreadID string
	Model    string
}

type TurnOptions struct {
	ThreadID       string
	Message        string
	Model          string
	CWD            string
	ApprovalPolicy string
	Sandbox        string
	Effort         string
}

type CodexEvent struct {
	Type    string
	Method  string
	Payload map[string]any
}

type ApprovalRequest struct {
	ID        string
	Kind      string
	Tool      string
	CallID    string
	Prompt    string
	Payload   map[string]any
	Legacy    bool
	MCP       bool
	RawID     int
	RawMethod string
}

type CodexClient interface {
	Connect(context.Context) error
	StartThread(context.Context, ThreadOptions) (ThreadResult, error)
	ResumeThread(context.Context, ThreadOptions) (ThreadResult, error)
	StartTurn(context.Context, TurnOptions) (string, error)
	InterruptTurn(context.Context, string, string) error
	SetEventHandler(func(CodexEvent))
	SetApprovalHandler(func(context.Context, ApprovalRequest) (string, error))
	Close() error
}

func DefaultClientFactory(cfg ClientConfig) CodexClient {
	return NewAppServerClient(cfg)
}

func clonePayload(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]any, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}

func rawJSONMap(raw map[string]any) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return data
}
