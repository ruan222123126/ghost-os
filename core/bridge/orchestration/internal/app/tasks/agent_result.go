package tasks

import (
	"context"
	"errors"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	sharedtext "ghost-os/bridge/orchestration/internal/shared/text"
	"ghost-os/bridge/orchestration/internal/trace/runcards"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/taskdefs"
)

type AgentTaskStreamRunner interface {
	RunAgentTaskStream(
		ctx context.Context,
		task taskdefs.ScheduledTask,
		traceID string,
		sink streaming.Sink,
	) (string, string, error)
}

type AgentExecutor struct {
	StreamRunner AgentTaskStreamRunner
	RelayRunner  RelayTaskRunner
	Now          func() time.Time
}

type AgentRunCardExecutor struct {
	StreamRunner AgentTaskStreamRunner
}

type NodeResultTimestamps struct {
	StartedAt  time.Time
	FinishedAt time.Time
}

func (e AgentExecutor) ExecuteTask(
	ctx context.Context,
	task taskdefs.ScheduledTask,
	traceID string,
) taskdefs.ExecutionResult {
	startedAt := e.now()
	if taskdefs.NormalizeAgentMode(task.AgentMode) == taskdefs.AgentModeRelay {
		return e.executeRelay(ctx, task, traceID, startedAt)
	}
	result := AgentRunCardExecutor{StreamRunner: e.StreamRunner}.ExecuteTracked(ctx, task, traceID, startedAt)
	result.NodeResults = []taskdefs.RunNodeResult{
		BuildAgentMessageNodeResult(task, result, NodeResultTimestamps{
			StartedAt:  startedAt,
			FinishedAt: e.now(),
		}),
	}
	return result
}

func (e AgentExecutor) executeRelay(
	ctx context.Context,
	task taskdefs.ScheduledTask,
	traceID string,
	startedAt time.Time,
) taskdefs.ExecutionResult {
	if e.RelayRunner == nil {
		return taskdefs.ExecutionResult{
			Status: taskdefs.RunStatusError,
			Error:  "task relay runner is not configured",
		}
	}
	result, err := e.RelayRunner.ExecuteRelayTask(ctx, task, traceID)
	execution := ExecutionResultFromRelayOutcome(result, err)
	if IsRelayPreflightError(err) {
		return execution
	}
	execution.NodeResults = []taskdefs.RunNodeResult{
		BuildRelayAgentMessageNodeResult(task, execution, result, NodeResultTimestamps{
			StartedAt:  startedAt,
			FinishedAt: e.now(),
		}),
	}
	return execution
}

func (e AgentExecutor) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}

func (e AgentRunCardExecutor) ExecuteTracked(
	ctx context.Context,
	task taskdefs.ScheduledTask,
	traceID string,
	startedAt time.Time,
) taskdefs.ExecutionResult {
	recorder := runcards.RecorderFromContext(ctx)
	if recorder == nil {
		return taskdefs.ExecutionResult{
			Status: taskdefs.RunStatusError,
			Error:  "task run card recorder is not configured",
		}
	}
	if e.StreamRunner == nil {
		return taskdefs.ExecutionResult{
			Status: taskdefs.RunStatusError,
			Error:  "task agent runner is not configured",
		}
	}
	handle, err := recorder.StartCard(ctx, runcards.StartInput{
		Kind:            taskdefs.RunCardKindAgentTask,
		Title:           AgentRunCardTaskTitle(task),
		NodeID:          taskdefs.KindAgentMessage,
		NodeType:        taskdefs.KindAgentMessage,
		SourceSessionID: strings.TrimSpace(task.SessionID),
		StartedAt:       startedAt,
	})
	if err != nil {
		return taskdefs.ExecutionResult{Status: taskdefs.RunStatusError, Error: err.Error()}
	}

	response, sessionID, runErr := e.StreamRunner.RunAgentTaskStream(
		ctx,
		task,
		traceID,
		runcards.NewStreamSink(handle),
	)
	result := ExecutionResultFromStreamOutcome(ctx, response, sessionID, runErr)
	if finishErr := handle.Finish(ctx, runcards.FinishInput{
		Status:          result.Status,
		Preview:         result.ResponsePreview,
		ErrorText:       result.Error,
		SourceSessionID: result.SessionIDOutput,
		FinishedAt:      time.Now().UTC(),
	}); finishErr != nil {
		return taskdefs.ExecutionResult{Status: taskdefs.RunStatusError, Error: finishErr.Error()}
	}
	return result
}

