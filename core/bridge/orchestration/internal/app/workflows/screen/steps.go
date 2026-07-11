package screen

import (
	"fmt"
	"strings"

	"ghost-os/bridge/taskdefs"
)

func ParseSteps(arguments map[string]any) ([]Step, map[string]any, error) {
	rawSteps, hasWorkflowSteps := arguments[WorkflowStepsKey]
	if !hasWorkflowSteps {
		return nil, nil, nil
	}
	if _, hasAction := arguments[ActionKey]; hasAction {
		return nil, nil, fmt.Errorf("screen_control workflow_steps conflicts with action/params")
	}
	if _, hasParams := arguments[ParamsKey]; hasParams {
		return nil, nil, fmt.Errorf("screen_control workflow_steps conflicts with action/params")
	}
	return parseStepList(arguments, rawSteps)
}

func parseStepList(arguments map[string]any, rawSteps any) ([]Step, map[string]any, error) {
	stepsRaw, ok := normalizeStepList(rawSteps)
	if !ok {
		return nil, nil, fmt.Errorf("screen_control workflow_steps must be an array")
	}
	if len(stepsRaw) == 0 {
		return nil, nil, fmt.Errorf("screen_control workflow_steps must contain at least 1 step")
	}
	steps, err := decodeSteps(stepsRaw)
	if err != nil {
		return nil, nil, err
	}
	baseArgs := taskdefs.CloneActionParams(arguments)
	delete(baseArgs, WorkflowStepsKey)
	return steps, baseArgs, nil
}

func normalizeStepList(input any) ([]any, bool) {
	if steps, ok := input.([]any); ok {
		return steps, true
	}
	if typed, ok := input.([]map[string]any); ok {
		steps := make([]any, 0, len(typed))
		for index := range typed {
			steps = append(steps, typed[index])
		}
		return steps, true
	}
	return nil, false
}

func decodeSteps(stepsRaw []any) ([]Step, error) {
	steps := make([]Step, 0, len(stepsRaw))
	for index := range stepsRaw {
		rawStep, ok := stepsRaw[index].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("screen_control workflow_steps[%d] must be an object", index+1)
		}
		step, err := DecodeStep(rawStep, index)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, nil
}

func DecodeStep(rawStep map[string]any, index int) (Step, error) {
	action := strings.ToLower(strings.TrimSpace(mapString(rawStep, ActionKey)))
	if action == "" {
		return Step{}, fmt.Errorf("screen_control workflow_steps[%d] requires action", index+1)
	}
	return decodeStepAction(rawStep, index, action)
}

func decodeStepAction(rawStep map[string]any, index int, action string) (Step, error) {
	toolAction, err := MapStepAction(action)
	if err != nil {
		return Step{}, fmt.Errorf("screen_control workflow_steps[%d]: %w", index+1, err)
	}
	params, err := mapObject(rawStep, ParamsKey)
	if err != nil {
		return Step{}, fmt.Errorf("screen_control workflow_steps[%d]: %w", index+1, err)
	}
	return Step{Action: action, ToolAction: toolAction, Params: params}, nil
}

func MapStepAction(action string) (string, error) {
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
	return taskdefs.CloneActionParams(value), nil
}
