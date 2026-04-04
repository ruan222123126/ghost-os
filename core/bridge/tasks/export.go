package tasks

import (
	"context"
	"time"
)

type TaskStore = Store
type TaskRunLog = RunLog
type TaskLoadIssue = LoadIssue

const (
	taskScheduleTypeInterval     = ScheduleTypeInterval
	taskScheduleTypeCron         = ScheduleTypeCron
	defaultTaskRunLogRetention   = DefaultRunLogRetention
	taskRunStatusSuccess         = RunStatusSuccess
	taskRunStatusCancelled       = RunStatusCancelled
	taskRunStatusError           = RunStatusError
	taskRunStatusSkipped         = RunStatusSkipped
	taskRunStatusAwaitingHuman   = RunStatusAwaitingHuman
	taskLoadIssueInvalidFilename = LoadIssueInvalidFilename
	taskLoadIssueReadError       = LoadIssueReadError
	taskLoadIssueDecodeError     = LoadIssueDecodeError
	taskLoadIssueInvalidConfig   = LoadIssueInvalidConfig
	taskLoadIssueIDMismatch      = LoadIssueIDMismatch
)

func NewTaskStore(baseDir string) (*TaskStore, error) {
	return NewStore(baseDir, nil)
}

func (s *TaskScheduler) SetExecuteHook(hook func(context.Context, ScheduledTask, string) ExecutionResult) {
	if s != nil && hook != nil {
		s.execute = hook
	}
}

func (s *TaskScheduler) SetExecutionTimeout(timeout time.Duration) {
	if s != nil {
		s.executionTimeout = timeout
	}
}

func (s *TaskScheduler) HasTask(taskID string) bool {
	return s.lookupTask(taskID) != nil
}

func (s *TaskScheduler) Running() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func NextTaskRunAt(task ScheduledTask, now time.Time) (time.Time, error) {
	return nextTaskRunAt(task, now)
}
