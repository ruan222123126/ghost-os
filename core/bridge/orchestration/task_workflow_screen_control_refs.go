package orchestration

import appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"

const workflowScreenControlCoordinateRefKey = appworkflows.ScreenControlCoordinateRefKey
const workflowScreenControlFindIconCoordinateRef = appworkflows.ScreenControlFindIconRef

func resolveWorkflowScreenControlStepParams(
	step workflowScreenControlStep,
	lastFindIconOutput any,
) (map[string]any, error) {
	return appworkflows.ResolveScreenControlStepParams(toAppScreenControlStep(step), lastFindIconOutput)
}
