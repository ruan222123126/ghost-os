package workflow

import (
	"fmt"
	"strings"

	taskdefs "ghost-os/bridge/taskdefs"
)

var nodeValidationSpecs = map[string]NodeValidationSpec{
	NodeTypeStart: {payload: NodePayloadSpec{allowStart: true}, validate: validateStartNode},
	NodeTypeTool:  {payload: NodePayloadSpec{tool: true}, validate: validateToolNode},
	NodeTypeLLM:   {payload: NodePayloadSpec{llm: true}, validate: validateLLMNode},
	NodeTypeAgent: {payload: NodePayloadSpec{agent: true}, validate: validateAgentNode},
	NodeTypeIf:    {payload: NodePayloadSpec{ifNode: true}, validate: validateIfNode},
	NodeTypeLoop:  {payload: NodePayloadSpec{loop: true}, validate: validateLoopNode},
	NodeTypeEnd:   {},
}

func (PlanBuilder) Build(definition *Definition) (Plan, error) {
	return BuildPlan(definition)
}

func BuildPlan(definition *Definition) (Plan, error) {
	if definition == nil {
		return Plan{}, fmt.Errorf("%w: workflow is required", taskdefs.ErrInvalidTaskConfig)
	}
	index, err := buildNodeIndex(definition)
	if err != nil {
		return Plan{}, err
	}
	graph, err := buildGraph(definition, index)
	if err != nil {
		return Plan{}, err
	}
	if err := validateGraph(index, graph); err != nil {
		return Plan{}, err
	}
	return Plan{nodes: index.nodes, startID: index.startID, endID: index.endID, graph: graph}, nil
}

func buildNodeIndex(definition *Definition) (NodeIndex, error) {
	index := NodeIndex{nodes: make(map[string]Node, len(definition.Nodes))}
	for _, node := range definition.Nodes {
		if err := validateNode(node); err != nil {
			return NodeIndex{}, err
		}
		if _, exists := index.nodes[node.ID]; exists {
			return NodeIndex{}, fmt.Errorf("%w: duplicate workflow node id %q", taskdefs.ErrInvalidTaskConfig, node.ID)
		}
		index.nodes[node.ID] = node
		index.rememberBoundary(node)
	}
	return requireBoundaryNodes(index)
}

func (i *NodeIndex) rememberBoundary(node Node) {
	if node.Type == NodeTypeStart {
		i.startID = node.ID
	}
	if node.Type == NodeTypeEnd {
		i.endID = node.ID
	}
}

func requireBoundaryNodes(index NodeIndex) (NodeIndex, error) {
	if countNodes(index, NodeTypeStart) != 1 {
		return NodeIndex{}, fmt.Errorf("%w: workflow requires exactly 1 start node", taskdefs.ErrInvalidTaskConfig)
	}
	if countNodes(index, NodeTypeEnd) != 1 {
		return NodeIndex{}, fmt.Errorf("%w: workflow requires exactly 1 end node", taskdefs.ErrInvalidTaskConfig)
	}
	return index, nil
}

func validateNode(node Node) error {
	if node.ID == "" {
		return fmt.Errorf("%w: workflow node id is required", taskdefs.ErrInvalidTaskConfig)
	}
	spec, ok := nodeValidationSpecs[node.Type]
	if !ok {
		return fmt.Errorf("%w: unsupported workflow node type %q", taskdefs.ErrInvalidTaskConfig, node.Type)
	}
	if err := validateNodePayload(node, spec.payload); err != nil {
		return err
	}
	if spec.validate == nil {
		return nil
	}
	return spec.validate(node)
}

func validateNodePayload(node Node, spec NodePayloadSpec) error {
	matches := (!hasStartPayload(node) || spec.allowStart) &&
		(node.Tool != nil) == spec.tool &&
		(node.LLM != nil) == spec.llm &&
		(node.Agent != nil) == spec.agent &&
		(node.If != nil) == spec.ifNode &&
		(node.Loop != nil) == spec.loop
	if matches {
		return nil
	}
	return fmt.Errorf("%w: workflow node %q payload does not match type %q", taskdefs.ErrInvalidTaskConfig, node.ID, node.Type)
}

func hasStartPayload(node Node) bool {
	return node.Start != nil
}

func validateToolNode(node Node) error {
	if strings.TrimSpace(node.Tool.ToolName) == "" {
		return fmt.Errorf("%w: workflow tool node %q requires tool_name", taskdefs.ErrInvalidTaskConfig, node.ID)
	}
	return nil
}

func validateLLMNode(node Node) error {
	if strings.TrimSpace(node.LLM.Prompt) == "" {
		return fmt.Errorf("%w: workflow llm node %q requires prompt", taskdefs.ErrInvalidTaskConfig, node.ID)
	}
	return nil
}

func validateAgentNode(node Node) error {
	if strings.TrimSpace(node.Agent.Message) == "" {
		return fmt.Errorf("%w: workflow agent node %q requires message", taskdefs.ErrInvalidTaskConfig, node.ID)
	}
	return nil
}
