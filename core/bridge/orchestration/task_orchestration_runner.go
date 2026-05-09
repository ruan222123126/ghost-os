package orchestration

import (
	"context"
	"strings"

	groupdomain "ghost-os/bridge/orchestration/internal/domain/group"
	bridgeTasks "ghost-os/bridge/tasks"
)

type orchestrationTranscript = groupdomain.Transcript
type orchestrationTranscriptEntry = groupdomain.TranscriptEntry
type orchestrationDispatchRequest = groupdomain.DispatchCommand

type orchestrationGroupResult struct {
	status          string
	preview         string
	errText         string
	completedRounds int
	transcript      orchestrationTranscript
	memberResults   []orchestrationMemberResult
	memberSessions  map[string]string
	ownerAgentID    string
	ownerSessionID  string
	dispatchResults []orchestrationDispatchResult
}

type orchestrationRunState struct {
	memberSessionIDs    map[string]map[string]string
	ownerSessionIDs     map[string]string
	lastGroupID         string
	lastGroupTranscript orchestrationTranscript
}

type orchestrationDispatchResult struct {
	Round             int                         `json:"round"`
	Action            string                      `json:"action"`
	Order             string                      `json:"order,omitempty"`
	Instruction       string                      `json:"instruction,omitempty"`
	ParticipantIDs    []string                    `json:"participant_ids,omitempty"`
	MemberResults     []orchestrationMemberResult `json:"member_results,omitempty"`
	PrivateTranscript orchestrationTranscript     `json:"private_transcript,omitempty"`
	OwnerVisible      bool                        `json:"owner_visible"`
}

func (a taskExecutorAdapter) executeOrchestrationTask(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
) bridgeTasks.ExecutionResult {
	plan, err := buildOrchestrationExecutionPlan(task.Orchestration)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	if a.service == nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "task executor service is not configured"}
	}
	return newOrchestrationTaskRunner(a, plan, traceID).execute(ctx)
}

type orchestrationTaskRunner struct {
	adapter taskExecutorAdapter
	plan    orchestrationExecutionPlan
	traceID string
}

func newOrchestrationTaskRunner(adapter taskExecutorAdapter, plan orchestrationExecutionPlan, traceID string) orchestrationTaskRunner {
	return orchestrationTaskRunner{adapter: adapter, plan: plan, traceID: strings.TrimSpace(traceID)}
}

func (r orchestrationTaskRunner) execute(ctx context.Context) bridgeTasks.ExecutionResult {
	recorder := newWorkflowNodeResultRecorder(len(r.plan.nodes))
	state := orchestrationRunState{
		memberSessionIDs: make(map[string]map[string]string),
		ownerSessionIDs:  make(map[string]string),
	}
	currentID := r.plan.entryGroupID
	lastPreview := "orchestration completed"
	for currentID != "" {
		node := r.plan.nodes[currentID]
		result := r.executeGroupNode(ctx, node, &state)
		recorder.record(workflowNodeRecord{
			NodeID:   node.ID,
			NodeType: node.Type,
			Status:   result.status,
			Input: map[string]any{
				"title":          node.Group.Title,
				"shared_context": node.Group.SharedContext,
				"speaking_mode":  node.Group.SpeakingMode,
				"owner_agent_id": node.Group.OwnerAgentID,
				"max_rounds":     node.Group.MaxRounds,
				"member_order":   append([]string(nil), r.plan.groupMember[node.ID]...),
			},
			Output: map[string]any{
				"completed_rounds":   result.completedRounds,
				"speaking_mode":      node.Group.SpeakingMode,
				"owner_agent_id":     result.ownerAgentID,
				"owner_session_id":   result.ownerSessionID,
				"member_order":       append([]string(nil), r.plan.groupMember[node.ID]...),
				"member_session_ids": result.memberSessions,
				"shared_transcript":  result.transcript,
				"member_results":     result.memberResults,
				"dispatch_results":   result.dispatchResults,
			},
			Preview: result.preview,
			Error:   result.errText,
		})
		lastPreview = result.preview
		if result.status != taskRunStatusSuccess {
			return bridgeTasks.ExecutionResult{
				Status:          result.status,
				ResponsePreview: result.preview,
				Error:           result.errText,
				NodeResults:     recorder.snapshot(),
			}
		}
		state.lastGroupID = node.ID
		state.lastGroupTranscript = result.transcript.Clone()
		currentID = r.plan.controlNext[currentID]
	}
	return bridgeTasks.ExecutionResult{
		Status:          taskRunStatusSuccess,
		ResponsePreview: truncateRunes(lastPreview, maxTaskResponsePreviewRunes),
		NodeResults:     recorder.snapshot(),
	}
}

func (r orchestrationTaskRunner) executeGroupNode(
	ctx context.Context,
	node OrchestrationNode,
	state *orchestrationRunState,
) orchestrationGroupResult {
	if node.Group.SpeakingMode == orchestrationModeOwner {
		return r.executeOwnerGroupNode(ctx, node, state)
	}
	return r.executeStandardGroupNode(ctx, node, state)
}
