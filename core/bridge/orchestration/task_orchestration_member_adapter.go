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
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeTasks "ghost-os/bridge/tasks"
)

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
