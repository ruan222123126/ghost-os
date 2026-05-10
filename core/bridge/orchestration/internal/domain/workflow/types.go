package workflow

import bridgeTasks "ghost-os/bridge/tasks"

const (
	NodeTypeStart = "start"
	NodeTypeTool  = "tool"
	NodeTypeLLM   = "llm"
	NodeTypeAgent = "agent"
	NodeTypeIf    = "if"
	NodeTypeLoop  = "loop"
	NodeTypeEnd   = "end"

	InputTypeString  = "string"
	InputTypeNumber  = "number"
	InputTypeBoolean = "boolean"
	InputTypeObject  = "object"
	InputTypeArray   = "array"

	IfOperatorEquals      = "equals"
	IfOperatorNotEquals   = "not_equals"
	IfOperatorContains    = "contains"
	IfOperatorNotContains = "not_contains"
	IfOperatorIsEmpty     = "is_empty"
	IfOperatorNotEmpty    = "not_empty"
)

type Definition = bridgeTasks.WorkflowDefinition
type Node = bridgeTasks.WorkflowNode
type StartNode = bridgeTasks.WorkflowStartNode
type InputVariable = bridgeTasks.WorkflowInputVariable
type ToolNode = bridgeTasks.WorkflowToolNode
type LLMNode = bridgeTasks.WorkflowLLMNode
type AgentNode = bridgeTasks.WorkflowAgentNode
type IfNode = bridgeTasks.WorkflowIfNode
type LoopNode = bridgeTasks.WorkflowLoopNode
type Edge = bridgeTasks.WorkflowEdge
type RuntimeOverrides = bridgeTasks.TaskRuntimeOverrides

type NodeIndex struct {
	nodes   map[string]Node
	startID string
	endID   string
}

type Plan struct {
	nodes   map[string]Node
	startID string
	endID   string
	graph   Graph
}

type Graph struct {
	outgoing  map[string][]string
	incoming  map[string][]string
	indegree  map[string]int
	outdegree map[string]int
}

type NodePayloadSpec struct {
	allowStart bool
	tool       bool
	llm        bool
	agent      bool
	ifNode     bool
	loop       bool
}

type NodeValidationSpec struct {
	payload  NodePayloadSpec
	validate func(Node) error
}

type PlanBuilder struct{}
