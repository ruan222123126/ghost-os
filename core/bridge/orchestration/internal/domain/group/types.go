package group

import bridgeTasks "ghost-os/bridge/tasks"

const (
	NodeTypeStart = "start"
	NodeTypeGroup = bridgeTasks.OrchestrationNodeTypeGroup
	NodeTypeAgent = bridgeTasks.OrchestrationNodeTypeAgent
	NodeTypeEnd   = "end"

	EdgeKindControl = bridgeTasks.OrchestrationEdgeKindControl
	EdgeKindMember  = bridgeTasks.OrchestrationEdgeKindMember

	SpeakingModeSequential = bridgeTasks.OrchestrationSpeakingModeSequential
	SpeakingModeParallel   = bridgeTasks.OrchestrationSpeakingModeParallel
	SpeakingModeOwner      = bridgeTasks.OrchestrationSpeakingModeOwner
)

type Definition = bridgeTasks.OrchestrationDefinition
type Node = bridgeTasks.OrchestrationNode
type GroupNode = bridgeTasks.OrchestrationGroupNode
type AgentNode = bridgeTasks.OrchestrationAgentNode
type Edge = bridgeTasks.OrchestrationEdge
type RuntimeOverrides = bridgeTasks.TaskRuntimeOverrides
