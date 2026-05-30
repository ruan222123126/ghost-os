package tasks

import bridgeTasks "ghost-os/bridge/tasks"

type Store interface {
	ListTasks() ([]bridgeTasks.ScheduledTask, error)
	LoadTask(taskID string) (*bridgeTasks.ScheduledTask, error)
	SaveTask(task *bridgeTasks.ScheduledTask) error
	DeleteTask(taskID string) error
	ListRunLogs(taskID string, limit int) ([]bridgeTasks.RunLog, error)
}

type Scheduler interface {
	Upsert(task bridgeTasks.ScheduledTask) error
	Unregister(taskID string) error
	RunNow(task bridgeTasks.ScheduledTask, traceID string) (bridgeTasks.RunLog, error)
	StartNow(task bridgeTasks.ScheduledTask, traceID string) (bridgeTasks.RunLog, error)
}
