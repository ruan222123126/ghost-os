package tasks

import (
	"context"
	"log"
	"strings"
	"time"
)

type preparedManualRun struct {
	ctx         context.Context
	reg         *taskRegistration
	task        ScheduledTask
	scheduledAt time.Time
	traceID     string
}

type startedManualRun struct {
	runID      string
	startedAt  time.Time
	runSession RunSession
	runningLog RunLog
	activeRun  *activeRunHandle
}

func (s *TaskScheduler) StartNow(task ScheduledTask, traceID string) (RunLog, error) {
	prepared, skippedRun, err := s.prepareManualRun(task, traceID)
	if err != nil {
		return RunLog{}, err
	}
	if skippedRun != nil {
		return *skippedRun, nil
	}
	startedRun, err := s.startPreparedManualRun(prepared)
	if err != nil {
		return RunLog{}, err
	}
	go s.finishPreparedManualRun(prepared, startedRun)
	return startedRun.runningLog, nil
}

func (s *TaskScheduler) prepareManualRun(
	task ScheduledTask,
	traceID string,
) (preparedManualRun, *RunLog, error) {
	if err := s.requireConfigured(); err != nil {
		return preparedManualRun{}, nil, err
	}
	if err := NormalizeScheduledTask(&task, schedulerValidator(s.store)); err != nil {
		return preparedManualRun{}, nil, err
	}
	reg, lifecycleCtx, err := s.runningTask(task.ID)
	if err != nil {
		return preparedManualRun{}, nil, err
	}
	if reg == nil {
		reg = &taskRegistration{task: task, loopCtx: lifecycleCtx}
	}
	scheduledAt := s.now().UTC()
	runTraceID := strings.TrimSpace(traceID)
	if runTraceID == "" {
		runTraceID = s.traceID()
	}
	task, runCtx, reg, skipped, reason := s.beginManualRun(reg, task)
	if skipped {
		run := skippedTaskRunLog(task, runTraceID, scheduledAt, reason)
		return preparedManualRun{}, &run, s.store.AppendRunLog(run)
	}
	return preparedManualRun{
		ctx:         runCtx,
		reg:         reg,
		task:        task,
		scheduledAt: scheduledAt,
		traceID:     runTraceID,
	}, nil, nil
}

func (s *TaskScheduler) startPreparedManualRun(input preparedManualRun) (startedManualRun, error) {
	startedAt := s.now().UTC()
	runID := NewRunID()
	activeRun, err := s.startActiveRun(input.task.ID, runID, input.reg)
	if err != nil {
		input.reg.finishRun()
		return startedManualRun{}, err
	}
	runSession, err := s.appendRunningRunLog(
		input.ctx,
		input.task,
		input.scheduledAt,
		input.traceID,
		runID,
		startedAt,
	)
	if err != nil {
		s.discardActiveRun(activeRun)
		input.reg.finishRun()
		return startedManualRun{}, err
	}
	return startedManualRun{
		runID:      runID,
		startedAt:  startedAt,
		runSession: runSession,
		activeRun:  activeRun,
		runningLog: s.newRunLog(runLogInput{
			task:            input.task,
			runID:           runID,
			traceID:         input.traceID,
			scheduledAt:     input.scheduledAt,
			startedAt:       startedAt,
			status:          RunStatusRunning,
			sessionIDOutput: runSession.SessionID,
		}),
	}, nil
}

func (s *TaskScheduler) finishPreparedManualRun(input preparedManualRun, started startedManualRun) {
	run, err := s.executeStartedManualRun(input, started)
	if err != nil {
		log.Printf(
			"task scheduler start-now completion failed: task_id=%s run_id=%s status=%s error=%v",
			input.task.ID,
			started.runID,
			run.Status,
			err,
		)
	}
}

func (s *TaskScheduler) executeStartedManualRun(
	input preparedManualRun,
	started startedManualRun,
) (RunLog, error) {
	defer input.reg.finishRun()
	result := s.executeTaskRun(
		input.ctx,
		input.reg.executionTimeout(),
		input.task,
		input.traceID,
		started.runSession,
	)
	run, err := s.persistFinishedRun(input.reg, finishRunInput{
		task:        input.task,
		runID:       started.runID,
		traceID:     input.traceID,
		scheduledAt: input.scheduledAt,
		startedAt:   started.startedAt,
		finishedAt:  s.now().UTC(),
		runSession:  started.runSession,
		result:      result,
	})
	s.finishActiveRun(started.activeRun, run)
	return run, err
}
