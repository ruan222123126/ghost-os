package screen

import (
	"fmt"
	"strings"

	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	"ghost-os/bridge/taskdefs"
)

func ResolveStepParams(step Step, lastFindIconOutput any) (map[string]any, error) {
	params := taskdefs.CloneActionParams(step.Params)
	if step.Action != "click" {
		return params, nil
	}
	reference := strings.TrimSpace(mapString(params, CoordinateRefKey))
	if reference == "" {
		return params, nil
	}
	if reference != FindIconRef {
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
	delete(params, CoordinateRefKey)
	params["x"] = match.X
	params["y"] = match.Y
	if _, exists := params["display_id"]; !exists && match.DisplayID >= 0 {
		params["display_id"] = match.DisplayID
	}
	return params, nil
}
