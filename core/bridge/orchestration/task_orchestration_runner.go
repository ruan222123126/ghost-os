package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	groupdomain "ghost-os/bridge/orchestration/internal/domain/group"
	bridgeTasks "ghost-os/bridge/tasks"
	"ghost-os/bridge/tools"
)

type orchestrationTranscript = groupdomain.Transcript
type orchestrationTranscriptEntry = groupdomain.TranscriptEntry
type orchestrationDispatchRequest = groupdomain.DispatchCommand

type orchestrationMemberResult struct {
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

func (r orchestrationTaskRunner) executeStandardGroupNode(
	ctx context.Context,
	node OrchestrationNode,
	state *orchestrationRunState,
) orchestrationGroupResult {
	groupID := node.ID
	transcript := groupdomain.Initial(node, state.lastGroupID, state.lastGroupTranscript)
	memberOrder := r.plan.groupMember[groupID]
	memberSessions := cloneGroupSessionIDs(state.memberSessionIDs[groupID])
	memberResults := make([]orchestrationMemberResult, 0, len(memberOrder)*node.Group.MaxRounds)
	completedRounds := 0
	for round := 1; round <= node.Group.MaxRounds; round++ {
		roundResults := r.executeGroupRound(ctx, groupID, node, memberOrder, transcript, memberSessions, round)
		memberResults = append(memberResults, roundResults...)
		failure := selectGroupFailure(roundResults)
		for _, result := range roundResults {
			if result.Status == taskRunStatusSuccess {
				transcript = transcript.AppendMember(result.Round, result.Title, result.AgentID, result.Content)
			}
			if result.SessionID != "" {
				memberSessions[result.AgentID] = result.SessionID
			}
		}
		if failure != nil {
			return orchestrationGroupResult{
				status:          failure.Status,
				preview:         truncateRunes(failure.Preview, maxTaskResponsePreviewRunes),
				errText:         strings.TrimSpace(failure.Error),
				completedRounds: completedRounds,
				transcript:      transcript,
				memberResults:   memberResults,
				memberSessions:  memberSessions,
			}
		}
		completedRounds = round
	}
	state.memberSessionIDs[groupID] = cloneGroupSessionIDs(memberSessions)
	return orchestrationGroupResult{
		status:          taskRunStatusSuccess,
		preview:         truncateRunes(transcript.Format(), maxTaskResponsePreviewRunes),
		completedRounds: completedRounds,
		transcript:      transcript,
		memberResults:   memberResults,
		memberSessions:  memberSessions,
	}
}

func (r orchestrationTaskRunner) executeOwnerGroupNode(
	ctx context.Context,
	node OrchestrationNode,
	state *orchestrationRunState,
) orchestrationGroupResult {
	groupID := node.ID
	publicTranscript := groupdomain.Initial(node, state.lastGroupID, state.lastGroupTranscript)
	memberOrder := r.plan.groupMember[groupID]
	memberSessions := cloneGroupSessionIDs(state.memberSessionIDs[groupID])
	ownerSessionID := strings.TrimSpace(state.ownerSessionIDs[groupID])
	memberResults := make([]orchestrationMemberResult, 0, len(memberOrder)*node.Group.MaxRounds)
	dispatchResults := make([]orchestrationDispatchResult, 0, node.Group.MaxRounds)
	completedRounds := 0
	lastDispatch := orchestrationDispatchResult{}

	for round := 1; round <= node.Group.MaxRounds; round++ {
		dispatch, nextSessionID, dispatchErr := r.runOwnerDispatch(ctx, node, memberOrder, publicTranscript, lastDispatch, ownerSessionID, round)
		if dispatchErr != nil {
			return orchestrationGroupResult{
				status:          taskRunStatusError,
				preview:         truncateRunes(dispatchErr.Error(), maxTaskResponsePreviewRunes),
				errText:         dispatchErr.Error(),
				transcript:      publicTranscript,
				memberResults:   memberResults,
				memberSessions:  memberSessions,
				ownerAgentID:    node.Group.OwnerAgentID,
				ownerSessionID:  ownerSessionID,
				dispatchResults: dispatchResults,
			}
		}
		ownerSessionID = nextSessionID
		dispatchResult := newDispatchResult(dispatch, round, node.Group.OwnerAgentID)
		if dispatch.Action == groupdomain.DispatchActionEndGroup {
			dispatchResults = append(dispatchResults, dispatchResult)
			state.memberSessionIDs[groupID] = cloneGroupSessionIDs(memberSessions)
			state.ownerSessionIDs[groupID] = ownerSessionID
			return orchestrationGroupResult{
				status:          taskRunStatusSuccess,
				preview:         truncateRunes(publicTranscript.Format(), maxTaskResponsePreviewRunes),
				completedRounds: completedRounds,
				transcript:      publicTranscript,
				memberResults:   memberResults,
				memberSessions:  memberSessions,
				ownerAgentID:    node.Group.OwnerAgentID,
				ownerSessionID:  ownerSessionID,
				dispatchResults: dispatchResults,
			}
		}

		dispatchResult, transcriptUpdate, memberSessionUpdate := r.executeOwnerDispatch(ctx, node, publicTranscript, memberSessions, dispatchResult, round)
		memberResults = append(memberResults, dispatchResult.MemberResults...)
		memberSessions = memberSessionUpdate
		publicTranscript = transcriptUpdate
		lastDispatch = dispatchResult

		failure := selectGroupFailure(dispatchResult.MemberResults)
		if failure != nil {
			state.ownerSessionIDs[groupID] = ownerSessionID
			return orchestrationGroupResult{
				status:          failure.Status,
				preview:         truncateRunes(failure.Preview, maxTaskResponsePreviewRunes),
				errText:         strings.TrimSpace(failure.Error),
				completedRounds: completedRounds,
				transcript:      publicTranscript,
				memberResults:   memberResults,
				memberSessions:  memberSessions,
				ownerAgentID:    node.Group.OwnerAgentID,
				ownerSessionID:  ownerSessionID,
				dispatchResults: dispatchResults,
			}
		}
		dispatchResults = append(dispatchResults, dispatchResult)
		completedRounds = round
	}

	state.memberSessionIDs[groupID] = cloneGroupSessionIDs(memberSessions)
	state.ownerSessionIDs[groupID] = ownerSessionID
	return orchestrationGroupResult{
		status:          taskRunStatusSuccess,
		preview:         truncateRunes(publicTranscript.Format(), maxTaskResponsePreviewRunes),
		completedRounds: completedRounds,
		transcript:      publicTranscript,
		memberResults:   memberResults,
		memberSessions:  memberSessions,
		ownerAgentID:    node.Group.OwnerAgentID,
		ownerSessionID:  ownerSessionID,
		dispatchResults: dispatchResults,
	}
}

func (r orchestrationTaskRunner) runOwnerDispatch(
	ctx context.Context,
	groupNode OrchestrationNode,
	memberOrder []string,
	publicTranscript orchestrationTranscript,
	lastDispatch orchestrationDispatchResult,
	sessionID string,
	round int,
) (orchestrationDispatchRequest, string, error) {
	ownerNode, ok := r.plan.nodes[groupNode.Group.OwnerAgentID]
	if !ok || ownerNode.Agent == nil {
		return orchestrationDispatchRequest{}, sessionID, fmt.Errorf("owner agent %q is not available", groupNode.Group.OwnerAgentID)
	}
	registry := tools.NewRegistry()
	registry.Register(r.newOwnerDispatchTool(groupNode, memberOrder))
	catalog := tools.NewScopedCatalog(registry, []string{orchestrationDispatchToolName})
	return r.runOwnerDispatchTurn(ctx, ownerNode, groupNode, catalog, publicTranscript, lastDispatch, sessionID, round)
}

func (r orchestrationTaskRunner) runOwnerDispatchTurn(
	ctx context.Context,
	ownerNode OrchestrationNode,
	groupNode OrchestrationNode,
	catalog tools.ToolCatalog,
	publicTranscript orchestrationTranscript,
	lastDispatch orchestrationDispatchResult,
	sessionID string,
	round int,
) (orchestrationDispatchRequest, string, error) {
	if r.adapter.service == nil || r.adapter.service.runtimeFactory == nil {
		return orchestrationDispatchRequest{}, sessionID, errors.New("owner dispatch runtime is not configured")
	}
	deps, err := r.buildOwnerDispatchDeps(ownerNode)
	if err != nil {
		return orchestrationDispatchRequest{}, sessionID, err
	}
	defer deps.Close()
	systemPrompt := buildOwnerControlPrompt(
		deps.systemPrompt,
		ownerNode,
		groupNode,
		r.plan,
		publicTranscript,
		lastDispatch,
		round,
	)

	sess, created, err := newSessionHistoryBuilder(
		deps.cfg.Provider,
		systemPrompt,
		r.adapter.service.sessionStore,
		deps.cfg.ToolSearch.IdleTurns,
		deps.cfg.MicrocompactEnabled,
		"",
	).LoadOrCreateSession(sessionID)
	if err != nil {
		return orchestrationDispatchRequest{}, sessionID, err
	}
	if created && r.adapter.service.sessionStore != nil {
		if err := r.adapter.service.sessionStore.Save(sess); err != nil {
			return orchestrationDispatchRequest{}, sessionID, err
		}
	}

	preparer := newSessionTurnPreparer(
		r.adapter.service.runtimeFactory,
		r.adapter.service.configStore,
		r.adapter.service.sessionStore,
		r.adapter.service.runRegistry,
		nil,
	)
	history := agent.NewHistory("")
	history.SetConversationState(sess.ConversationState)
	for _, message := range sess.Messages {
		history.Append(message)
	}
	history.UpdateSystemPrompt(systemPrompt)
	runAgent := agent.NewAgentWithHistory(deps.client, catalog, history, deps.cfg.MaxTurns)
	runAgent.SetResponseOptions(llm.CloneResponseOptions(deps.cfg.ResponseOptions))
	runAgent.SetCompletionRetryPolicy(agent.NewCompletionRetryPolicy(
		deps.cfg.LLMCompletionRetryCount,
		time.Duration(deps.cfg.LLMCompletionRetryIntervalMS)*time.Millisecond,
	))
	preparer.attachDynamicPromptRefresh(runAgent, deps, sess, catalog, func() (string, error) {
		return systemPrompt, nil
	})

	userMessage := llm.Message{Role: llm.RoleUser, Text: buildOwnerControlUserPrompt(round)}
	execCtx, cleanup, err := preparer.prepareExecutionContext(ctx, sess, deps.registry, r.traceID)
	if err != nil {
		return orchestrationDispatchRequest{}, sessionID, err
	}
	defer cleanup()
	response, runErr := runAgent.RunMessageWithTraceID(execCtx, userMessage, r.traceID)
	newMessages := runAgent.GetNewMessages()
	sess.ConversationState = runAgent.GetConversationState()
	for _, message := range newMessages {
		sess.AddMessage(message)
	}
	if r.adapter.service.sessionStore != nil {
		if saveErr := r.adapter.service.sessionStore.Save(sess); saveErr != nil {
			return orchestrationDispatchRequest{}, sessionID, saveErr
		}
	}
	if runErr != nil {
		var handoff *agent.ErrIterationHandoff
		if errors.As(runErr, &handoff) {
			var request orchestrationDispatchRequest
			if err := json.Unmarshal([]byte(strings.TrimSpace(handoff.Did)), &request); err != nil {
				return orchestrationDispatchRequest{}, sess.ID, fmt.Errorf("decode owner dispatch handoff: %w", err)
			}
			return request, sess.ID, nil
		}
		return orchestrationDispatchRequest{}, sess.ID, runErr
	}
	if strings.TrimSpace(response) == "" {
		return orchestrationDispatchRequest{}, sess.ID, errors.New("owner must call orchestration_dispatch")
	}
	return orchestrationDispatchRequest{}, sess.ID, errors.New("owner must call orchestration_dispatch")
}

func (r orchestrationTaskRunner) executeOwnerDispatch(
	ctx context.Context,
	groupNode OrchestrationNode,
	publicTranscript orchestrationTranscript,
	memberSessions map[string]string,
	result orchestrationDispatchResult,
	round int,
) (orchestrationDispatchResult, orchestrationTranscript, map[string]string) {
	participantIDs := append([]string(nil), result.ParticipantIDs...)
	result.ParticipantIDs = append([]string(nil), participantIDs...)
	result.OwnerVisible = containsString(participantIDs, strings.TrimSpace(groupNode.Group.OwnerAgentID))
	updatedSessions := cloneGroupSessionIDs(memberSessions)

	switch result.Action {
	case groupdomain.DispatchActionPublicOnce:
		order := strings.TrimSpace(result.Order)
		results := r.executeOwnerPublicDispatch(ctx, groupNode, participantIDs, publicTranscript, updatedSessions, round, order, strings.TrimSpace(result.Instruction))
		result.Order = order
		result.MemberResults = results
		nextTranscript := publicTranscript.Clone()
		for _, item := range results {
			if item.Status == taskRunStatusSuccess {
				nextTranscript = nextTranscript.AppendMember(round, item.Title, item.AgentID, item.Content)
			}
			if item.SessionID != "" {
				updatedSessions[item.AgentID] = item.SessionID
			}
		}
		return result, nextTranscript, updatedSessions
	case groupdomain.DispatchActionPrivateOnce:
		results, privateTranscript := r.executeOwnerPrivateDispatch(ctx, groupNode, participantIDs, publicTranscript, round, strings.TrimSpace(result.Instruction))
		result.MemberResults = results
		if result.OwnerVisible {
			result.PrivateTranscript = privateTranscript
		}
		return result, publicTranscript.Clone(), updatedSessions
	default:
		return result, publicTranscript.Clone(), updatedSessions
	}
}

func newDispatchResult(
	dispatch orchestrationDispatchRequest,
	round int,
	ownerAgentID string,
) orchestrationDispatchResult {
	participants := append([]string(nil), dispatch.ParticipantIDs...)
	return orchestrationDispatchResult{
		Round:          round,
		Action:         strings.TrimSpace(dispatch.Action),
		Order:          strings.TrimSpace(dispatch.Order),
		Instruction:    strings.TrimSpace(dispatch.Instruction),
		ParticipantIDs: append([]string(nil), participants...),
		OwnerVisible:   containsString(participants, strings.TrimSpace(ownerAgentID)),
	}
}

func (r orchestrationTaskRunner) buildOwnerDispatchDeps(ownerNode OrchestrationNode) (agentRuntimeDependencies, error) {
	if r.adapter.service == nil || r.adapter.service.runtimeFactory == nil {
		return agentRuntimeDependencies{}, errors.New("owner dispatch runtime is not configured")
	}
	overrides := cloneTaskRuntimeOverrides(ownerNode.Agent.RuntimeOverrides)
	normalized, err := normalizeTaskRuntimeOverrides(overrides)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	preparer := newSessionTurnPreparer(
		r.adapter.service.runtimeFactory,
		r.adapter.service.configStore,
		r.adapter.service.sessionStore,
		r.adapter.service.runRegistry,
		nil,
	)
	deps, _, _, err := preparer.buildPrepareDependencies(normalized)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	return deps, nil
}

func (r orchestrationTaskRunner) executeOwnerPublicDispatch(
	ctx context.Context,
	groupNode OrchestrationNode,
	participantIDs []string,
	transcript orchestrationTranscript,
	memberSessions map[string]string,
	round int,
	order string,
	instruction string,
) []orchestrationMemberResult {
	if order == groupdomain.SpeakingModeParallel {
		return r.executeParallelDispatch(ctx, groupNode, participantIDs, transcript, memberSessions, round, instruction)
	}
	return r.executeSequentialDispatch(ctx, groupNode, participantIDs, transcript, memberSessions, round, instruction)
}

func (r orchestrationTaskRunner) executeSequentialDispatch(
	ctx context.Context,
	groupNode OrchestrationNode,
	participantIDs []string,
	transcript orchestrationTranscript,
	memberSessions map[string]string,
	round int,
	instruction string,
) []orchestrationMemberResult {
	results := make([]orchestrationMemberResult, 0, len(participantIDs))
	visibleTranscript := transcript.Clone()
	for _, agentID := range participantIDs {
		member := r.runGroupMemberWithInstruction(
			ctx,
			groupNode,
			r.plan.nodes[agentID],
			visibleTranscript.Format(),
			round,
			memberSessions[agentID],
			instruction,
			false,
		)
		results = append(results, member)
		if member.Status == taskRunStatusSuccess {
			visibleTranscript = visibleTranscript.AppendMember(round, member.Title, member.AgentID, member.Content)
		}
		if member.Status != taskRunStatusSuccess {
			return results
		}
	}
	return results
}

func (r orchestrationTaskRunner) executeParallelDispatch(
	ctx context.Context,
	groupNode OrchestrationNode,
	participantIDs []string,
	transcript orchestrationTranscript,
	memberSessions map[string]string,
	round int,
	instruction string,
) []orchestrationMemberResult {
	visibleTranscript := transcript.Format()
	roundCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make([]orchestrationMemberResult, len(participantIDs))
	var wg sync.WaitGroup
	for index, agentID := range participantIDs {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			results[i] = r.runGroupMemberWithInstruction(
				roundCtx,
				groupNode,
				r.plan.nodes[id],
				visibleTranscript,
				round,
				memberSessions[id],
				instruction,
				false,
			)
			if results[i].Status != taskRunStatusSuccess {
				cancel()
			}
		}(index, agentID)
	}
	wg.Wait()
	return results
}

