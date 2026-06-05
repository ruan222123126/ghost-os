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
	runID := NewRunID()
	activeRun, err := s.startActiveRun(task.ID, runID, reg)
	if err != nil {
		return RunLog{}, err
	}
	runSession, err := s.appendRunningRunLog(ctx, task, scheduledAt, traceID, runID, startedAt)
	if err != nil {
		s.discardActiveRun(activeRun)
		return RunLog{}, err
	}
	result := s.executeTaskRun(ctx, reg.executionTimeout(), task, traceID, runSession)
	finishedAt := s.now().UTC()
	run, err := s.persistFinishedRun(reg, finishRunInput{
		task:        task,
		runID:       runID,
		traceID:     traceID,
		scheduledAt: scheduledAt,
		startedAt:   startedAt,
		finishedAt:  finishedAt,
		runSession:  runSession,
		result:      result,
	})
	s.finishActiveRun(activeRun, run)
	return run, err
}

func (s *TaskScheduler) appendRunningRunLog(
	ctx context.Context,
	task ScheduledTask,
	scheduledAt time.Time,
	traceID string,
	runID string,
	startedAt time.Time,
) (RunSession, error) {
	runSession, err := s.prepareRunSession(ctx, task, traceID)
	if err != nil {
		return RunSession{}, err
	}
	running := s.newRunLog(runLogInput{
		task:            task,
		runID:           runID,
		traceID:         traceID,
		scheduledAt:     scheduledAt,
		startedAt:       startedAt,
		status:          RunStatusRunning,
		sessionIDOutput: runSession.SessionID,
	})
	runSession.RunID = runID
	runSession.ProgressWriter = newRunningRunLogWriter(s.store, running)
	if err := s.store.AppendRunLog(running); err != nil {
		return RunSession{}, fmt.Errorf("append running run log for %s: %w", task.ID, err)
	}
	return runSession, nil
}

func (s *TaskScheduler) executeTaskRun(
	ctx context.Context,
	executionTimeout time.Duration,
	task ScheduledTask,
	traceID string,
	runSession RunSession,
) ExecutionResult {
	executionCtx := WithRunSession(ctx, runSession)
	executionTask := taskWithRunSession(task, runSession)
	result := s.finalizeExecutionResult(
		ctx,
		executionTimeout,
		s.execute(executionCtx, executionTask, traceID),
	)
	if strings.TrimSpace(result.SessionIDOutput) == "" {
		result.SessionIDOutput = runSession.SessionID
	}
	return result
}

type finishRunInput struct {
	task        ScheduledTask
	runID       string
	traceID     string
	scheduledAt time.Time
	startedAt   time.Time
	finishedAt  time.Time
	runSession  RunSession
	result      ExecutionResult
}

func (s *TaskScheduler) persistFinishedRun(
	reg *taskRegistration,
	input finishRunInput,
) (RunLog, error) {
	reg.mu.Lock()
	reg.task.LastRunAt = input.startedAt
	reg.task.LastError = strings.TrimSpace(input.result.Error)
	updated := reg.task
	reg.mu.Unlock()

	if err := s.store.SaveTask(&updated); err != nil {
		return RunLog{}, fmt.Errorf("save run state for %s: %w", input.task.ID, err)
	}
	run := s.newRunLog(runLogInput{
		task:            input.task,
		runID:           input.runID,
		traceID:         input.traceID,
		scheduledAt:     input.scheduledAt,
		startedAt:       input.startedAt,
		finishedAt:      input.finishedAt,
		status:          input.result.Status,
		sessionIDOutput: input.result.SessionIDOutput,
		responsePreview: input.result.ResponsePreview,
		nodeResults:     input.result.NodeResults,
		runCards:        input.result.RunCards,
		errorText:       input.result.Error,
	})
	if err := s.store.AppendRunLog(run); err != nil {
		return run, fmt.Errorf("append run log for %s: %w", input.task.ID, err)
	}
	return run, nil
}

func taskWithRunSession(task ScheduledTask, runSession RunSession) ScheduledTask {
	sessionID := strings.TrimSpace(runSession.SessionID)
	if sessionID == "" || NormalizeKind(task.TaskKind) != KindAgentMessage {
		return task
	}
	task.SessionID = sessionID
	return task
}

type runLogInput struct {
	task            ScheduledTask
	runID           string
	traceID         string
	scheduledAt     time.Time
	startedAt       time.Time
	finishedAt      time.Time
	status          string
	sessionIDOutput string
	responsePreview string
	nodeResults     []RunNodeResult
	runCards        []RunCard
	errorText       string
}

func (s *TaskScheduler) newRunLog(input runLogInput) RunLog {
	return RunLog{
		TaskID:          input.task.ID,
		RunID:           input.runID,
		TraceID:         input.traceID,
		TaskKind:        input.task.TaskKind,
		Action:          input.task.Action,
		ScheduledAt:     input.scheduledAt.UTC(),
		StartedAt:       input.startedAt,
		FinishedAt:      input.finishedAt,
		Status:          input.status,
		SessionIDInput:  input.task.SessionID,
		SessionIDOutput: input.sessionIDOutput,
		ResponsePreview: input.responsePreview,
		NodeResults:     CloneRunNodeResults(input.nodeResults),
		RunCards:        CloneRunCards(input.runCards),
		Error:           input.errorText,
	}
}

func (s *TaskScheduler) prepareRunSession(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
) (RunSession, error) {
	binder, ok := s.executor.(RunSessionBinder)
	if !ok || binder == nil {
		return RunSession{SessionID: task.SessionID}, nil
	}
	session, err := binder.PrepareRunSession(ctx, task, traceID)
	if err != nil {
		return RunSession{}, fmt.Errorf("prepare run session for %s: %w", task.ID, err)
	}
	if strings.TrimSpace(session.SessionID) == "" {
		session.SessionID = task.SessionID
	}
	return session, nil
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
		result.Error = ""
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
