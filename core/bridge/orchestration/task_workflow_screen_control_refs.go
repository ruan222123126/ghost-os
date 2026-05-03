package orchestration

import (
	"fmt"
	"strings"
)

const workflowScreenControlCoordinateRefKey = "coordinate_ref"
const workflowScreenControlFindIconCoordinateRef = "${find_icon}"

func resolveWorkflowScreenControlStepParams(
	step workflowScreenControlStep,
	lastFindIconOutput any,
) (map[string]any, error) {
	params := cloneTaskActionParams(step.Params)
	if step.Action != "click" {
		return params, nil
	}
	reference := strings.TrimSpace(workflowMapString(params, workflowScreenControlCoordinateRefKey))
	if reference == "" {
		return params, nil
	}
	if reference != workflowScreenControlFindIconCoordinateRef {
		return nil, fmt.Errorf("unsupported coordinate_ref %q", reference)
	}
	if lastFindIconOutput == nil {
		return nil, fmt.Errorf("coordinate_ref %q requires a previous find_icon result", reference)
	}
	match, err := workflowFindIconMatchCenter(lastFindIconOutput)
	if err != nil {
		return nil, err
	}
	delete(params, workflowScreenControlCoordinateRefKey)
	params["x"] = match.x
	params["y"] = match.y
	if _, exists := params["display_id"]; !exists && match.displayID >= 0 {
		params["display_id"] = match.displayID
	}
	return params, nil
}

type workflowFindIconMatch struct {
	x         float64
	y         float64
	displayID int
}

func workflowFindIconMatchCenter(output any) (workflowFindIconMatch, error) {
	record, ok := output.(map[string]any)
	if !ok {
		return workflowFindIconMatch{}, fmt.Errorf("find_icon output must be an object")
	}
	matches, ok := record["matches"].([]any)
	if !ok || len(matches) == 0 {
		return workflowFindIconMatch{}, fmt.Errorf("find_icon output field matches is empty")
	}
	first, ok := matches[0].(map[string]any)
	if !ok {
		return workflowFindIconMatch{}, fmt.Errorf("find_icon output field matches[0] must be an object")
	}
	center, ok := first["center"].(map[string]any)
	if !ok {
		return workflowFindIconMatch{}, fmt.Errorf("find_icon output field matches[0].center must be an object")
	}
	x, ok := workflowAnyNumber(center["x"])
	if !ok {
		return workflowFindIconMatch{}, fmt.Errorf("find_icon output field matches[0].center.x must be a number")
	}
	y, ok := workflowAnyNumber(center["y"])
	if !ok {
		return workflowFindIconMatch{}, fmt.Errorf("find_icon output field matches[0].center.y must be a number")
	}
	displayID := -1
	if value, ok := workflowAnyInteger(record["display_id"]); ok {
		displayID = value
	}
	return workflowFindIconMatch{x: x, y: y, displayID: displayID}, nil
}

func workflowAnyNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	default:
		return 0, false
	}
}

func workflowAnyInteger(value any) (int, bool) {
	number, ok := workflowAnyNumber(value)
	if !ok {
		return 0, false
	}
	return int(number), true
}
