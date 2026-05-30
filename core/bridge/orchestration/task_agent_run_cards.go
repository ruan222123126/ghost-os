package orchestration

import (
	"context"
	"errors"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

func (a taskExecutorAdapter) executeTrackedAgentTask(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
	startedAt time.Time,
) bridgeTasks.ExecutionResult {
	recorder := taskRunCardRecorderFromContext(ctx)
	if recorder == nil {
		return bridgeTasks.ExecutionResult{
			Status: taskRunStatusError,
			Error:  "task run card recorder is not configured",
		}
	}
	handle, err := recorder.StartCard(ctx, taskRunCardStartInput{
		kind:            bridgeTasks.RunCardKindAgentTask,
		title:           taskRunCardTaskTitle(task),
		nodeID:          taskKindAgentMessage,
		nodeType:        taskKindAgentMessage,
		sourceSessionID: strings.TrimSpace(task.SessionID),
		startedAt:       startedAt,
	})
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}

	response, sessionID, runErr := a.runAgentTaskStream(
		ctx,
		task,
		traceID,
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
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: finishErr.Error()}
	}
	return result
}

func (a taskExecutorAdapter) runAgentTaskStream(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	if a.service != nil && a.service.agentRunner != nil && task.RuntimeOverrides == nil {
		return a.service.agentRunner.RunTurnStream(
			ctx,
			task.Message,
			strings.TrimSpace(task.SessionID),
			traceID,
			sink,
		)
	}
	runner := a.sessionAgentRunner()
	if runner == nil {
		return "", strings.TrimSpace(task.SessionID), errors.New("task agent runner is not configured")
	}
	return runner.RunTurnStreamInputWithOverrides(
		ctx,
		llm.Message{Role: llm.RoleUser, Text: task.Message},
		strings.TrimSpace(task.SessionID),
		traceID,
		sink,
		cloneTaskRuntimeOverrides(task.RuntimeOverrides),
	)
}

func (a taskExecutorAdapter) sessionAgentRunner() *SessionAgentRunner {
	if a.service == nil {
		return nil
	}
	return NewSessionAgentRunner(
		a.service.runtimeFactory,
		a.service.configStore,
		a.service.sessionStore,
		a.service.runRegistry,
	)
}

func executionResultFromStreamOutcome(
	ctx context.Context,
	response string,
	sessionID string,
	runErr error,
) bridgeTasks.ExecutionResult {
	outputSessionID := strings.TrimSpace(sessionID)
	switch {
	case runErr == nil:
		return bridgeTasks.ExecutionResult{
			Status:          taskRunStatusSuccess,
			SessionIDOutput: outputSessionID,
			ResponsePreview: truncateRunes(strings.TrimSpace(response), maxTaskResponsePreviewRunes),
		}
	case errors.Is(ctx.Err(), context.Canceled), errors.Is(runErr, context.Canceled):
		return bridgeTasks.ExecutionResult{
			Status:          taskRunStatusCancelled,
			SessionIDOutput: outputSessionID,
			Error:           context.Canceled.Error(),
		}
	default:
		var awaitingErr *agent.ErrAwaitingHuman
		if errors.As(runErr, &awaitingErr) {
			return bridgeTasks.ExecutionResult{
				Status:          taskRunStatusAwaitingHuman,
				SessionIDOutput: outputSessionID,
				ResponsePreview: truncateRunes(strings.TrimSpace(awaitingErr.Prompt), maxTaskResponsePreviewRunes),
			}
		}
		return bridgeTasks.ExecutionResult{
			Status:          taskRunStatusError,
			SessionIDOutput: outputSessionID,
			Error:           runErr.Error(),
		}
	}
}

func taskRunCardTaskTitle(task ScheduledTask) string {
	if name := strings.TrimSpace(task.Name); name != "" {
		return name
	}
	if id := strings.TrimSpace(task.ID); id != "" {
		return id
	}
	return taskKindAgentMessage
}
