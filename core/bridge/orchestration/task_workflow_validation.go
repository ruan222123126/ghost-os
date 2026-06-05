package orchestration

import (
	"fmt"

	appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"
	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
)

const (
	workflowNodeTypeStart = workflowdomain.NodeTypeStart
	workflowNodeTypeTool  = workflowdomain.NodeTypeTool
	workflowNodeTypeLLM   = workflowdomain.NodeTypeLLM
	workflowNodeTypeAgent = workflowdomain.NodeTypeAgent
	workflowNodeTypeIf    = workflowdomain.NodeTypeIf
	workflowNodeTypeLoop  = workflowdomain.NodeTypeLoop
	workflowNodeTypeEnd   = workflowdomain.NodeTypeEnd

	workflowInputTypeString  = workflowdomain.InputTypeString
	workflowInputTypeNumber  = workflowdomain.InputTypeNumber
	workflowInputTypeBoolean = workflowdomain.InputTypeBoolean
	workflowInputTypeObject  = workflowdomain.InputTypeObject
	workflowInputTypeArray   = workflowdomain.InputTypeArray

	workflowIfOperatorEquals      = workflowdomain.IfOperatorEquals
	workflowIfOperatorNotEquals   = workflowdomain.IfOperatorNotEquals
	workflowIfOperatorContains    = workflowdomain.IfOperatorContains
	workflowIfOperatorNotContains = workflowdomain.IfOperatorNotContains
	workflowIfOperatorIsEmpty     = workflowdomain.IfOperatorIsEmpty
	workflowIfOperatorNotEmpty    = workflowdomain.IfOperatorNotEmpty
)

type workflowExecutionPlan = workflowdomain.Plan

func validateWorkflowTaskDefinition(task *ScheduledTask) error {
	if task.Workflow == nil {
		return fmt.Errorf("%w: workflow is required for workflow task", ErrInvalidTaskConfig)
	}
	task.Name = ""
	task.Orchestration = nil
	if task.Message != "" {
		return fmt.Errorf("%w: workflow task does not allow message", ErrInvalidTaskConfig)
	}
	if task.SessionID != "" {
		return fmt.Errorf("%w: workflow task does not allow session_id", ErrInvalidTaskConfig)
	}
	if task.Action != "" {
		return fmt.Errorf("%w: workflow task does not allow action", ErrInvalidTaskConfig)
	}
	if len(task.ActionParams) > 0 {
		return fmt.Errorf("%w: workflow task does not allow action_params", ErrInvalidTaskConfig)
	}
	_, err := buildWorkflowExecutionPlan(task.Workflow)
	return err
}

func buildWorkflowExecutionPlan(definition *WorkflowDefinition) (workflowExecutionPlan, error) {
	return workflowdomain.PlanBuilder{}.Build(definition)
}

func evaluateWorkflowIfCondition(operator string, source string, value string) (bool, error) {
	return workflowdomain.EvaluateIfCondition(operator, source, value)
}

func isWorkflowIfOperatorSupported(operator string) bool {
	return workflowdomain.IsIfOperatorSupported(operator)
}

func requiresWorkflowIfValue(operator string) bool {
	return workflowdomain.RequiresIfValue(operator)
}

func resolveWorkflowActionNode(node WorkflowNode, findIconOutput any) (WorkflowNode, error) {
	return workflowdomain.ResolveActionNode(node, findIconOutput)
}

func resolveWorkflowIfValue(value string, findIconOutput any) (string, error) {
	return workflowdomain.ResolveIfValue(value, findIconOutput)
}

func resolveWorkflowFindIconString(input string, findIconOutput any) (string, error) {
	return workflowdomain.ResolveFindIconString(input, findIconOutput)
}

func resolveWorkflowFindIconTemplateValue(value any, findIcon any) (any, error) {
	return workflowdomain.ResolveFindIconTemplateValue(value, findIcon)
}

func resolveWorkflowFindIconTemplateString(input string, findIcon any) (any, error) {
	return workflowdomain.ResolveFindIconTemplateString(input, findIcon)
}

func resolveWorkflowFindIconPath(path string, findIcon any) (any, bool, error) {
	return workflowdomain.ResolveFindIconPath(path, findIcon)
}

func stringifyWorkflowFindIconTemplateValue(value any) string {
	return workflowdomain.StringifyFindIconTemplateValue(value)
}

func extractWorkflowFindIconOutput(node WorkflowNode, outputValue any) (any, bool) {
	return workflowdomain.ExtractFindIconOutput(node, screenControlToolID, outputValue)
}

func workflowFindIconMatchCenter(output any) (workflowFindIconMatch, error) {
	match, err := workflowdomain.FindIconMatchCenter(output)
	return workflowFindIconMatch{match.X, match.Y, match.DisplayID}, err
}

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

type workflowFindIconMatch struct {
	x         float64
	y         float64
	displayID int
}