func (r orchestrationTaskRunner) executeOwnerPrivateDispatch(
	ctx context.Context,
	groupNode OrchestrationNode,
	participantIDs []string,
	publicTranscript orchestrationTranscript,
	round int,
	instruction string,
) ([]orchestrationMemberResult, orchestrationTranscript) {
	privateTranscript := publicTranscript.Clone()
	results := make([]orchestrationMemberResult, 0, len(participantIDs))
	for _, agentID := range participantIDs {
		member := r.runGroupMemberWithInstruction(
			ctx,
			groupNode,
			r.plan.nodes[agentID],
			privateTranscript.Format(),
			round,
			"",
			instruction,
			true,
		)
		results = append(results, member)
		if member.Status == taskRunStatusSuccess {
			privateTranscript = privateTranscript.AppendMember(round, member.Title, member.AgentID, member.Content)
		}
		if member.Status != taskRunStatusSuccess {
			return results, privateTranscript
		}
	}
	return results, privateTranscript
}

func (r orchestrationTaskRunner) executeGroupRound(
	ctx context.Context,
	groupID string,
	groupNode OrchestrationNode,
	memberOrder []string,
	transcript orchestrationTranscript,
	memberSessions map[string]string,
	round int,
) []orchestrationMemberResult {
	if groupNode.Group.SpeakingMode == orchestrationModeSequential {
		return r.executeSequentialRound(ctx, groupID, groupNode, memberOrder, transcript, memberSessions, round)
	}
	return r.executeParallelRound(ctx, groupID, groupNode, memberOrder, transcript, memberSessions, round)
}

