package orchestration

import (
	"fmt"
	"strings"
)

func validateTaskDefinition(task *ScheduledTask) error {
	if task == nil {
		return fmt.Errorf("%w: task is nil", ErrInvalidTaskConfig)
	}
	task.TaskKind = normalizeTaskKind(task.TaskKind)
	task.Action = strings.TrimSpace(task.Action)
	task.ActionParams = cloneTaskActionParams(task.ActionParams)

	switch task.TaskKind {
	case taskKindAgentMessage:
		if strings.TrimSpace(task.Message) == "" {
			return fmt.Errorf("%w: message is required", ErrInvalidTaskConfig)
		}
		task.Action = ""
		task.ActionParams = nil
	case taskKindSystemAction:
		task.Message = ""
		if strings.TrimSpace(task.SessionID) != "" {
			return fmt.Errorf("%w: system_action does not allow session_id", ErrInvalidTaskConfig)
		}
		if task.Action == "" {
			return fmt.Errorf("%w: action is required for system_action", ErrInvalidTaskConfig)
		}
		switch task.Action {
		case busActionRSSInboxPoll:
			params, err := decodeRSSInboxPollParams(task.ActionParams)
			if err != nil {
				return fmt.Errorf("%w: invalid %s params: %v", ErrInvalidTaskConfig, busActionRSSInboxPoll, err)
			}
			task.ActionParams = rssInboxPollParamsToMap(params)
		case busActionRSSBriefingBuild:
			params, err := decodeRSSBriefingParams(task.ActionParams)
			if err != nil {
				return fmt.Errorf("%w: invalid %s params: %v", ErrInvalidTaskConfig, busActionRSSBriefingBuild, err)
			}
			task.ActionParams = rssBriefingParamsToMap(params)
		default:
			return fmt.Errorf("%w: unsupported system action %q", ErrInvalidTaskConfig, task.Action)
		}
	default:
		return fmt.Errorf("%w: unsupported task_kind %q", ErrInvalidTaskConfig, task.TaskKind)
	}
	return nil
}
