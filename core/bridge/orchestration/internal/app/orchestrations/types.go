package orchestrations

import (
	"context"

	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeTasks "ghost-os/bridge/tasks"
)

type Planner interface {
	Build(definition *bridgeTasks.OrchestrationDefinition) (group.Plan, error)
}

type GroupExecutor interface {
	ExecuteGroup(ctx context.Context, cmd GroupExecuteCommand) GroupResult
}

type GroupExecuteCommand struct {
	GroupNode          bridgeTasks.OrchestrationNode
	MemberNodes        map[string]bridgeTasks.OrchestrationNode
	MemberOrder        []string
	PreviousGroupID    string
	PreviousTranscript group.Transcript
	MemberSessions     map[string]string
	OwnerSessionID     string
	TraceID            string
}

type GroupResult struct {
	Status          string
	Preview         string
	Error           string
	CompletedRounds int
	Transcript      group.Transcript
	MemberResults   []ports.MemberResult
	MemberSessions  map[string]string
	OwnerAgentID    string
	OwnerSessionID  string
	DispatchResults []DispatchResult
}
