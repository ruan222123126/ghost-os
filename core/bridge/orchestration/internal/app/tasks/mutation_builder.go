package tasks

import (
	"strings"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/taskdefs"
	bridgeTasks "ghost-os/bridge/tasks"
)

type TaskSessionGuard interface {
	EnsureTaskSessionActive(taskKind string, sessionID string) error
}

type TaskMutationBuilder struct {
	ConfigStore  bridgeconfig.Store
	SessionGuard TaskSessionGuard
	Now          func() time.Time
}

func (b TaskMutationBuilder) NewScheduledTask(params api.TaskCreateParams) (bridgeTasks.ScheduledTask, error) {
	task, err := b.buildScheduledTask(params)
	if err != nil {
		return bridgeTasks.ScheduledTask{}, err
	}
	nextRunAt, err := bridgeTasks.NextTaskRunAt(task, b.now())
	if err != nil {
		return bridgeTasks.ScheduledTask{}, WrapConfigError(err)
	}
	task.NextRunAt = nextRunAt
	return task, nil
}

func (b TaskMutationBuilder) buildScheduledTask(params api.TaskCreateParams) (bridgeTasks.ScheduledTask, error) {
	task, err := BuildTaskFromCreateParams(params, b.now())
	if err != nil {
		return bridgeTasks.ScheduledTask{}, err
	}
	if err := ValidateDefinition(&task); err != nil {
		return bridgeTasks.ScheduledTask{}, WrapConfigError(err)
	}
	if err := ValidateRuntime(&task, b.ConfigStore); err != nil {
		return bridgeTasks.ScheduledTask{}, err
	}
	if err := b.ensureTaskSessionActive(task.TaskKind, task.SessionID); err != nil {
		return bridgeTasks.ScheduledTask{}, err
	}
	return task, nil
}

func (b TaskMutationBuilder) ApplyUpdate(
	task *bridgeTasks.ScheduledTask,
	params api.TaskUpdateParams,
) error {
	if task == nil {
		return InvalidConfig("task is nil")
	}
	previousEnabled := task.Enabled
	scheduleChanged, err := ApplyTaskPatch(task, params)
	if err != nil {
		return err
	}
	if err := ValidateDefinition(task); err != nil {
		return WrapConfigError(err)
	}
	if err := ValidateRuntime(task, b.ConfigStore); err != nil {
		return err
	}
	if err := b.ensureTaskSessionActive(task.TaskKind, task.SessionID); err != nil {
		return err
	}
	return b.refreshNextRunAt(task, scheduleChanged, previousEnabled)
}

func (b TaskMutationBuilder) refreshNextRunAt(
	task *bridgeTasks.ScheduledTask,
	scheduleChanged bool,
	previousEnabled bool,
) error {
	if !task.Enabled {
		task.NextRunAt = time.Time{}
		return nil
	}
	if !scheduleChanged && previousEnabled && !task.NextRunAt.IsZero() {
		return nil
	}
	nextRunAt, err := bridgeTasks.NextTaskRunAt(*task, b.now())
	if err != nil {
		return WrapConfigError(err)
	}
	task.NextRunAt = nextRunAt
	return nil
}

func (b TaskMutationBuilder) ensureTaskSessionActive(taskKind string, sessionID string) error {
	if b.SessionGuard == nil {
		return nil
	}
	return b.SessionGuard.EnsureTaskSessionActive(taskKind, sessionID)
}

func (b TaskMutationBuilder) now() time.Time {
	if b.Now != nil {
		return b.Now().UTC()
	}
	return time.Now().UTC()
}

func BuildTaskFromCreateParams(
	params api.TaskCreateParams,
	now time.Time,
) (bridgeTasks.ScheduledTask, error) {
	cronExpr := strings.TrimSpace(params.CronExpr)
	if invalidScheduleFields(params.IntervalSeconds, cronExpr) {
		return bridgeTasks.ScheduledTask{}, InvalidConfig("exactly one of interval_seconds or cron_expr is required")
	}
	taskKind := bridgeTasks.NormalizeKind(params.TaskKind)
	if err := EnsureWorkflowAllowedForKind(taskKind, params.Workflow); err != nil {
		return bridgeTasks.ScheduledTask{}, err
	}
	if err := EnsureOrchestrationAllowedForKind(taskKind, params.Orchestration); err != nil {
		return bridgeTasks.ScheduledTask{}, err
	}

	task := bridgeTasks.ScheduledTask{
		Name:             strings.TrimSpace(params.Name),
		Message:          strings.TrimSpace(params.Message),
		SessionID:        strings.TrimSpace(params.SessionID),
		RuntimeOverrides: taskdefs.CloneTaskRuntimeOverrides(params.RuntimeOverrides),
		AgentMode:        strings.TrimSpace(params.AgentMode),
		Relay:            taskdefs.CloneTaskRelayConfig(params.Relay),
		TaskKind:         strings.TrimSpace(params.TaskKind),
		Action:           strings.TrimSpace(params.Action),
		ActionParams:     taskdefs.CloneActionParams(params.ActionParams),
		Workflow:         taskdefs.CloneWorkflowDefinition(params.Workflow),
		Orchestration:    taskdefs.CloneOrchestrationDefinition(params.Orchestration),
		Enabled:          true,
		CreatedAt:        now.UTC(),
		ScheduleType:     bridgeTasks.ScheduleTypeInterval,
		IntervalSeconds:  params.IntervalSeconds,
	}
	if cronExpr != "" {
		task.ScheduleType = bridgeTasks.ScheduleTypeCron
		task.IntervalSeconds = 0
		task.CronExpr = cronExpr
	}
	return task, nil
}