func (r orchestrationTaskRunner) executeSequentialRound(ctx context.Context, groupID string, groupNode OrchestrationNode, memberOrder []string, transcript orchestrationTranscript, memberSessions map[string]string, round int) []orchestrationMemberResult {
	results := make([]orchestrationMemberResult, 0, len(memberOrder))
	visibleTranscript := transcript.Clone()
	for _, agentID := range memberOrder {
		member := r.runGroupMember(ctx, groupNode, r.plan.nodes[agentID], visibleTranscript.Format(), round, memberSessions[agentID])
		results = append(results, member)
		if member.Status == taskRunStatusSuccess {
			visibleTranscript = visibleTranscript.AppendMember(round, member.Title, member.AgentID, member.Content)
		}
		if member.Status != taskRunStatusSuccess {
			return results
		}
	}
	return results
}

func (r orchestrationTaskRunner) executeParallelRound(ctx context.Context, groupID string, groupNode OrchestrationNode, memberOrder []string, transcript orchestrationTranscript, memberSessions map[string]string, round int) []orchestrationMemberResult {
	visibleTranscript := transcript.Format()
	roundCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make([]orchestrationMemberResult, len(memberOrder))
	var wg sync.WaitGroup
	for index, agentID := range memberOrder {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			results[i] = r.runGroupMember(roundCtx, groupNode, r.plan.nodes[id], visibleTranscript, round, memberSessions[id])
			if results[i].Status != taskRunStatusSuccess {
				cancel()
			}
		}(index, agentID)
	}
	wg.Wait()
	return results
}

