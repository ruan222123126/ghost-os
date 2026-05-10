package orchestration

import workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"

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
