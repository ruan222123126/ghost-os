package workflow

import taskdefs "ghost-os/bridge/taskdefs"

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

type Definition = taskdefs.WorkflowDefinition
type Node = taskdefs.WorkflowNode
type StartNode = taskdefs.WorkflowStartNode
type InputVariable = taskdefs.WorkflowInputVariable
type ToolNode = taskdefs.WorkflowToolNode
type LLMNode = taskdefs.WorkflowLLMNode
type AgentNode = taskdefs.WorkflowAgentNode
type IfNode = taskdefs.WorkflowIfNode
type LoopNode = taskdefs.WorkflowLoopNode
type Edge = taskdefs.WorkflowEdge
type RuntimeOverrides = taskdefs.TaskRuntimeOverrides

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