func (r orchestrationTaskRunner) runGroupMember(ctx context.Context, groupNode OrchestrationNode, memberNode OrchestrationNode, transcriptText string, round int, sessionID string) orchestrationMemberResult {
	return r.runGroupMemberWithInstruction(ctx, groupNode, memberNode, transcriptText, round, sessionID, "", false)
}

func (r orchestrationTaskRunner) runGroupMemberWithInstruction(
	ctx context.Context,
	groupNode OrchestrationNode,
	memberNode OrchestrationNode,
	transcriptText string,
	round int,
	sessionID string,
	instruction string,
	private bool,
) orchestrationMemberResult {
	startedAt := time.Now().UTC()
	_ = startedAt
	message := buildGroupMemberMessage(groupNode, memberNode, transcriptText, round, instruction, private)
	runtimeOverrides := cloneTaskRuntimeOverrides(memberNode.Agent.RuntimeOverrides)
	snapshot := taskRuntimeOverrideSnapshot(runtimeOverrides)
	result, err := r.adapter.service.executeAgentActionWithRuntimeOverrides(
		ctx,
		agentParams{Message: message, SessionID: sessionID},
		runtimeOverrides,
		r.traceID,
	)
	if err != nil {
		return orchestrationMemberResult{
			Round:            round,
			AgentID:          memberNode.ID,
			Title:            memberNode.Agent.Title,
			Status:           taskRunStatusError,
			Preview:          err.Error(),
			Error:            err.Error(),
			RuntimeOverrides: snapshot,
		}
	}
	switch payload := result.Payload.(type) {
	case agentResponse:
		return orchestrationMemberResult{
			Round:            round,
			AgentID:          memberNode.ID,
			Title:            memberNode.Agent.Title,
			Status:           taskRunStatusSuccess,
			SessionID:        payload.SessionID,
			Content:          payload.Message,
			Preview:          payload.Message,
			RuntimeOverrides: snapshot,
		}
	case askHumanAwaitingResponse:
		return orchestrationMemberResult{
			Round:            round,
			AgentID:          memberNode.ID,
			Title:            memberNode.Agent.Title,
			Status:           taskRunStatusAwaitingHuman,
			SessionID:        payload.SessionID,
			Preview:          payload.Prompt,
			Error:            payload.Prompt,
			RuntimeOverrides: snapshot,
		}
	default:
		return orchestrationMemberResult{
			Round:            round,
			AgentID:          memberNode.ID,
			Title:            memberNode.Agent.Title,
			Status:           taskRunStatusError,
			Preview:          "unsupported orchestration member response",
			Error:            fmt.Sprintf("unsupported orchestration member response %T", result.Payload),
			RuntimeOverrides: snapshot,
		}
	}
}

