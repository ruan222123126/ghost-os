package orchestration

import (
	"strings"
	"time"
)

func (r taskMutationRunner) newScheduledTask(params taskCreateParams) (ScheduledTask, error) {
	task, err := r.buildScheduledTask(params)
	if err != nil {
		return ScheduledTask{}, err
	}
	nextRunAt, err := nextTaskRunAt(task, r.now())
	if err != nil {
		return ScheduledTask{}, wrapTaskConfigError(err)
	}
	task.NextRunAt = nextRunAt
	return task, nil
}

func (r taskMutationRunner) buildScheduledTask(params taskCreateParams) (ScheduledTask, error) {
	task, err := buildTaskFromCreateParams(params, r.now())
	if err != nil {
		return ScheduledTask{}, err
	}
	if err := validateTaskDefinition(&task); err != nil {
		return ScheduledTask{}, wrapTaskConfigError(err)
	}
	if err := r.validateTaskRuntime(&task); err != nil {
		return ScheduledTask{}, err
	}
	if err := r.ensureTaskSessionExists(task.TaskKind, task.SessionID); err != nil {
		return ScheduledTask{}, err
	}
	return task, nil
}

func buildTaskFromCreateParams(params taskCreateParams, now time.Time) (ScheduledTask, error) {
	cronExpr := strings.TrimSpace(params.CronExpr)
	if invalidScheduleFields(params.IntervalSeconds, cronExpr) {
		return ScheduledTask{}, invalidTaskConfig("exactly one of interval_seconds or cron_expr is required")
	}
	taskKind := normalizeTaskKind(params.TaskKind)
	if err := ensureWorkflowAllowedForTaskKind(taskKind, params.Workflow); err != nil {
		return ScheduledTask{}, err
	}
	if err := ensureOrchestrationAllowedForTaskKind(taskKind, params.Orchestration); err != nil {
		return ScheduledTask{}, err
	}

	task := ScheduledTask{
		Name:             strings.TrimSpace(params.Name),
		Message:          strings.TrimSpace(params.Message),
		SessionID:        strings.TrimSpace(params.SessionID),
		RuntimeOverrides: cloneTaskRuntimeOverrides(params.RuntimeOverrides),
		TaskKind:         strings.TrimSpace(params.TaskKind),
		Action:           strings.TrimSpace(params.Action),
		ActionParams:     cloneTaskActionParams(params.ActionParams),
		Workflow:         cloneTaskWorkflow(params.Workflow),
		Orchestration:    cloneTaskOrchestration(params.Orchestration),
		Enabled:          true,
		CreatedAt:        now,
		ScheduleType:     taskScheduleTypeInterval,
		IntervalSeconds:  params.IntervalSeconds,
	}
	if cronExpr != "" {
		task.ScheduleType = taskScheduleTypeCron
		task.IntervalSeconds = 0
		task.CronExpr = cronExpr
	}
	return task, nil
}

func invalidScheduleFields(intervalSeconds int, cronExpr string) bool {
	return (intervalSeconds > 0 && cronExpr != "") || (intervalSeconds <= 0 && cronExpr == "")
}
