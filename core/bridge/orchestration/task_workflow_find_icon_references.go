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