func buildGroupMemberMessage(
	groupNode OrchestrationNode,
	memberNode OrchestrationNode,
	transcriptText string,
	round int,
	instruction string,
	private bool,
) string {
	privacyText := "公开"
	if private {
		privacyText = "私密"
	}
	instructionBlock := ""
	if strings.TrimSpace(instruction) != "" {
		instructionBlock = "\n\n本轮群主附加指令：\n" + strings.TrimSpace(instruction)
	}
	return strings.TrimSpace(fmt.Sprintf(
		"成员角色提示：\n%s\n\n群共享上下文：\n%s\n\n当前可见 group transcript：\n%s\n\n轮次信息：\n当前轮次：%d\n总轮次上限：%d\n发言模式：%s\n回合可见性：%s%s\n\n请继续群聊发言，直接输出你这一轮要说的话。",
		memberNode.Agent.Message,
		groupNode.Group.SharedContext,
		transcriptText,
		round,
		groupNode.Group.MaxRounds,
		groupNode.Group.SpeakingMode,
		privacyText,
		instructionBlock,
	))
}

const orchestrationDispatchToolName = "orchestration_dispatch"

type orchestrationDispatchTool struct {
	groupNode   OrchestrationNode
	memberOrder []string
}

func (t orchestrationDispatchTool) Name() string {
	return orchestrationDispatchToolName
}

