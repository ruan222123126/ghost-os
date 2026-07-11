package group

import taskdefs "ghost-os/bridge/taskdefs"

const (
	NodeTypeStart = "start"
	NodeTypeGroup = taskdefs.OrchestrationNodeTypeGroup
	NodeTypeAgent = taskdefs.OrchestrationNodeTypeAgent
	NodeTypeEnd   = "end"

	EdgeKindControl = taskdefs.OrchestrationEdgeKindControl
	EdgeKindMember  = taskdefs.OrchestrationEdgeKindMember

	SpeakingModeSequential = taskdefs.OrchestrationSpeakingModeSequential
	SpeakingModeParallel   = taskdefs.OrchestrationSpeakingModeParallel
	SpeakingModeOwner      = taskdefs.OrchestrationSpeakingModeOwner
)

type Definition = taskdefs.OrchestrationDefinition
type Node = taskdefs.OrchestrationNode
type GroupNode = taskdefs.OrchestrationGroupNode
type AgentNode = taskdefs.OrchestrationAgentNode
type Edge = taskdefs.OrchestrationEdge
type RuntimeOverrides = taskdefs.TaskRuntimeOverrides
