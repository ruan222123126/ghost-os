package workflows

import (
	"fmt"
	"strings"

	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	bridgeTasks "ghost-os/bridge/tasks"
)

func ResolveScreenControlStepParams(
	step ScreenControlStep,
	lastFindIconOutput any,
) (map[string]any, error) {
	params := bridgeTasks.CloneActionParams(step.Params)
	if step.Action != "click" {
		return params, nil
	}
	reference := strings.TrimSpace(mapString(params, ScreenControlCoordinateRefKey))
	if reference == "" {
		return params, nil
	}
	if reference != ScreenControlFindIconRef {
		return nil, fmt.Errorf("unsupported coordinate_ref %q", reference)
	}
	return resolveFindIconCoordinateRef(params, reference, lastFindIconOutput)
}

func resolveFindIconCoordinateRef(
	params map[string]any,
	reference string,
	lastFindIconOutput any,
) (map[string]any, error) {
	if lastFindIconOutput == nil {
		return nil, fmt.Errorf("coordinate_ref %q requires a previous find_icon result", reference)
	}
	match, err := workflowdomain.FindIconMatchCenter(lastFindIconOutput)
	if err != nil {
		return nil, err
	}
	delete(params, ScreenControlCoordinateRefKey)
	params["x"] = match.X
	params["y"] = match.Y
	if _, exists := params["display_id"]; !exists && match.DisplayID >= 0 {
		params["display_id"] = match.DisplayID
	}
	return params, nil
}

func PrepareScreenControlStepArguments(
	baseArgs map[string]any,
	step ScreenControlStep,
	lastFindIconOutput any,
	uploader TemplateUploader,
) (map[string]any, error) {
	args := bridgeTasks.CloneActionParams(baseArgs)
	if args == nil {
		args = map[string]any{}
	}
	args["mode"] = ScreenControlAtomicMode
	args[ScreenControlActionKey] = step.ToolAction
	params, err := ResolveScreenControlStepParams(step, lastFindIconOutput)
	if err != nil {
		return nil, err
	}
	args[ScreenControlParamsKey] = paramsOrEmpty(params)
	return PrepareToolArguments(ScreenControlToolID, args, uploader)
}

func paramsOrEmpty(params map[string]any) map[string]any {
	if params == nil {
		return map[string]any{}
	}
	return params
}
