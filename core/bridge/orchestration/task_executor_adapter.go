package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/session"
	bridgeTasks "ghost-os/bridge/tasks"
)

type taskExecutorAdapter struct {
	service *bridgeService
}

type taskNodeResultTimestamps struct {
	startedAt  time.Time
	finishedAt time.Time
}

func (a taskExecutorAdapter) Execute(ctx context.Context, task ScheduledTask, traceID string) bridgeTasks.ExecutionResult {
	result := a.executeTask(ctx, task, traceID)
	return a.attachTaskRunTranscript(ctx, task, traceID, result)
}

func (a taskExecutorAdapter) PrepareRunSession(
	ctx context.Context,
	task ScheduledTask,
	_ string,
) (bridgeTasks.RunSession, error) {
	if id := strings.TrimSpace(task.SessionID); id != "" {
		return bridgeTasks.RunSession{SessionID: id}, nil
	}
	if !shouldPrecreateTaskRunSession(task.TaskKind) {
		return bridgeTasks.RunSession{}, nil
	}
	if err := ctx.Err(); err != nil {
		return bridgeTasks.RunSession{}, err
	}
	return a.saveTaskRunSession(task, "")
}

func (a taskExecutorAdapter) saveTaskRunSession(
	task ScheduledTask,
	systemPrompt string,
) (bridgeTasks.RunSession, error) {
	if a.service == nil || a.service.sessionStore == nil {
		return bridgeTasks.RunSession{}, errors.New("task run session store is not configured")
	}
	sess := session.NewSession(systemPrompt)
	sess.Title = taskRunSessionTitle(task)
	if err := a.service.sessionStore.Save(sess); err != nil {
		return bridgeTasks.RunSession{}, err
	}
	return bridgeTasks.RunSession{SessionID: sess.ID}, nil
}

func shouldPrecreateTaskRunSession(taskKind string) bool {
	switch normalizeTaskKind(taskKind) {
	case taskKindAgentMessage, taskKindWorkflow, taskKindOrchestration:
		return true
	default:
		return false
	}
}

func taskRunSessionTitle(task ScheduledTask) string {
	kind := normalizeTaskKind(task.TaskKind)
	name := strings.TrimSpace(task.Name)
	if name == "" {
		name = strings.TrimSpace(task.ID)
	}
	if name == "" {
		return kind + " run"
	}
	return kind + ": " + name
}

func (a taskExecutorAdapter) executeTask(ctx context.Context, task ScheduledTask, traceID string) bridgeTasks.ExecutionResult {
	switch kind := normalizeTaskKind(task.TaskKind); kind {
	case taskKindWorkflow:
		return a.executeWorkflowTask(ctx, task, traceID)
	case taskKindOrchestration:
		return a.executeOrchestrationTask(ctx, task, traceID)
	case taskKindSystemAction:
		return a.executeSystemTask(ctx, task, traceID)
	case taskKindAgentMessage:
		return a.executeAgentTask(ctx, task, traceID)
	default:
		return bridgeTasks.ExecutionResult{
			Status: taskRunStatusError,
			Error:  fmt.Sprintf("unsupported task_kind %q", kind),
		}
	}
}

func (a taskExecutorAdapter) executeAgentTask(ctx context.Context, task ScheduledTask, traceID string) bridgeTasks.ExecutionResult {
	timestamps := taskNodeResultTimestamps{
		startedAt:  time.Now().UTC(),
		finishedAt: time.Now().UTC(),
	}
	if task.AgentMode == taskAgentModeRelay {
		return a.executeRelayAgentTask(ctx, task, traceID, timestamps.startedAt)
	}
	result := a.runAgentAction(ctx, agentParams{
		Message:   task.Message,
		SessionID: task.SessionID,
	}, cloneTaskRuntimeOverrides(task.RuntimeOverrides), traceID)
	timestamps.finishedAt = time.Now().UTC()
	result.NodeResults = []bridgeTasks.RunNodeResult{
		buildAgentMessageNodeResult(task, result, timestamps),
	}
	return result
}

func (a taskExecutorAdapter) executeRelayAgentTask(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
	startedAt time.Time,
) bridgeTasks.ExecutionResult {
	if a.service == nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "task executor service is not configured"}
	}
	if task.Relay == nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "relay config is required"}
	}
	result, err := newRelayModeRunner(a.service).ExecuteTask(ctx, task, traceID)
	execution := relayTaskExecutionResult(result, err)
	execution.NodeResults = []bridgeTasks.RunNodeResult{
		buildRelayAgentNodeResult(relayAgentNodeResultInput{
			task:       task,
			execution:  execution,
			result:     result,
			startedAt:  startedAt,
			finishedAt: time.Now().UTC(),
		}),
	}
	return execution
}