func invalidScheduleFields(intervalSeconds int, cronExpr string) bool {
	return (intervalSeconds > 0 && cronExpr != "") || (intervalSeconds <= 0 && cronExpr == "")
}

func ApplyTaskPatch(
	task *bridgeTasks.ScheduledTask,
	params api.TaskUpdateParams,
) (bool, error) {
	applyTaskCoreTextFields(task, params)
	applyTaskRuntimePatch(task, params.RuntimeOverrides)
	applyTaskRelayPatch(task, params)
	taskKind := applyTaskKindActionPatch(task, params)
	applyTaskActionParamsPatch(task, params.ActionParams)
	if err := applyTaskWorkflowPatch(task, taskKind, params.Workflow); err != nil {
		return false, err
	}
	if err := applyTaskOrchestrationPatch(task, taskKind, params.Name, params.Orchestration); err != nil {
		return false, err
	}
	applyTaskEnabledPatch(task, params.Enabled)
	return applyTaskSchedulePatch(task, params.IntervalSeconds, params.CronExpr)
}

func applyTaskRelayPatch(task *bridgeTasks.ScheduledTask, params api.TaskUpdateParams) {
	if params.AgentMode != nil {
		task.AgentMode = strings.TrimSpace(*params.AgentMode)
	}
	if params.Relay != nil {
		task.Relay = taskdefs.CloneTaskRelayConfig(params.Relay)
	}
}

func applyTaskCoreTextFields(task *bridgeTasks.ScheduledTask, params api.TaskUpdateParams) {
	if params.Name != nil {
		task.Name = strings.TrimSpace(*params.Name)
	}
	if params.Message != nil {
		task.Message = strings.TrimSpace(*params.Message)
	}
	if params.SessionID != nil {
		task.SessionID = strings.TrimSpace(*params.SessionID)
	}
}

func applyTaskRuntimePatch(task *bridgeTasks.ScheduledTask, runtimeOverrides *taskdefs.TaskRuntimeOverrides) {
	if runtimeOverrides != nil {
		task.RuntimeOverrides = taskdefs.CloneTaskRuntimeOverrides(runtimeOverrides)
	}
}

func applyTaskKindActionPatch(task *bridgeTasks.ScheduledTask, params api.TaskUpdateParams) string {
	if params.TaskKind != nil {
		task.TaskKind = strings.TrimSpace(*params.TaskKind)
	}
	if params.Action != nil {
		task.Action = strings.TrimSpace(*params.Action)
	}
	return bridgeTasks.NormalizeKind(task.TaskKind)
}

func applyTaskActionParamsPatch(task *bridgeTasks.ScheduledTask, actionParams *map[string]any) {
	if actionParams != nil {
		task.ActionParams = taskdefs.CloneActionParams(*actionParams)
	}
}

func applyTaskWorkflowPatch(
	task *bridgeTasks.ScheduledTask,
	taskKind string,
	workflow *taskdefs.WorkflowDefinition,
) error {
	if workflow != nil {
		if err := EnsureWorkflowAllowedForKind(taskKind, workflow); err != nil {
			return err
		}
		task.Workflow = taskdefs.CloneWorkflowDefinition(workflow)
	}
	if taskKind != bridgeTasks.KindWorkflow {
		task.Workflow = nil
	}
	return nil
}

func applyTaskOrchestrationPatch(
	task *bridgeTasks.ScheduledTask,
	taskKind string,
	name *string,
	definition *taskdefs.OrchestrationDefinition,
) error {
	if definition != nil {
		if err := EnsureOrchestrationAllowedForKind(taskKind, definition); err != nil {
			return err
		}
		task.Orchestration = taskdefs.CloneOrchestrationDefinition(definition)
	}
	if taskKind != bridgeTasks.KindOrchestration {
		task.Name = ""
		task.Orchestration = nil
		return nil
	}
	if name != nil {
		task.Name = strings.TrimSpace(*name)
	}
	return nil
}

func applyTaskEnabledPatch(task *bridgeTasks.ScheduledTask, enabled *bool) {
	if enabled != nil {
		task.Enabled = *enabled
	}
}

func applyTaskSchedulePatch(
	task *bridgeTasks.ScheduledTask,
	intervalSeconds *int,
	cronExpr *string,
) (bool, error) {
	if intervalSeconds != nil && cronExpr != nil {
		return false, InvalidConfig("exactly one schedule field can be updated at a time")
	}
	if intervalSeconds != nil {
		task.ScheduleType = bridgeTasks.ScheduleTypeInterval
		task.IntervalSeconds = *intervalSeconds
		task.CronExpr = ""
		return true, nil
	}
	if cronExpr != nil {
		task.ScheduleType = bridgeTasks.ScheduleTypeCron
		task.CronExpr = strings.TrimSpace(*cronExpr)
		task.IntervalSeconds = 0
		return true, nil
	}
	return false, nil
}
