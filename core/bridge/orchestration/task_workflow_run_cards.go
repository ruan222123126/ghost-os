package orchestration

import (
	"context"
	"time"

	"ghost-os/bridge/llm"
	appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

type workflowTaskRunCardObserver struct {
	recorder *taskRunCardRecorder
}

type workflowTaskRunCardHandle struct {
	inner *taskRunCardHandle
}

func newWorkflowTaskRunCardObserver(ctx context.Context) appworkflows.CardObserver {
	recorder := taskRunCardRecorderFromContext(ctx)
	if recorder == nil {
		return nil
	}
	return workflowTaskRunCardObserver{recorder: recorder}
}

func (o workflowTaskRunCardObserver) StartCard(
	ctx context.Context,
	req appworkflows.CardStartRequest,
) (appworkflows.CardHandle, error) {
	handle, err := o.recorder.StartCard(ctx, taskRunCardStartInput{
		kind:      req.Kind,
		title:     req.Title,
		nodeID:    req.NodeID,
		nodeType:  req.NodeType,
		branchID:  req.BranchID,
		iteration: req.Iteration,
		startedAt: req.StartedAt,
	})
	if err != nil {
		return nil, err
	}
	return workflowTaskRunCardHandle{inner: handle}, nil
}

func (h workflowTaskRunCardHandle) Finish(
	ctx context.Context,
	req appworkflows.CardFinishRequest,
) error {
	return h.inner.Finish(ctx, taskRunCardFinishInput{
		status:     req.Status,
		preview:    req.Preview,
		errorText:  req.Error,
		finalText:  req.FinalText,
		finishedAt: req.FinishedAt,
	})
}

func (a taskExecutorAdapter) executeWorkflowAgent(
	ctx context.Context,
	req appworkflows.AgentRequest,
) bridgeTasks.ExecutionResult {
	runner := a.sessionAgentRunner()
	if runner == nil {
		return bridgeTasks.ExecutionResult{
			Status: taskRunStatusError,
			Error:  "workflow agent runner is not configured",
		}
	}
	recorder := taskRunCardRecorderFromContext(ctx)
	if req.RuntimeOverrides != nil {
		overrideRunner, ok := a.workflowStreamOverrideRunner()
		if !ok && a.service != nil && a.service.agentRunner != nil {
			return bridgeTasks.ExecutionResult{
				Status: taskRunStatusError,
				Error:  "workflow agent runtime overrides require a streaming runner",
			}
		}
		if ok {
			return executeWorkflowAgentWithOverrides(ctx, overrideRunner, recorder, req)
		}
	}
	if a.service != nil && a.service.agentRunner != nil && req.RuntimeOverrides == nil {
		return a.executeWorkflowAgentWithServiceRunner(ctx, req, recorder)
	}

	input := llm.Message{Role: llm.RoleUser, Text: req.Message}
	turn, err := runner.prepareTurnWithRuntimeOverrides(ctx, input, "", req.TraceID, req.RuntimeOverrides)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	defer turn.close()

	handle, startErr := startWorkflowAgentCard(ctx, recorder, req, turn.currentSessionID())
	if startErr != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: startErr.Error()}
	}
	response, sessionID, runErr := runPreparedTurnStream(ctx, turn, input, workflowAgentSink(handle))
	result := executionResultFromStreamOutcome(ctx, response, sessionID, runErr)
	if finishErr := finishWorkflowAgentCard(ctx, handle, result); finishErr != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: finishErr.Error()}
	}
	return result
}

func executeWorkflowAgentWithOverrides(
	ctx context.Context,
	runner SessionTurnStreamRunnerWithOverrides,
	recorder *taskRunCardRecorder,
	req appworkflows.AgentRequest,
) bridgeTasks.ExecutionResult {
	handle, err := startWorkflowAgentCard(ctx, recorder, req, "")
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	response, sessionID, runErr := runner.RunTurnStreamWithOverrides(
		ctx,
		req.Message,
		"",
		req.TraceID,
		workflowAgentSink(handle),
		req.RuntimeOverrides,
	)
	result := executionResultFromStreamOutcome(ctx, response, sessionID, runErr)
	if finishErr := finishWorkflowAgentCard(ctx, handle, result); finishErr != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: finishErr.Error()}
	}
	return result
}

func (a taskExecutorAdapter) executeWorkflowAgentWithServiceRunner(
	ctx context.Context,
	req appworkflows.AgentRequest,
	recorder *taskRunCardRecorder,
) bridgeTasks.ExecutionResult {
	handle, err := startWorkflowAgentCard(ctx, recorder, req, "")
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	response, sessionID, runErr := a.service.agentRunner.RunTurnStream(
		ctx,
		req.Message,
		"",
		req.TraceID,
		workflowAgentSink(handle),
	)
	result := executionResultFromStreamOutcome(ctx, response, sessionID, runErr)
	if finishErr := finishWorkflowAgentCard(ctx, handle, result); finishErr != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: finishErr.Error()}
	}
	return result
}

func startWorkflowAgentCard(
	ctx context.Context,
	recorder *taskRunCardRecorder,
	req appworkflows.AgentRequest,
	sourceSessionID string,
) (*taskRunCardHandle, error) {
	if recorder == nil {
		return nil, nil
	}
	return recorder.StartCard(ctx, taskRunCardStartInput{
		kind:            bridgeTasks.RunCardKindWorkflowAgent,
		title:           req.NodeID,
		nodeID:          req.NodeID,
		nodeType:        req.NodeType,
		branchID:        req.BranchID,
		iteration:       req.Iteration,
		sourceSessionID: sourceSessionID,
		startedAt:       time.Now().UTC(),
	})
}

func finishWorkflowAgentCard(
	ctx context.Context,
	handle *taskRunCardHandle,
	result bridgeTasks.ExecutionResult,
) error {
	if handle == nil {
		return nil
	}
	return handle.Finish(ctx, taskRunCardFinishInput{
		status:          result.Status,
		preview:         result.ResponsePreview,
		errorText:       result.Error,
		sourceSessionID: result.SessionIDOutput,
		finishedAt:      time.Now().UTC(),
	})
}

func workflowAgentSink(handle *taskRunCardHandle) streaming.Sink {
	if handle == nil {
		return nil
	}
	return newTaskRunCardStreamSink(handle)
}

func (a taskExecutorAdapter) workflowStreamOverrideRunner() (SessionTurnStreamRunnerWithOverrides, bool) {
	if a.service == nil || a.service.agentRunner == nil {
		return nil, false
	}
	runner, ok := a.service.agentRunner.(SessionTurnStreamRunnerWithOverrides)
	return runner, ok
}
