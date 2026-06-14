package orchestrations

import (
	"context"

	"ghost-os/bridge/orchestration/internal/app/orchestrations/dispatch"
	"ghost-os/bridge/orchestration/internal/app/orchestrations/member"
	"ghost-os/bridge/orchestration/internal/app/orchestrations/owner"
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

type RoundDispatcher = dispatch.RoundDispatcher
type RoundDispatchCommand = dispatch.RoundDispatchCommand
type RoundDispatchResult = dispatch.RoundDispatchResult
type MemberRunner = member.Runner
type OwnerControlPromptRequest = owner.ControlPromptRequest

func BuildGroupMemberMessage(req ports.MemberRunRequest) string {
	return member.BuildGroupMemberMessage(req)
}

func BuildOwnerControlPrompt(req OwnerControlPromptRequest) string {
	return owner.BuildControlPrompt(req)
}

func BuildOwnerControlUserPrompt(round int, dispatchToolName string) string {
	return owner.BuildControlUserPrompt(round, dispatchToolName)
}
