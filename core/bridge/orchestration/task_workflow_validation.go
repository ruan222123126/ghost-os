package orchestration

import (
	"fmt"
	"strings"
)

const (
	workflowNodeTypeStart = "start"
	workflowNodeTypeTool  = "tool"
	workflowNodeTypeLLM   = "llm"
	workflowNodeTypeAgent = "agent"
	workflowNodeTypeIf    = "if"
	workflowNodeTypeLoop  = "loop"
	workflowNodeTypeEnd   = "end"

	workflowInputTypeString  = "string"
	workflowInputTypeNumber  = "number"
	workflowInputTypeBoolean = "boolean"
	workflowInputTypeObject  = "object"
	workflowInputTypeArray   = "array"

	workflowIfOperatorEquals      = "equals"
	workflowIfOperatorNotEquals   = "not_equals"
	workflowIfOperatorContains    = "contains"
	workflowIfOperatorNotContains = "not_contains"
	workflowIfOperatorIsEmpty     = "is_empty"
	workflowIfOperatorNotEmpty    = "not_empty"
)

type workflowNodeIndex struct {
	nodes   map[string]WorkflowNode
	startID string
	endID   string
}

type workflowExecutionPlan struct {
	nodes   map[string]WorkflowNode
	startID string
	endID   string
	graph   workflowGraph
}

type workflowNodePayloadSpec struct {
	allowStart bool
	tool       bool
	llm        bool
	agent      bool
	ifNode     bool
	loop       bool
}

type workflowNodeValidationSpec struct {
	payload  workflowNodePayloadSpec
	validate func(WorkflowNode) error
}

var workflowNodeValidationSpecs = map[string]workflowNodeValidationSpec{
	workflowNodeTypeStart: {
		payload:  workflowNodePayloadSpec{allowStart: true},
		validate: validateWorkflowStartNode,
	},
	workflowNodeTypeTool: {
		payload:  workflowNodePayloadSpec{tool: true},
		validate: validateWorkflowToolNode,
	},
	workflowNodeTypeLLM: {
		payload:  workflowNodePayloadSpec{llm: true},
		validate: validateWorkflowLLMNode,
	},
	workflowNodeTypeAgent: {
		payload:  workflowNodePayloadSpec{agent: true},
		validate: validateWorkflowAgentNode,
	},
	workflowNodeTypeIf: {
		payload:  workflowNodePayloadSpec{ifNode: true},
		validate: validateWorkflowIfNode,
	},
	workflowNodeTypeLoop: {
		payload:  workflowNodePayloadSpec{loop: true},
		validate: validateWorkflowLoopNode,
	},
	workflowNodeTypeEnd: {},
}

func validateWorkflowTaskDefinition(task *ScheduledTask) error {
	if task.Workflow == nil {
		return fmt.Errorf("%w: workflow is required for workflow task", ErrInvalidTaskConfig)
	}
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
	if definition == nil {
		return workflowExecutionPlan{}, fmt.Errorf("%w: workflow is required", ErrInvalidTaskConfig)
	}
	index, err := buildWorkflowNodeIndex(definition)
	if err != nil {
		return workflowExecutionPlan{}, err
	}
	graph, err := buildWorkflowGraph(definition, index)
	if err != nil {
		return workflowExecutionPlan{}, err
	}
	if err := validateWorkflowDegrees(index, graph); err != nil {
		return workflowExecutionPlan{}, err
	}
	if err := validateWorkflowControlTargets(index, graph); err != nil {
		return workflowExecutionPlan{}, err
	}
	if err := validateWorkflowConnectivity(index, graph); err != nil {
		return workflowExecutionPlan{}, err
	}
	if err := validateWorkflowCycles(index, graph); err != nil {
		return workflowExecutionPlan{}, err
	}
	return workflowExecutionPlan{
		nodes:   index.nodes,
		startID: index.startID,
		endID:   index.endID,
		graph:   graph,
	}, nil
}

func buildWorkflowNodeIndex(definition *WorkflowDefinition) (workflowNodeIndex, error) {
	index := workflowNodeIndex{nodes: make(map[string]WorkflowNode, len(definition.Nodes))}
	for _, node := range definition.Nodes {
		if err := validateWorkflowNode(node); err != nil {
			return workflowNodeIndex{}, err
		}
		if _, exists := index.nodes[node.ID]; exists {
			return workflowNodeIndex{}, fmt.Errorf("%w: duplicate workflow node id %q", ErrInvalidTaskConfig, node.ID)
		}
		index.nodes[node.ID] = node
		if node.Type == workflowNodeTypeStart {
			index.startID = node.ID
		}
		if node.Type == workflowNodeTypeEnd {
			index.endID = node.ID
		}
	}
	if countWorkflowNodes(index, workflowNodeTypeStart) != 1 {
		return workflowNodeIndex{}, fmt.Errorf("%w: workflow requires exactly 1 start node", ErrInvalidTaskConfig)
	}
	if countWorkflowNodes(index, workflowNodeTypeEnd) != 1 {
		return workflowNodeIndex{}, fmt.Errorf("%w: workflow requires exactly 1 end node", ErrInvalidTaskConfig)
	}
	return index, nil
}

