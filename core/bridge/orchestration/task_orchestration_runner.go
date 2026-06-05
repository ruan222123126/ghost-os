package orchestration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	agentadapter "ghost-os/bridge/orchestration/internal/adapters/agent"
	appagentturn "ghost-os/bridge/orchestration/internal/app/agentturn"
	apporchestrations "ghost-os/bridge/orchestration/internal/app/orchestrations"
	groupdomain "ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeTasks "ghost-os/bridge/tasks"
)

type orchestrationExecutionPlan struct {
	nodes        map[string]OrchestrationNode
	entryGroupID string
	controlNext  map[string]string
	groupMember  map[string][]string
}

func (a taskExecutorAdapter) executeOrchestrationTask(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
) bridgeTasks.ExecutionResult {
	if _, err := buildOrchestrationExecutionPlan(task.Orchestration); err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	if a.service == nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "task executor service is not configured"}
	}
	return newOrchestrationTaskRunner(a).execute(ctx, task.Orchestration, traceID)
}

type orchestrationTaskRunner struct {
	adapter taskExecutorAdapter
}

func newOrchestrationTaskRunner(adapter taskExecutorAdapter) orchestrationTaskRunner {
	return orchestrationTaskRunner{adapter: adapter}
}

func (r orchestrationTaskRunner) groupExecutor() apporchestrations.ModeGroupExecutor {
	return apporchestrations.ModeGroupExecutor{
		Standard: r.standardGroupExecutor(),
		Owner:    r.ownerGroupExecutor(),
	}
}

func (r orchestrationTaskRunner) standardGroupExecutor() apporchestrations.StandardGroupExecutor {
	return apporchestrations.StandardGroupExecutor{Dispatcher: r.roundDispatcher()}
}

func (r orchestrationTaskRunner) ownerGroupExecutor() apporchestrations.OwnerGroupExecutor {
	return apporchestrations.OwnerGroupExecutor{
		Decisions: r.ownerDecisionRunner(),
		Dispatches: apporchestrations.DispatchExecutor{
			Dispatcher: r.roundDispatcher(),
		},
	}
}

func (r orchestrationTaskRunner) roundDispatcher() apporchestrations.RoundDispatcher {
	return apporchestrations.RoundDispatcher{Members: r.memberRunner()}
}

func (r orchestrationTaskRunner) execute(
	ctx context.Context,
	definition *OrchestrationDefinition,
	traceID string,
) bridgeTasks.ExecutionResult {
	runner := apporchestrations.Runner{
		Planner: groupdomain.PlanBuilder{},
		Groups:  r.groupExecutor(),
		Mapper:  apporchestrations.ResultMapper{},
	}
	return runner.Execute(ctx, apporchestrations.ExecuteCommand{
		Definition: definition,
		TraceID:    traceID,
	})
}

func validateOrchestrationTaskDefinition(task *ScheduledTask) error {
	if task.Orchestration == nil {
		return fmt.Errorf("%w: orchestration is required for orchestration task", ErrInvalidTaskConfig)
	}
	if task.Name == "" {
		return fmt.Errorf("%w: orchestration name is required", ErrInvalidTaskConfig)
	}
	if task.Message != "" || task.SessionID != "" || task.Action != "" {
		return fmt.Errorf("%w: orchestration task does not allow message, session_id, or action", ErrInvalidTaskConfig)
	}
	if task.RuntimeOverrides != nil || task.Workflow != nil || len(task.ActionParams) > 0 {
		return fmt.Errorf("%w: orchestration task only allows name, orchestration, and schedule fields", ErrInvalidTaskConfig)
	}
	if err := normalizeOrchestrationDefinitionRuntimeOverrides(task.Orchestration); err != nil {
		return err
	}
	_, err := buildOrchestrationExecutionPlan(task.Orchestration)
	return err
}

func normalizeOrchestrationDefinitionRuntimeOverrides(definition *OrchestrationDefinition) error {
	if definition == nil {
		return nil
	}
	for index := range definition.Nodes {
		node := &definition.Nodes[index]
		if node.Type != orchestrationNodeTypeAgent || node.Agent == nil {
			continue
		}
		overrides, err := normalizeOrchestrationAgentRuntimeOverrides(node.Agent.RuntimeOverrides)
		if err != nil {
			return fmt.Errorf("%w: orchestration agent node %q %v", ErrInvalidTaskConfig, node.ID, err)
		}
		node.Agent.RuntimeOverrides = overrides
	}
	return nil
}

func buildOrchestrationExecutionPlan(definition *OrchestrationDefinition) (orchestrationExecutionPlan, error) {
	plan, err := groupdomain.PlanBuilder{}.Build(definition)
	if err != nil {
		return orchestrationExecutionPlan{}, err
	}
	return orchestrationExecutionPlan{
		nodes:        plan.Nodes,
		entryGroupID: plan.EntryGroupID,
		controlNext:  plan.ControlNext,
		groupMember:  plan.GroupMembers,
	}, nil
}

