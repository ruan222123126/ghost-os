package orchestrations

import (
	"context"
	"errors"
	"strings"

	"ghost-os/bridge/orchestration/internal/domain/group"
	sharedresult "ghost-os/bridge/orchestration/internal/shared/result"
	sharedtext "ghost-os/bridge/orchestration/internal/shared/text"
	bridgeTasks "ghost-os/bridge/tasks"
)

const completedPreview = "orchestration completed"

type Runner struct {
	Planner Planner
	Groups  GroupExecutor
	Mapper  ResultMapper
}

type ExecuteCommand struct {
	Definition *bridgeTasks.OrchestrationDefinition
	TraceID    string
}

type runState struct {
	memberSessionIDs    map[string]map[string]string
	ownerSessionIDs     map[string]string
	lastGroupID         string
	lastGroupTranscript group.Transcript
}

func (r Runner) Execute(ctx context.Context, cmd ExecuteCommand) bridgeTasks.ExecutionResult {
	plan, err := r.buildPlan(cmd.Definition)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: err.Error()}
	}
	if r.Groups == nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: "orchestration group executor is not configured"}
	}
	return r.executePlan(ctx, executePlanCommand{
		plan:    plan,
		traceID: strings.TrimSpace(cmd.TraceID),
	})
}

func (r Runner) buildPlan(definition *bridgeTasks.OrchestrationDefinition) (group.Plan, error) {
	if r.Planner == nil {
		return group.Plan{}, errors.New("orchestration planner is not configured")
	}
	return r.Planner.Build(definition)
}

type executePlanCommand struct {
	plan    group.Plan
	traceID string
}

func (r Runner) executePlan(
	ctx context.Context,
	cmd executePlanCommand,
) bridgeTasks.ExecutionResult {
	recorder := sharedresult.NewNodeResultRecorder(len(cmd.plan.Nodes))
	state := newRunState()
	currentID := cmd.plan.EntryGroupID
	lastPreview := completedPreview
	for currentID != "" {
		result := r.executeCurrentGroup(ctx, cmd, currentID, &state)
		recorder.Record(result.record)
		lastPreview = result.group.Preview
		if result.group.Status != bridgeTasks.RunStatusSuccess {
			return failedExecutionResult(result.group, recorder)
		}
		state.rememberSuccess(currentID, result.group)
		currentID = cmd.plan.ControlNext[currentID]
	}
	return successfulExecutionResult(lastPreview, recorder)
}

func newRunState() runState {
	return runState{
		memberSessionIDs: make(map[string]map[string]string),
		ownerSessionIDs:  make(map[string]string),
	}
}

type currentGroupResult struct {
	group  GroupResult
	record sharedresult.NodeRecord
}

func (r Runner) executeCurrentGroup(
	ctx context.Context,
	cmd executePlanCommand,
	groupID string,
	state *runState,
) currentGroupResult {
	node := cmd.plan.Nodes[groupID]
	memberOrder := append([]string(nil), cmd.plan.GroupMembers[groupID]...)
	result := r.Groups.ExecuteGroup(ctx, GroupExecuteCommand{
		GroupNode:          node,
		MemberNodes:        cmd.plan.Nodes,
		MemberOrder:        memberOrder,
		PreviousGroupID:    state.lastGroupID,
		PreviousTranscript: state.lastGroupTranscript,
		MemberSessions:     CloneSessionIDs(state.memberSessionIDs[groupID]),
		OwnerSessionID:     state.ownerSessionIDs[groupID],
		TraceID:            cmd.traceID,
	})
	return currentGroupResult{
		group:  result,
		record: r.Mapper.GroupRecord(ResultMapCommand{GroupNode: node, MemberOrder: memberOrder, Result: result}),
	}
}

func (s *runState) rememberSuccess(groupID string, result GroupResult) {
	s.memberSessionIDs[groupID] = CloneSessionIDs(result.MemberSessions)
	if result.OwnerSessionID != "" {
		s.ownerSessionIDs[groupID] = result.OwnerSessionID
	}
	s.lastGroupID = groupID
	s.lastGroupTranscript = result.Transcript.Clone()
}

func failedExecutionResult(
	result GroupResult,
	recorder *sharedresult.NodeResultRecorder,
) bridgeTasks.ExecutionResult {
	return bridgeTasks.ExecutionResult{
		Status:          result.Status,
		ResponsePreview: result.Preview,
		Error:           result.Error,
		NodeResults:     recorder.Snapshot(),
	}
}

func successfulExecutionResult(
	lastPreview string,
	recorder *sharedresult.NodeResultRecorder,
) bridgeTasks.ExecutionResult {
	return bridgeTasks.ExecutionResult{
		Status:          bridgeTasks.RunStatusSuccess,
		ResponsePreview: sharedtext.TruncateRunes(lastPreview, bridgeTasks.MaxResponsePreviewRunes),
		NodeResults:     recorder.Snapshot(),
	}
}
