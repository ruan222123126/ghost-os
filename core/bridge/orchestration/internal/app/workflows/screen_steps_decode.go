package workflows

import (
	"fmt"
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
)

func decodeScreenControlStepAction(
	rawStep map[string]any,
	index int,
	action string,
) (ScreenControlStep, error) {
	toolAction, err := MapScreenControlStepAction(action)
	if err != nil {
		return ScreenControlStep{}, fmt.Errorf("screen_control workflow_steps[%d]: %w", index+1, err)
	}
	params, err := mapObject(rawStep, ScreenControlParamsKey)
	if err != nil {
		return ScreenControlStep{}, fmt.Errorf("screen_control workflow_steps[%d]: %w", index+1, err)
	}
	return ScreenControlStep{Action: action, ToolAction: toolAction, Params: params}, nil
}

func MapScreenControlStepAction(action string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "screenshot":
		return "screenshot", nil
	case "find_text":
		return "find_text", nil
	case "find_icon":
		return "find_icon", nil
	case "click":
		return "click_icon", nil
	default:
		return "", fmt.Errorf("unsupported action %q", action)
	}
}

func mapObject(record map[string]any, key string) (map[string]any, error) {
	rawValue, exists := record[key]
	if !exists || rawValue == nil {
		return map[string]any{}, nil
	}
	value, ok := rawValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", key)
	}
	return bridgeTasks.CloneActionParams(value), nil
}
