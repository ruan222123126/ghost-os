package ports

import (
	"context"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
	"ghost-os/bridge/tools"
)

type AgentTurnRunner interface {
	RunTurn(ctx context.Context, message string, sessionID string, traceID string) (string, string, error)
	RunTurnStream(ctx context.Context, message string, sessionID string, traceID string, sink streaming.Sink) (string, string, error)
}

type AgentTurnRunnerWithOverrides interface {
	RunTurnWithOverrides(
		ctx context.Context,
		message string,
		sessionID string,
		traceID string,
		runtimeOverrides *bridgeTasks.TaskRuntimeOverrides,
	) (string, string, error)
}

type RuntimeFactory interface {
	Build(store bridgeconfig.Store) (RuntimeDependencies, error)
}

type RuntimeDependencies interface {
	Config() bridgeconfig.Config
	Completer() agent.Completer
	ToolCatalog() tools.ToolCatalog
	SystemPrompt() string
	Close()
}

type CompletionClient interface {
	Complete(context.Context, llm.CompletionRequest) (*llm.CompletionResponse, error)
}

type TaskExecutor interface {
	Execute(ctx context.Context, task bridgeTasks.ScheduledTask, traceID string) bridgeTasks.ExecutionResult
}

type TaskStore interface {
	SaveTask(task *bridgeTasks.ScheduledTask) error
	LoadTask(taskID string) (*bridgeTasks.ScheduledTask, error)
	DeleteTask(taskID string) error
	ListTasks() ([]bridgeTasks.ScheduledTask, error)
	AppendRunLog(run bridgeTasks.RunLog) error
	ListRunLogs(taskID string, limit int) ([]bridgeTasks.RunLog, error)
}

type SessionStore interface {
	Load(sessionID string) (*session.Session, error)
	Save(sess *session.Session) error
	Delete(sessionID string) error
	List() ([]string, error)
	ListMetadata() ([]session.SessionMetadata, error)
}

type ToolRegistry interface {
	tools.ToolCatalog
	Register(tool tools.Tool)
}
