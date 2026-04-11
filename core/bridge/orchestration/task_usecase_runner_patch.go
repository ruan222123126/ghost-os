package orchestration

import (
	"strings"
	"time"
)

func (r taskMutationRunner) applyUpdate(task *ScheduledTask, params taskUpdateParams) error {
	if task == nil {
		return invalidTaskConfig("task is nil")
	}
	previousEnabled := task.Enabled
	scheduleChanged, err := applyTaskPatch(task, params)
	if err != nil {
		return err
	}
	if err := validateTaskDefinition(task); err != nil {
		return wrapTaskConfigError(err)
	}
	if err := r.validateTaskRuntime(task); err != nil {
		return err
	}
	if err := r.ensureTaskSessionExists(task.TaskKind, task.SessionID); err != nil {
		return err
	}
	return r.refreshNextRunAt(task, scheduleChanged, previousEnabled)
}

func applyTaskPatch(task *ScheduledTask, params taskUpdateParams) (bool, error) {
	applyTaskCoreTextFields(task, params)
	applyTaskRuntimePatch(task, params.RuntimeOverrides)
	taskKind := applyTaskKindActionPatch(task, params)
	applyTaskActionParamsPatch(task, params.ActionParams)
	if err := applyTaskWorkflowPatch(task, taskKind, params.Workflow); err != nil {
		return false, err
	}
	applyTaskEnabledPatch(task, params.Enabled)
	return applyTaskSchedulePatch(task, params.IntervalSeconds, params.CronExpr)
}

func applyTaskCoreTextFields(task *ScheduledTask, params taskUpdateParams) {
	if params.Message != nil {
		task.Message = strings.TrimSpace(*params.Message)
	}
	if params.SessionID != nil {
		task.SessionID = strings.TrimSpace(*params.SessionID)
	}
}

func applyTaskRuntimePatch(task *ScheduledTask, runtimeOverrides *TaskRuntimeOverrides) {
	if runtimeOverrides != nil {
		task.RuntimeOverrides = cloneTaskRuntimeOverrides(runtimeOverrides)
	}
}

func applyTaskKindActionPatch(task *ScheduledTask, params taskUpdateParams) string {
	if params.TaskKind != nil {
		task.TaskKind = strings.TrimSpace(*params.TaskKind)
	}
	if params.Action != nil {
		task.Action = strings.TrimSpace(*params.Action)
	}
	return normalizeTaskKind(task.TaskKind)
}

func applyTaskActionParamsPatch(task *ScheduledTask, actionParams *map[string]any) {
	if actionParams != nil {
		task.ActionParams = cloneTaskActionParams(*actionParams)
	}
}

func applyTaskWorkflowPatch(task *ScheduledTask, taskKind string, workflow *WorkflowDefinition) error {
	if workflow != nil {
		if err := ensureWorkflowAllowedForTaskKind(taskKind, workflow); err != nil {
			return err
		}
		task.Workflow = cloneTaskWorkflow(workflow)
	}
	if taskKind != taskKindWorkflow {
		task.Workflow = nil
	}
	return nil
}

func applyTaskEnabledPatch(task *ScheduledTask, enabled *bool) {
	if enabled != nil {
		task.Enabled = *enabled
	}
}

func applyTaskSchedulePatch(task *ScheduledTask, intervalSeconds *int, cronExpr *string) (bool, error) {
	if intervalSeconds != nil && cronExpr != nil {
		return false, invalidTaskConfig("exactly one schedule field can be updated at a time")
	}
	if intervalSeconds != nil {
		task.ScheduleType = taskScheduleTypeInterval
		task.IntervalSeconds = *intervalSeconds
		task.CronExpr = ""
		return true, nil
	}
	if cronExpr != nil {
		task.ScheduleType = taskScheduleTypeCron
		task.CronExpr = strings.TrimSpace(*cronExpr)
		task.IntervalSeconds = 0
		return true, nil
	}
	return false, nil
}

func (r taskMutationRunner) refreshNextRunAt(task *ScheduledTask, scheduleChanged bool, previousEnabled bool) error {
	if !task.Enabled {
		task.NextRunAt = time.Time{}
		return nil
	}
	if !scheduleChanged && previousEnabled && !task.NextRunAt.IsZero() {
		return nil
	}
	nextRunAt, err := nextTaskRunAt(*task, r.now())
	if err != nil {
		return wrapTaskConfigError(err)
	}
	task.NextRunAt = nextRunAt
	return nil
}
