package tasks

import "ghost-os/bridge/taskdefs"

type WorkflowDefinition = taskdefs.WorkflowDefinition
type WorkflowNode = taskdefs.WorkflowNode
type WorkflowStartNode = taskdefs.WorkflowStartNode
type WorkflowInputVariable = taskdefs.WorkflowInputVariable
type WorkflowToolNode = taskdefs.WorkflowToolNode
type WorkflowLLMNode = taskdefs.WorkflowLLMNode
type WorkflowAgentNode = taskdefs.WorkflowAgentNode
type WorkflowIfNode = taskdefs.WorkflowIfNode
type WorkflowLoopNode = taskdefs.WorkflowLoopNode
type WorkflowEdge = taskdefs.WorkflowEdge

func CloneWorkflowDefinition(input *WorkflowDefinition) *WorkflowDefinition {
	return taskdefs.CloneWorkflowDefinition(input)
}
