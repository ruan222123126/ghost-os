package agentturn

import (
	"context"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

const (
	ModeDefault = ""
	ModePlan    = "plan"
)

type RequestRuntimeOptions struct {
	ProjectRoot string
}

type PreparedRequest struct {
	UserInput        llm.Message
	Message          string
	Mode             string
	SessionID        string
	RequestRuntime   *RequestRuntimeOptions
	RuntimeOverrides *bridgeTasks.TaskRuntimeOverrides
}

type FinalizedTurn struct {
	Message    string
	SessionID  string
	SessionEnd *api.AssistantSessionEndSignalPayload
}

type ResponseMeta struct {
	Mode string
}

type SessionGuards interface {
	EnsureSessionNotInflight(sessionID string) error
	EnsureSessionActive(sessionID string) error
}

type Runner interface {
	RunTurn(ctx context.Context, req PreparedRequest, traceID string) (string, string, error)
	RunTurnStream(ctx context.Context, req PreparedRequest, traceID string, sink streaming.Sink) (string, string, error)
}

type SpecialModeRunner interface {
	RunPlan(ctx context.Context, req PreparedRequest, traceID string) (api.AgentResponse, int, error)
}

type Finalizer interface {
	Finalize(response string, sessionID string) (FinalizedTurn, error)
	NewResponsePayload(turn FinalizedTurn, meta ResponseMeta) (api.AgentResponse, error)
}

type Publisher interface {
	PublishAssistant(traceID string, turn FinalizedTurn)
	PublishAwaitingHuman(traceID string, sessionID string, awaitingErr *agent.ErrAwaitingHuman)
}

type ErrorClassifier interface {
	Classify(err error) (*agent.ErrAwaitingHuman, bus.ServiceErrorKind, bool, error)
}

type Logger interface {
	Log(traceID string, action string, status string, err error)
}

type RunStopper interface {
	CancelAndWaitBySessionID(ctx context.Context, sessionID string) (StopHandle, error)
	CancelAndWaitByTraceID(ctx context.Context, traceID string) (StopHandle, error)
}

type StopHandle struct {
	SessionID string
}

type Service struct {
	Guards     SessionGuards
	Runner     Runner
	Special    SpecialModeRunner
	Finalizer  Finalizer
	Publisher  Publisher
	Classifier ErrorClassifier
	Logger     Logger
	Stopper    RunStopper
}
