package workflows

import (
	"fmt"
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
)

type ScreenControlStep struct {
	Action     string
	ToolAction string
	Params     map[string]any
}

func ParseScreenControlSteps(arguments map[string]any) ([]ScreenControlStep, map[string]any, error) {
	rawSteps, hasWorkflowSteps := arguments[ScreenControlWorkflowStepsKey]
	if !hasWorkflowSteps {
		return nil, nil, nil
	}
	if _, hasAction := arguments[ScreenControlActionKey]; hasAction {
		return nil, nil, fmt.Errorf("screen_control workflow_steps conflicts with action/params")
	}
	if _, hasParams := arguments[ScreenControlParamsKey]; hasParams {
		return nil, nil, fmt.Errorf("screen_control workflow_steps conflicts with action/params")
	}
	return parseScreenControlStepList(arguments, rawSteps)
}

func parseScreenControlStepList(arguments map[string]any, rawSteps any) ([]ScreenControlStep, map[string]any, error) {
	stepsRaw, ok := normalizeScreenControlStepList(rawSteps)
	if !ok {
		return nil, nil, fmt.Errorf("screen_control workflow_steps must be an array")
	}
	if len(stepsRaw) == 0 {
		return nil, nil, fmt.Errorf("screen_control workflow_steps must contain at least 1 step")
	}
	steps, err := decodeScreenControlSteps(stepsRaw)
	if err != nil {
		return nil, nil, err
	}
	baseArgs := bridgeTasks.CloneActionParams(arguments)
	delete(baseArgs, ScreenControlWorkflowStepsKey)
	return steps, baseArgs, nil
}

func normalizeScreenControlStepList(input any) ([]any, bool) {
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

func decodeScreenControlSteps(stepsRaw []any) ([]ScreenControlStep, error) {
	steps := make([]ScreenControlStep, 0, len(stepsRaw))
	for index := range stepsRaw {
		rawStep, ok := stepsRaw[index].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("screen_control workflow_steps[%d] must be an object", index+1)
		}
		step, err := DecodeScreenControlStep(rawStep, index)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, nil
}

func DecodeScreenControlStep(rawStep map[string]any, index int) (ScreenControlStep, error) {
	action := strings.ToLower(strings.TrimSpace(mapString(rawStep, ScreenControlActionKey)))
	if action == "" {
		return ScreenControlStep{}, fmt.Errorf("screen_control workflow_steps[%d] requires action", index+1)
	}
	return decodeScreenControlStepAction(rawStep, index, action)
}