func (t orchestrationDispatchTool) Description() string {
	return "Dispatch exactly one orchestration sub-round for the current owner-led group."
}

func (t orchestrationDispatchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["public_once","private_once","end_group"]},
			"participant_ids":{"type":"array","items":{"type":"string"}},
			"order":{"type":"string","enum":["sequential","parallel"]},
			"instruction":{"type":"string"}
		},
		"required":["action"],
		"additionalProperties":false
	}`)
}

func (t orchestrationDispatchTool) Execute(_ context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	var req orchestrationDispatchRequest
	if err := json.Unmarshal(argsJSON, &req); err != nil {
		return "", fmt.Errorf("decode dispatch args: %w", err)
	}
	normalized, err := groupdomain.DispatchValidator{}.Validate(
		req,
		groupdomain.NewGroupRef(t.groupNode),
		t.memberOrder,
	)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func (t orchestrationDispatchTool) InterpretResult(output string) tools.ExecuteMeta {
	var req orchestrationDispatchRequest
	if err := json.Unmarshal([]byte(output), &req); err != nil {
		return tools.ExecuteMeta{}
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return tools.ExecuteMeta{}
	}
	return tools.ExecuteMeta{
		Iteration: &tools.IterationHandoffSignal{
			Did:       string(payload),
			Remaining: "owner dispatch submitted",
		},
	}
}

func (r orchestrationTaskRunner) newOwnerDispatchTool(groupNode OrchestrationNode, memberOrder []string) tools.Tool {
	return orchestrationDispatchTool{
		groupNode:   groupNode,
		memberOrder: append([]string(nil), memberOrder...),
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}

func buildOwnerControlPrompt(
	basePrompt string,
	ownerNode OrchestrationNode,
	groupNode OrchestrationNode,
	plan orchestrationExecutionPlan,
	publicTranscript orchestrationTranscript,
	lastDispatch orchestrationDispatchResult,
	round int,
) string {
	memberLines := make([]string, 0, len(plan.groupMember[groupNode.ID]))
	for _, memberID := range plan.groupMember[groupNode.ID] {
		memberNode := plan.nodes[memberID]
		title := memberID
		if memberNode.Agent != nil && strings.TrimSpace(memberNode.Agent.Title) != "" {
			title = memberNode.Agent.Title
		}
		memberLines = append(memberLines, fmt.Sprintf("- %s: %s", memberID, title))
	}
	ownerPrelude := buildOwnerPromptPrelude(basePrompt, ownerNode.Agent.Message)
	return strings.TrimSpace(fmt.Sprintf(
		"%s\n\n你是当前群组的群主，只能通过工具 `%s` 做调度，不允许自由聊天。\n\n当前群组成员：\n%s\n\n当前公开 transcript：\n%s\n\n上一轮 dispatch 结果：\n%s\n\n本轮是第 %d 次群主指派。你必须调用一次 `%s`，选择 public_once、private_once 或 end_group。",
		ownerPrelude,
		orchestrationDispatchToolName,
		strings.Join(memberLines, "\n"),
		publicTranscript.Format(),
		formatOwnerLastDispatch(lastDispatch),
		round,
		orchestrationDispatchToolName,
	))
}

func buildOwnerPromptPrelude(basePrompt string, ownerMessage string) string {
	parts := make([]string, 0, 2)
	if prompt := strings.TrimSpace(basePrompt); prompt != "" {
		parts = append(parts, prompt)
	}
	if message := strings.TrimSpace(ownerMessage); message != "" {
		parts = append(parts, "群主节点开场指令：\n"+message)
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func buildOwnerControlUserPrompt(round int) string {
	return fmt.Sprintf("开始第 %d 次群主调度。必须调用 orchestration_dispatch。", round)
}

func formatOwnerLastDispatch(dispatch orchestrationDispatchResult) string {
	if strings.TrimSpace(dispatch.Action) == "" {
		return "(none)"
	}
	return fmt.Sprintf("action=%s order=%s participants=%s instruction=%s",
		dispatch.Action,
		dispatch.Order,
		strings.Join(dispatch.ParticipantIDs, ","),
		dispatch.Instruction,
	)
}

func selectGroupFailure(results []orchestrationMemberResult) *orchestrationMemberResult {
	for index := range results {
		if results[index].Status == taskRunStatusAwaitingHuman {
			return &results[index]
		}
	}
	for index := range results {
		if results[index].Status == taskRunStatusError && results[index].Error != context.Canceled.Error() {
			return &results[index]
		}
	}
	for index := range results {
		if results[index].Status != taskRunStatusSuccess {
			return &results[index]
		}
	}
	return nil
}

func cloneGroupSessionIDs(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