type orchestrationMemberActionInvoker struct {
	service *bridgeService
}

func (i orchestrationMemberActionInvoker) ExecuteAgentAction(
	ctx context.Context,
	req ports.AgentActionRequest,
) (ports.AgentActionPayload, error) {
	if i.service == nil {
		return ports.AgentActionPayload{}, fmt.Errorf("task executor service is not configured")
	}
	runner := NewSessionAgentRunner(
		i.service.runtimeFactory,
		i.service.configStore,
		i.service.sessionStore,
		i.service.runRegistry,
	)
	recorder := taskRunCardRecorderFromContext(ctx)
	if recorder == nil {
		return ports.AgentActionPayload{}, fmt.Errorf("orchestration task run card recorder is not configured")
	}
	input := llm.Message{Role: llm.RoleUser, Text: req.Message}
	if i.service.agentRunner != nil && req.RuntimeOverrides == nil {
		return i.executeMemberWithServiceRunner(ctx, req, recorder)
	}
	turn, err := runner.prepareTurnWithRuntimeOverrides(ctx, input, req.SessionID, req.TraceID, req.RuntimeOverrides)
	if err != nil {
		return ports.AgentActionPayload{}, err
	}
	defer turn.close()

	handle, err := recorder.StartCard(ctx, taskRunCardStartInput{
		kind:            bridgeTasks.RunCardKindOrchestrationMember,
		title:           req.Title,
		nodeID:          req.AgentID,
		nodeType:        "agent",
		round:           req.Round,
		sourceSessionID: turn.currentSessionID(),
		startedAt:       time.Now().UTC(),
	})
	if err != nil {
		return ports.AgentActionPayload{}, err
	}

	response, sessionID, runErr := runPreparedTurnStream(ctx, turn, input, newTaskRunCardStreamSink(handle))
	result := executionResultFromStreamOutcome(ctx, response, sessionID, runErr)
	if finishErr := handle.Finish(ctx, taskRunCardFinishInput{
		status:          result.Status,
		preview:         result.ResponsePreview,
		errorText:       result.Error,
		sourceSessionID: result.SessionIDOutput,
		finishedAt:      time.Now().UTC(),
	}); finishErr != nil {
		return ports.AgentActionPayload{}, finishErr
	}
	return orchestrationTaskAgentActionPayload(response, sessionID, runErr)
}

func (i orchestrationMemberActionInvoker) executeMemberWithServiceRunner(
	ctx context.Context,
	req ports.AgentActionRequest,
	recorder *taskRunCardRecorder,
) (ports.AgentActionPayload, error) {
	handle, err := recorder.StartCard(ctx, taskRunCardStartInput{
		kind:            bridgeTasks.RunCardKindOrchestrationMember,
		title:           req.Title,
		nodeID:          req.AgentID,
		nodeType:        "agent",
		round:           req.Round,
		sourceSessionID: req.SessionID,
		startedAt:       time.Now().UTC(),
	})
	if err != nil {
		return ports.AgentActionPayload{}, err
	}
	response, sessionID, runErr := i.service.agentRunner.RunTurnStream(
		ctx,
		req.Message,
		req.SessionID,
		req.TraceID,
		newTaskRunCardStreamSink(handle),
	)
	result := executionResultFromStreamOutcome(ctx, response, sessionID, runErr)
	if finishErr := handle.Finish(ctx, taskRunCardFinishInput{
		status:          result.Status,
		preview:         result.ResponsePreview,
		errorText:       result.Error,
		sourceSessionID: result.SessionIDOutput,
		finishedAt:      time.Now().UTC(),
	}); finishErr != nil {
		return ports.AgentActionPayload{}, finishErr
	}
	return orchestrationTaskAgentActionPayload(response, sessionID, runErr)
}

func orchestrationTaskAgentActionPayload(
	response string,
	sessionID string,
	runErr error,
) (ports.AgentActionPayload, error) {
	if runErr == nil {
		return ports.AgentActionPayload{
			Kind:      ports.AgentActionPayloadSuccess,
			SessionID: sessionID,
			Message:   response,
		}, nil
	}
	var awaitingErr *agent.ErrAwaitingHuman
	if errors.As(runErr, &awaitingErr) {
		return ports.AgentActionPayload{
			Kind:      ports.AgentActionPayloadAwaiting,
			SessionID: sessionID,
			Prompt:    awaitingErr.Prompt,
		}, nil
	}
	return ports.AgentActionPayload{}, appagentturn.WrapErrorWithSessionID(runErr, sessionID)
}

func (r orchestrationTaskRunner) memberRunner() ports.MemberAgentRunner {
	return apporchestrations.MemberRunner{
		Executor: agentadapter.MemberAgentRunner{
			Invoker: orchestrationMemberActionInvoker{service: r.adapter.service},
		},
	}
}
