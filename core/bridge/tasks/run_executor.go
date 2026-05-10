package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *TaskScheduler) executeRun(
	ctx context.Context,
	reg *taskRegistration,
	task ScheduledTask,
	scheduledAt time.Time,
	traceID string,
) (RunLog, error) {
	defer reg.finishRun()

	startedAt := s.now().UTC()
	result := s.finalizeExecutionResult(ctx, reg.executionTimeout(), s.execute(ctx, task, traceID))
	finishedAt := s.now().UTC()

	reg.mu.Lock()
	reg.task.LastRunAt = startedAt
	reg.task.LastError = strings.TrimSpace(result.Error)
	updated := reg.task
	reg.mu.Unlock()

	if err := s.store.SaveTask(&updated); err != nil {
		return RunLog{}, fmt.Errorf("save run state for %s: %w", task.ID, err)
	}
	run := RunLog{
		TaskID:          task.ID,
		RunID:           NewRunID(),
		TraceID:         traceID,
		TaskKind:        task.TaskKind,
		Action:          task.Action,
		ScheduledAt:     scheduledAt.UTC(),
		StartedAt:       startedAt,
		FinishedAt:      finishedAt,
		Status:          result.Status,
		SessionIDInput:  task.SessionID,
		SessionIDOutput: result.SessionIDOutput,
		ResponsePreview: result.ResponsePreview,
		NodeResults:     CloneRunNodeResults(result.NodeResults),
		Error:           result.Error,
	}
	if err := s.store.AppendRunLog(run); err != nil {
		return run, fmt.Errorf("append run log for %s: %w", task.ID, err)
	}
	return run, nil
}

func (s *TaskScheduler) finalizeExecutionResult(
	ctx context.Context,
	executionTimeout time.Duration,
	result ExecutionResult,
) ExecutionResult {
	if strings.TrimSpace(result.Status) == "" {
		result.Status = RunStatusError
	}

	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		result.Status = RunStatusCancelled
		result.Error = "task execution cancelled"
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		result.Status = RunStatusError
		result.Error = fmt.Sprintf(
			"task execution timed out after %s",
			normalizeExecutionTimeout(executionTimeout),
		)
	}

	return result
}

func (s *TaskScheduler) persistNextRun(reg *taskRegistration, next time.Time) error {
	reg.mu.Lock()
	reg.task.NextRunAt = next.UTC()
	updated := reg.task
	running := reg.running
	reg.mu.Unlock()
	if running {
		return nil
	}
	return s.store.SaveTask(&updated)
}

func (s *TaskScheduler) executeWithExecutor(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
) ExecutionResult {
	if s == nil || s.executor == nil {
		return ExecutionResult{Status: RunStatusError, Error: "task executor is not configured"}
	}
	return s.executor.Execute(ctx, task, traceID)
}

func (s *TaskScheduler) taskExecutionTimeout() time.Duration {
	if s == nil {
		return defaultTaskExecutionTimeout
	}
	s.mu.Lock()
	timeout := s.executionTimeout
	s.mu.Unlock()
	return normalizeExecutionTimeout(timeout)
}

func (s *TaskScheduler) taskExecutionTimeoutForTask(task ScheduledTask) time.Duration {
	if IsRelayAgentTask(task) && task.Relay != nil && task.Relay.ExecutionTimeoutMS != nil {
		return time.Duration(*task.Relay.ExecutionTimeoutMS) * time.Millisecond
	}
	return s.taskExecutionTimeout()
}

func (s *TaskScheduler) ExecutionTimeout() time.Duration {
	return s.taskExecutionTimeout()
}

func skippedTaskRunLog(
	task ScheduledTask,
	traceID string,
	scheduledAt time.Time,
	reason string,
) RunLog {
	skipReason := strings.TrimSpace(reason)
	if skipReason == "" {
		skipReason = skipRunReasonAlreadyRunning
	}
	return RunLog{
		TaskID:         task.ID,
		RunID:          NewRunID(),
		TraceID:        traceID,
		TaskKind:       task.TaskKind,
		Action:         task.Action,
		ScheduledAt:    scheduledAt.UTC(),
		Status:         RunStatusSkipped,
		SessionIDInput: task.SessionID,
		Error:          skipReason,
	}
}
