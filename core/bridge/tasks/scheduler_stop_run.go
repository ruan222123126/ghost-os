package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (s *TaskScheduler) StopRun(
	ctx context.Context,
	taskID string,
	runID string,
) (RunLog, error) {
	if err := s.requireConfigured(); err != nil {
		return RunLog{}, err
	}
	if strings.TrimSpace(taskID) == "" {
		return RunLog{}, ErrInvalidTaskID
	}
	if strings.TrimSpace(runID) == "" {
		return RunLog{}, fmt.Errorf("%w: run_id is required", ErrInvalidTaskConfig)
	}
	if s.activeRuns == nil {
		return RunLog{}, ErrTaskRunNotRunning
	}
	return s.activeRuns.cancelAndWait(ctx, taskID, runID)
}

func (s *TaskScheduler) startActiveRun(
	taskID string,
	runID string,
	reg *taskRegistration,
) (*activeRunHandle, error) {
	if s == nil || s.activeRuns == nil {
		return nil, errors.New("active run registry is not configured")
	}
	if reg == nil {
		return nil, errors.New("task registration is required")
	}
	return s.activeRuns.register(taskID, runID, reg.currentRunCancel())
}

func (s *TaskScheduler) discardActiveRun(handle *activeRunHandle) {
	if s == nil || s.activeRuns == nil || handle == nil {
		return
	}
	s.activeRuns.discard(handle)
}

func (s *TaskScheduler) finishActiveRun(handle *activeRunHandle, run RunLog) {
	if s == nil || s.activeRuns == nil || handle == nil {
		return
	}
	s.activeRuns.finish(handle, run)
}
