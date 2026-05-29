package tasks

import (
	"time"
)

type TaskStore = Store
type TaskRunLog = RunLog
type TaskLoadIssue = LoadIssue

const (
	taskScheduleTypeInterval     = ScheduleTypeInterval
	taskScheduleTypeCron         = ScheduleTypeCron
	defaultTaskRunLogRetention   = DefaultRunLogRetention
	taskRunStatusRunning         = RunStatusRunning
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

func NextTaskRunAt(task ScheduledTask, now time.Time) (time.Time, error) {
	return nextTaskRunAt(task, now)
}
