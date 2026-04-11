package rss

import (
	"context"
	"fmt"
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
)

// ExecuteSystemTask executes an RSS system task and returns the execution result.
func (h *ActionHandler) ExecuteSystemTask(ctx context.Context, task bridgeTasks.ScheduledTask, traceID string) bridgeTasks.ExecutionResult {
	switch strings.TrimSpace(task.Action) {
	case ActionInboxPoll:
		return h.executeInboxPollTask(ctx, task, traceID)
	case ActionBriefingBuild:
		return h.executeBriefingBuildTask(ctx, task, traceID)
	default:
		return bridgeTasks.ExecutionResult{
			Status: bridgeTasks.RunStatusError,
			Error:  "unsupported system action: " + strings.TrimSpace(task.Action),
		}
	}
}

func (h *ActionHandler) executeInboxPollTask(
	ctx context.Context,
	task bridgeTasks.ScheduledTask,
	traceID string,
) bridgeTasks.ExecutionResult {
	params, err := DecodeInboxPollParams(task.ActionParams)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: err.Error()}
	}
	payload, _, err := h.ExecuteInboxPollUsecase(ctx, params, task.ID, traceID)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: err.Error()}
	}
	return bridgeTasks.ExecutionResult{
		Status:          bridgeTasks.RunStatusSuccess,
		ResponsePreview: FormatInboxPollPreview(payload),
	}
}

func (h *ActionHandler) executeBriefingBuildTask(
	ctx context.Context,
	task bridgeTasks.ScheduledTask,
	traceID string,
) bridgeTasks.ExecutionResult {
	params, err := DecodeBriefingParams(task.ActionParams)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: err.Error()}
	}
	params.TaskID = task.ID
	payload, _, err := h.ExecuteBriefingBuildAction(ctx, params, traceID)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: err.Error()}
	}
	typed, _ := payload.(RSSBriefingResult)
	return bridgeTasks.ExecutionResult{
		Status:          bridgeTasks.RunStatusSuccess,
		ResponsePreview: FormatBriefingPreview(typed),
	}
}

// ValidateSystemTaskParams validates and normalizes params for a system task definition.
func ValidateSystemTaskParams(task *bridgeTasks.ScheduledTask) error {
	switch task.Action {
	case ActionInboxPoll:
		return validateInboxPollTaskParams(task)
	case ActionBriefingBuild:
		return validateBriefingBuildTaskParams(task)
	default:
		return fmt.Errorf("%w: unsupported system action %q", bridgeTasks.ErrInvalidTaskConfig, task.Action)
	}
}

func validateInboxPollTaskParams(task *bridgeTasks.ScheduledTask) error {
	params, err := DecodeInboxPollParams(task.ActionParams)
	if err != nil {
		return fmt.Errorf("%w: invalid %s params: %v", bridgeTasks.ErrInvalidTaskConfig, ActionInboxPoll, err)
	}
	task.ActionParams = InboxPollParamsToMap(params)
	return nil
}

func validateBriefingBuildTaskParams(task *bridgeTasks.ScheduledTask) error {
	params, err := DecodeBriefingParams(task.ActionParams)
	if err != nil {
		return fmt.Errorf("%w: invalid %s params: %v", bridgeTasks.ErrInvalidTaskConfig, ActionBriefingBuild, err)
	}
	task.ActionParams = BriefingParamsToMap(params)
	return nil
}
