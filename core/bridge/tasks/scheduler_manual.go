package tasks

import "context"

func (s *TaskScheduler) beginManualRun(
	reg *taskRegistration,
	task ScheduledTask,
) (ScheduledTask, context.Context, *taskRegistration, bool, string) {
	if reg == nil {
		reg = &taskRegistration{task: task}
	}
	task, runCtx, skipped, reason := reg.beginRun(task, s.taskExecutionTimeoutForTask(task))
	if reason != skipRunReasonRegistrationRetired {
		return task, runCtx, reg, skipped, reason
	}
	reg = &taskRegistration{task: task, loopCtx: reg.loopCtx}
	task, runCtx, skipped, reason = reg.beginRun(task, s.taskExecutionTimeoutForTask(task))
	return task, runCtx, reg, skipped, reason
}
