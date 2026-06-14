package tasks

import (
	"context"
	"fmt"
	"strings"

	"ghost-os/bridge/taskdefs"
	bridgeTasks "ghost-os/bridge/tasks"
)

type RunSessionCreator interface {
	SaveTaskRunSession(task taskdefs.ScheduledTask, systemPrompt string) (bridgeTasks.RunSession, error)
}

func PrepareRunSession(
	ctx context.Context,
	task taskdefs.ScheduledTask,
	creator RunSessionCreator,
) (bridgeTasks.RunSession, error) {
	if id := strings.TrimSpace(task.SessionID); id != "" {
		return bridgeTasks.RunSession{SessionID: id}, nil
	}
	if !ShouldPrecreateRunSession(task.TaskKind) {
		return bridgeTasks.RunSession{}, nil
	}
	if err := ctx.Err(); err != nil {
		return bridgeTasks.RunSession{}, err
	}
	if creator == nil {
		return bridgeTasks.RunSession{}, fmt.Errorf("task run session store is not configured")
	}
	return creator.SaveTaskRunSession(task, "")
}

func ShouldPrecreateRunSession(taskKind string) bool {
	switch taskdefs.NormalizeTaskKind(taskKind) {
	case taskdefs.KindAgentMessage, taskdefs.KindWorkflow, taskdefs.KindOrchestration:
		return true
	default:
		return false
	}
}

func ShouldRecordRunCards(taskKind string) bool {
	return ShouldPrecreateRunSession(taskKind)
}

func RunSessionTitle(task taskdefs.ScheduledTask) string {
	kind := taskdefs.NormalizeTaskKind(task.TaskKind)
	name := strings.TrimSpace(task.Name)
	if name == "" {
		name = strings.TrimSpace(task.ID)
	}
	if name == "" {
		return kind + " run"
	}
	return kind + ": " + name
}

type KindExecutor interface {
	ExecuteTask(ctx context.Context, task taskdefs.ScheduledTask, traceID string) taskdefs.ExecutionResult
}

type KindExecutorFunc func(context.Context, taskdefs.ScheduledTask, string) taskdefs.ExecutionResult

func (f KindExecutorFunc) ExecuteTask(
	ctx context.Context,
	task taskdefs.ScheduledTask,
	traceID string,
) taskdefs.ExecutionResult {
	return f(ctx, task, traceID)
}

type TaskExecutionRunner struct {
	Workflow      KindExecutor
	Orchestration KindExecutor
	Agent         KindExecutor
}

type TaskExecutionCommand struct {
	Task    taskdefs.ScheduledTask
	TraceID string
}

func (r TaskExecutionRunner) Execute(
	ctx context.Context,
	cmd TaskExecutionCommand,
) taskdefs.ExecutionResult {
	kind := taskdefs.NormalizeTaskKind(cmd.Task.TaskKind)
	switch kind {
	case taskdefs.KindWorkflow:
		return executeWithKindRunner(ctx, r.Workflow, cmd, kind)
	case taskdefs.KindOrchestration:
		return executeWithKindRunner(ctx, r.Orchestration, cmd, kind)
	case taskdefs.KindAgentMessage:
		return executeWithKindRunner(ctx, r.Agent, cmd, kind)
	default:
		return taskdefs.ExecutionResult{
			Status: taskdefs.RunStatusError,
			Error:  fmt.Sprintf("unsupported task_kind %q", kind),
		}
	}
}

func executeWithKindRunner(
	ctx context.Context,
	runner KindExecutor,
	cmd TaskExecutionCommand,
	kind string,
) taskdefs.ExecutionResult {
	if runner == nil {
		return taskdefs.ExecutionResult{
			Status: taskdefs.RunStatusError,
			Error:  fmt.Sprintf("%s task executor is not configured", kind),
		}
	}
	return runner.ExecuteTask(ctx, cmd.Task, cmd.TraceID)
}
