package orchestration

import workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"

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

type workflowFindIconMatch struct {
	x         float64
	y         float64
	displayID int
}