func BuildAgentMessageNodeResult(
	task taskdefs.ScheduledTask,
	result taskdefs.ExecutionResult,
	timestamps NodeResultTimestamps,
) taskdefs.RunNodeResult {
	status := strings.TrimSpace(result.Status)
	if status == "" {
		status = taskdefs.RunStatusError
	}
	return taskdefs.RunNodeResult{
		NodeID:       taskdefs.KindAgentMessage,
		NodeType:     taskdefs.KindAgentMessage,
		Status:       status,
		StartedAt:    timestamps.StartedAt,
		FinishedAt:   timestamps.FinishedAt,
		CompletedSeq: 1,
		Input:        AgentMessageNodeInput(task),
		Output:       AgentMessageNodeOutput(result),
		Preview:      strings.TrimSpace(result.ResponsePreview),
		Error:        strings.TrimSpace(result.Error),
	}
}

func AgentMessageNodeInput(task taskdefs.ScheduledTask) map[string]any {
	input := map[string]any{
		"message":    task.Message,
		"session_id": strings.TrimSpace(task.SessionID),
		"agent_mode": strings.TrimSpace(task.AgentMode),
	}
	if task.Relay != nil {
		input["relay"] = taskdefs.CloneTaskRelayConfig(task.Relay)
	}
	runtimePayload := map[string]any{}
	if runtimeOverrides := taskdefs.CloneTaskRuntimeOverrides(task.RuntimeOverrides); runtimeOverrides != nil {
		runtimePayload = RuntimeOverrideSnapshot(runtimeOverrides)
	}
	input["runtime_overrides"] = runtimePayload
	return input
}

func AgentMessageNodeOutput(result taskdefs.ExecutionResult) map[string]any {
	output := map[string]any{
		"session_id_output": strings.TrimSpace(result.SessionIDOutput),
		"response_preview":  strings.TrimSpace(result.ResponsePreview),
	}
	if strings.TrimSpace(result.Error) != "" {
		output["error"] = strings.TrimSpace(result.Error)
	}
	return output
}

func RuntimeOverrideSnapshot(runtimeOverrides *taskdefs.TaskRuntimeOverrides) map[string]any {
	if runtimeOverrides == nil {
		return map[string]any{}
	}
	snapshot := map[string]any{
		"provider_name": strings.TrimSpace(runtimeOverrides.ProviderName),
		"model":         strings.TrimSpace(runtimeOverrides.Model),
		"system_prompt": strings.TrimSpace(runtimeOverrides.SystemPrompt),
		"preset_id":     strings.TrimSpace(runtimeOverrides.PresetID),
	}
	if runtimeOverrides.ToolAllowlistOnly != nil {
		snapshot["tool_allowlist_only"] = *runtimeOverrides.ToolAllowlistOnly
	}
	if runtimeOverrides.MaxTurns != nil {
		snapshot["max_turns"] = *runtimeOverrides.MaxTurns
	}
	if runtimeOverrides.ToolAllowlist != nil {
		snapshot["tool_allowlist"] = append([]string(nil), runtimeOverrides.ToolAllowlist...)
	}
	return snapshot
}

func AgentRunCardTaskTitle(task taskdefs.ScheduledTask) string {
	if name := strings.TrimSpace(task.Name); name != "" {
		return name
	}
	if id := strings.TrimSpace(task.ID); id != "" {
		return id
	}
	return taskdefs.KindAgentMessage
}

func ExecutionResultFromStreamOutcome(
	ctx context.Context,
	response string,
	sessionID string,
	runErr error,
) taskdefs.ExecutionResult {
	outputSessionID := strings.TrimSpace(sessionID)
	switch {
	case runErr == nil:
		return taskdefs.ExecutionResult{
			Status:          taskdefs.RunStatusSuccess,
			SessionIDOutput: outputSessionID,
			ResponsePreview: sharedtext.TruncateRunes(response, taskdefs.MaxResponsePreviewRunes),
		}
	case errors.Is(ctx.Err(), context.Canceled), errors.Is(runErr, context.Canceled):
		return taskdefs.ExecutionResult{
			Status:          taskdefs.RunStatusCancelled,
			SessionIDOutput: outputSessionID,
		}
	default:
		var awaitingErr *agent.ErrAwaitingHuman
		if errors.As(runErr, &awaitingErr) {
			return taskdefs.ExecutionResult{
				Status:          taskdefs.RunStatusAwaitingHuman,
				SessionIDOutput: outputSessionID,
				ResponsePreview: sharedtext.TruncateRunes(awaitingErr.Prompt, taskdefs.MaxResponsePreviewRunes),
			}
		}
		return taskdefs.ExecutionResult{
			Status:          taskdefs.RunStatusError,
			SessionIDOutput: outputSessionID,
			Error:           runErr.Error(),
		}
	}
}
