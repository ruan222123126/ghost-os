package orchestration

import appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"

func fromAppWorkflowOutcome(outcome appworkflows.NodeOutcome) workflowNodeOutcome {
	return workflowNodeOutcome{
		status:        outcome.Status,
		sessionID:     outcome.SessionID,
		preview:       outcome.Preview,
		outputText:    outcome.OutputText,
		outputValue:   outcome.OutputValue,
		inputSnapshot: outcome.InputSnapshot,
		err:           outcome.Err,
	}
}

func toAppScreenControlStep(step workflowScreenControlStep) appworkflows.ScreenControlStep {
	return appworkflows.ScreenControlStep{
		Action:     step.Action,
		ToolAction: step.ToolAction,
		Params:     cloneTaskActionParams(step.Params),
	}
}

func fromAppScreenControlStep(step appworkflows.ScreenControlStep) workflowScreenControlStep {
	return workflowScreenControlStep{
		Action:     step.Action,
		ToolAction: step.ToolAction,
		Params:     cloneTaskActionParams(step.Params),
	}
}

func fromAppScreenControlSteps(steps []appworkflows.ScreenControlStep) []workflowScreenControlStep {
	if steps == nil {
		return nil
	}
	out := make([]workflowScreenControlStep, 0, len(steps))
	for _, step := range steps {
		out = append(out, fromAppScreenControlStep(step))
	}
	return out
}

func resolveWorkflowScreenControlStepParams(
	step workflowScreenControlStep,
	lastFindIconOutput any,
) (map[string]any, error) {
	return appworkflows.ResolveScreenControlStepParams(toAppScreenControlStep(step), lastFindIconOutput)
}
