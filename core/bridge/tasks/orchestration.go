package tasks

import "ghost-os/bridge/taskdefs"

const (
	OrchestrationNodeTypeGroup = taskdefs.OrchestrationNodeTypeGroup
	OrchestrationNodeTypeAgent = taskdefs.OrchestrationNodeTypeAgent

	OrchestrationEdgeKindControl = taskdefs.OrchestrationEdgeKindControl
	OrchestrationEdgeKindMember  = taskdefs.OrchestrationEdgeKindMember

	OrchestrationSpeakingModeSequential = taskdefs.OrchestrationSpeakingModeSequential
	OrchestrationSpeakingModeParallel   = taskdefs.OrchestrationSpeakingModeParallel
	OrchestrationSpeakingModeOwner      = taskdefs.OrchestrationSpeakingModeOwner
)

type OrchestrationDefinition = taskdefs.OrchestrationDefinition
type OrchestrationNode = taskdefs.OrchestrationNode
type OrchestrationGroupNode = taskdefs.OrchestrationGroupNode
type OrchestrationAgentNode = taskdefs.OrchestrationAgentNode
type OrchestrationEdge = taskdefs.OrchestrationEdge

func CloneOrchestrationDefinition(input *OrchestrationDefinition) *OrchestrationDefinition {
	return taskdefs.CloneOrchestrationDefinition(input)
}
