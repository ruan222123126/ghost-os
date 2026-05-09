package ports

import (
	"context"
	"time"

	"ghost-os/bridge/orchestration/internal/domain/group"
	bridgeTasks "ghost-os/bridge/tasks"
	"ghost-os/bridge/tools"
)

type MemberAgentRunner interface {
	RunMember(ctx context.Context, req MemberRunRequest) (MemberResult, error)
}

type MemberRunRequest struct {
	GroupNode      bridgeTasks.OrchestrationNode
	MemberNode     bridgeTasks.OrchestrationNode
	TranscriptText string
	Round          int
	SessionID      string
	Instruction    string
	Private        bool
	TraceID        string
}

type MemberResult struct {
	Round            int            `json:"round"`
	AgentID          string         `json:"agent_id"`
	Title            string         `json:"title"`
	Status           string         `json:"status"`
	SessionID        string         `json:"session_id,omitempty"`
	Content          string         `json:"content,omitempty"`
	Preview          string         `json:"preview,omitempty"`
	Error            string         `json:"error,omitempty"`
	RuntimeOverrides map[string]any `json:"runtime_overrides,omitempty"`
}

type MemberTurnExecutor interface {
	RunMemberTurn(ctx context.Context, req MemberTurnRequest) (MemberResult, error)
}

type MemberTurnRequest struct {
	Message          string
	SessionID        string
	TraceID          string
	Round            int
	AgentID          string
	Title            string
	RuntimeOverrides *bridgeTasks.TaskRuntimeOverrides
}

type AgentActionInvoker interface {
	ExecuteAgentAction(ctx context.Context, req AgentActionRequest) (AgentActionPayload, error)
}

type AgentActionRequest struct {
	Message          string
	SessionID        string
	TraceID          string
	RuntimeOverrides *bridgeTasks.TaskRuntimeOverrides
}

type AgentActionPayloadKind string

const (
	AgentActionPayloadSuccess     AgentActionPayloadKind = "success"
	AgentActionPayloadAwaiting    AgentActionPayloadKind = "awaiting_human"
	AgentActionPayloadUnsupported AgentActionPayloadKind = "unsupported"
)

type AgentActionPayload struct {
	Kind      AgentActionPayloadKind
	SessionID string
	Message   string
	Prompt    string
	TypeName  string
}

type OwnerDecisionRunner interface {
	Decide(ctx context.Context, req OwnerDecisionRequest) (group.DispatchCommand, string, error)
}

type OwnerDecisionTurnExecutor interface {
	RunOwnerDecisionTurn(ctx context.Context, req OwnerDecisionTurnRequest) (group.DispatchCommand, string, error)
}

type OwnerDecisionRequest struct {
	OwnerNode        bridgeTasks.OrchestrationNode
	GroupNode        bridgeTasks.OrchestrationNode
	MemberNodes      map[string]bridgeTasks.OrchestrationNode
	MemberOrder      []string
	PublicTranscript group.Transcript
	LastDispatch     group.DispatchCommand
	OwnerSessionID   string
	Round            int
	TraceID          string
}

type OwnerDecisionTurnRequest struct {
	OwnerNode        bridgeTasks.OrchestrationNode
	GroupNode        bridgeTasks.OrchestrationNode
	MemberNodes      map[string]bridgeTasks.OrchestrationNode
	MemberOrder      []string
	PublicTranscript group.Transcript
	LastDispatch     group.DispatchCommand
	RuntimeOverrides *bridgeTasks.TaskRuntimeOverrides
	UserPrompt       string
	SessionID        string
	Round            int
	TraceID          string
	Catalog          tools.ToolCatalog
}

type Clock interface {
	Now() time.Time
}