func validateWorkflowNode(node WorkflowNode) error {
	if node.ID == "" {
		return fmt.Errorf("%w: workflow node id is required", ErrInvalidTaskConfig)
	}
	spec, ok := workflowNodeValidationSpecs[node.Type]
	if !ok {
		return fmt.Errorf("%w: unsupported workflow node type %q", ErrInvalidTaskConfig, node.Type)
	}
	if err := validateWorkflowNodePayload(node, spec.payload); err != nil {
		return err
	}
	if spec.validate == nil {
		return nil
	}
	return spec.validate(node)
}

func validateWorkflowNodePayload(node WorkflowNode, spec workflowNodePayloadSpec) error {
	hasStart := node.Start != nil
	hasTool := node.Tool != nil
	hasLLM := node.LLM != nil
	hasAgent := node.Agent != nil
	hasIf := node.If != nil
	hasLoop := node.Loop != nil
	if (!hasStart || spec.allowStart) &&
		hasTool == spec.tool &&
		hasLLM == spec.llm &&
		hasAgent == spec.agent &&
		hasIf == spec.ifNode &&
		hasLoop == spec.loop {
		return nil
	}
	return fmt.Errorf("%w: workflow node %q payload does not match type %q", ErrInvalidTaskConfig, node.ID, node.Type)
}

func validateWorkflowToolNode(node WorkflowNode) error {
	if strings.TrimSpace(node.Tool.ToolName) == "" {
		return fmt.Errorf("%w: workflow tool node %q requires tool_name", ErrInvalidTaskConfig, node.ID)
	}
	return nil
}

func validateWorkflowLLMNode(node WorkflowNode) error {
	if strings.TrimSpace(node.LLM.Prompt) == "" {
		return fmt.Errorf("%w: workflow llm node %q requires prompt", ErrInvalidTaskConfig, node.ID)
	}
	return nil
}

func validateWorkflowAgentNode(node WorkflowNode) error {
	if strings.TrimSpace(node.Agent.Message) == "" {
		return fmt.Errorf("%w: workflow agent node %q requires message", ErrInvalidTaskConfig, node.ID)
	}
	return nil
}

func validateWorkflowIfNode(node WorkflowNode) error {
	if !isWorkflowIfOperatorSupported(node.If.Operator) {
		return fmt.Errorf("%w: workflow if node %q uses unsupported operator %q", ErrInvalidTaskConfig, node.ID, node.If.Operator)
	}
	if strings.TrimSpace(node.If.TrueNodeID) == "" || strings.TrimSpace(node.If.FalseNodeID) == "" {
		return fmt.Errorf("%w: workflow if node %q requires true_node_id and false_node_id", ErrInvalidTaskConfig, node.ID)
	}
	if node.If.TrueNodeID == node.If.FalseNodeID {
		return fmt.Errorf("%w: workflow if node %q true_node_id and false_node_id must differ", ErrInvalidTaskConfig, node.ID)
	}
	if requiresWorkflowIfValue(node.If.Operator) && strings.TrimSpace(node.If.Value) == "" {
		return fmt.Errorf("%w: workflow if node %q requires value for operator %q", ErrInvalidTaskConfig, node.ID, node.If.Operator)
	}
	return nil
}

func validateWorkflowLoopNode(node WorkflowNode) error {
	if node.Loop.MaxIterations <= 0 {
		return fmt.Errorf("%w: workflow loop node %q requires max_iterations > 0", ErrInvalidTaskConfig, node.ID)
	}
	if strings.TrimSpace(node.Loop.BodyNodeID) == "" || strings.TrimSpace(node.Loop.ExitNodeID) == "" {
		return fmt.Errorf("%w: workflow loop node %q requires body_node_id and exit_node_id", ErrInvalidTaskConfig, node.ID)
	}
	if node.Loop.BodyNodeID == node.Loop.ExitNodeID {
		return fmt.Errorf("%w: workflow loop node %q body_node_id and exit_node_id must differ", ErrInvalidTaskConfig, node.ID)
	}
	return nil
}

func isWorkflowIfOperatorSupported(operator string) bool {
	switch strings.TrimSpace(operator) {
	case workflowIfOperatorEquals,
		workflowIfOperatorNotEquals,
		workflowIfOperatorContains,
		workflowIfOperatorNotContains,
		workflowIfOperatorIsEmpty,
		workflowIfOperatorNotEmpty:
		return true
	default:
		return false
	}
}

func requiresWorkflowIfValue(operator string) bool {
	switch strings.TrimSpace(operator) {
	case workflowIfOperatorIsEmpty, workflowIfOperatorNotEmpty:
		return false
	default:
		return true
	}
}

func countWorkflowNodes(index workflowNodeIndex, want string) int {
	count := 0
	for _, node := range index.nodes {
		if node.Type == want {
			count++
		}
	}
	return count
}
