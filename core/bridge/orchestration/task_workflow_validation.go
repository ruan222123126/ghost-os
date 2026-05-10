package orchestration

import (
	"fmt"

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