func relayTaskExecutionResult(result relayModeResult, err error) bridgeTasks.ExecutionResult {
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	status := taskRunStatusSuccess
	if result.StoppedBy == relayModeStopMaxRounds {
		status = taskRunStatusIncomplete
	}
	return bridgeTasks.ExecutionResult{
		Status:          status,
		SessionIDOutput: strings.TrimSpace(result.SessionID),
		ResponsePreview: truncateRunes(strings.TrimSpace(result.Message), maxTaskResponsePreviewRunes),
	}
}

func (a taskExecutorAdapter) runAgentAction(
	ctx context.Context,
	params agentParams,
	runtimeOverrides *TaskRuntimeOverrides,
	traceID string,
) bridgeTasks.ExecutionResult {
	if a.service == nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "task executor service is not configured"}
	}
	result, err := a.service.executeAgentActionWithRuntimeOverrides(ctx, params, runtimeOverrides, traceID)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	return taskExecutionResultFromAgentPayload(result.Payload)
}

func taskExecutionResultFromAgentPayload(payload any) bridgeTasks.ExecutionResult {
	switch typed := payload.(type) {
	case agentResponse:
		return bridgeTasks.ExecutionResult{
			Status:          taskRunStatusSuccess,
			SessionIDOutput: strings.TrimSpace(typed.SessionID),
			ResponsePreview: truncateRunes(strings.TrimSpace(typed.Message), maxTaskResponsePreviewRunes),
		}
	case askHumanAwaitingResponse:
		return bridgeTasks.ExecutionResult{
			Status:          taskRunStatusAwaitingHuman,
			SessionIDOutput: strings.TrimSpace(typed.SessionID),
			ResponsePreview: truncateRunes(strings.TrimSpace(typed.Prompt), maxTaskResponsePreviewRunes),
		}
	default:
		return bridgeTasks.ExecutionResult{Status: taskRunStatusSuccess}
	}
}

func buildAgentMessageNodeResult(task ScheduledTask, result bridgeTasks.ExecutionResult, timestamps taskNodeResultTimestamps) bridgeTasks.RunNodeResult {
	status := strings.TrimSpace(result.Status)
	if status == "" {
		status = taskRunStatusError
	}
	input := map[string]any{
		"message":    task.Message,
		"session_id": strings.TrimSpace(task.SessionID),
		"agent_mode": strings.TrimSpace(task.AgentMode),
	}
	if task.Relay != nil {
		input["relay"] = bridgeTasks.CloneTaskRelayConfig(task.Relay)
	}
	runtimePayload := map[string]any{}
	if runtimeOverrides := cloneTaskRuntimeOverrides(task.RuntimeOverrides); runtimeOverrides != nil {
		runtimePayload = taskRuntimeOverrideSnapshot(runtimeOverrides)
	}
	input["runtime_overrides"] = runtimePayload
	output := map[string]any{
		"session_id_output": strings.TrimSpace(result.SessionIDOutput),
		"response_preview":  strings.TrimSpace(result.ResponsePreview),
	}
	if strings.TrimSpace(result.Error) != "" {
		output["error"] = strings.TrimSpace(result.Error)
	}
	return bridgeTasks.RunNodeResult{
		NodeID:       taskKindAgentMessage,
		NodeType:     taskKindAgentMessage,
		Status:       status,
		StartedAt:    timestamps.startedAt,
		FinishedAt:   timestamps.finishedAt,
		CompletedSeq: 1,
		Input:        input,
		Output:       output,
		Preview:      strings.TrimSpace(result.ResponsePreview),
		Error:        strings.TrimSpace(result.Error),
	}
}

func taskRuntimeOverrideSnapshot(runtimeOverrides *TaskRuntimeOverrides) map[string]any {
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

func (a taskExecutorAdapter) executeSystemTask(_ context.Context, task ScheduledTask, _ string) bridgeTasks.ExecutionResult {
	if a.service == nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "task executor service is not configured"}
	}
	return bridgeTasks.ExecutionResult{
		Status: taskRunStatusError,
		Error:  "unsupported system action: " + strings.TrimSpace(task.Action),
	}
}
