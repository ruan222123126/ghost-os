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
	workflowNodeTypeEnd   = "end"
)

type workflowNodeIndex struct {
	nodes   map[string]WorkflowNode
	startID string
	endID   string
}

type workflowGraph struct {
	next      map[string]string
	indegree  map[string]int
	outdegree map[string]int
}

type workflowExecutionPlan struct {
	ordered []WorkflowNode
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
	ordered, err := orderWorkflowNodes(index, graph)
	if err != nil {
		return workflowExecutionPlan{}, err
	}
	return workflowExecutionPlan{ordered: ordered}, nil
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
	switch node.Type {
	case workflowNodeTypeStart:
		return validateWorkflowNodePayload(node, false, false, false)
	case workflowNodeTypeTool:
		if err := validateWorkflowNodePayload(node, true, false, false); err != nil {
			return err
		}
		if strings.TrimSpace(node.Tool.ToolName) == "" {
			return fmt.Errorf("%w: workflow tool node %q requires tool_name", ErrInvalidTaskConfig, node.ID)
		}
		return nil
	case workflowNodeTypeLLM:
		if err := validateWorkflowNodePayload(node, false, true, false); err != nil {
			return err
		}
		if strings.TrimSpace(node.LLM.Prompt) == "" {
			return fmt.Errorf("%w: workflow llm node %q requires prompt", ErrInvalidTaskConfig, node.ID)
		}
		return nil
	case workflowNodeTypeAgent:
		if err := validateWorkflowNodePayload(node, false, false, true); err != nil {
			return err
		}
		if strings.TrimSpace(node.Agent.Message) == "" {
			return fmt.Errorf("%w: workflow agent node %q requires message", ErrInvalidTaskConfig, node.ID)
		}
		return nil
	case workflowNodeTypeEnd:
		return validateWorkflowNodePayload(node, false, false, false)
	default:
		return fmt.Errorf("%w: unsupported workflow node type %q", ErrInvalidTaskConfig, node.Type)
	}
}

func validateWorkflowNodePayload(
	node WorkflowNode,
	wantTool bool,
	wantLLM bool,
	wantAgent bool,
) error {
	hasTool := node.Tool != nil
	hasLLM := node.LLM != nil
	hasAgent := node.Agent != nil
	if hasTool == wantTool && hasLLM == wantLLM && hasAgent == wantAgent {
		return nil
	}
	return fmt.Errorf("%w: workflow node %q payload does not match type %q", ErrInvalidTaskConfig, node.ID, node.Type)
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

func buildWorkflowGraph(definition *WorkflowDefinition, index workflowNodeIndex) (workflowGraph, error) {
	graph := workflowGraph{
		next:      make(map[string]string, len(definition.Edges)),
		indegree:  make(map[string]int, len(index.nodes)),
		outdegree: make(map[string]int, len(index.nodes)),
	}
	if len(definition.Edges) != len(index.nodes)-1 {
		return workflowGraph{}, fmt.Errorf("%w: workflow requires exactly len(nodes)-1 edges", ErrInvalidTaskConfig)
	}
	for _, edge := range definition.Edges {
		if err := connectWorkflowEdge(edge, index, &graph); err != nil {
			return workflowGraph{}, err
		}
	}
	return graph, nil
}

func connectWorkflowEdge(edge WorkflowEdge, index workflowNodeIndex, graph *workflowGraph) error {
	if edge.FromNodeID == "" || edge.ToNodeID == "" {
		return fmt.Errorf("%w: workflow edge endpoints are required", ErrInvalidTaskConfig)
	}
	if edge.FromNodeID == edge.ToNodeID {
		return fmt.Errorf("%w: workflow does not allow self-loop edge %q", ErrInvalidTaskConfig, edge.FromNodeID)
	}
	if _, ok := index.nodes[edge.FromNodeID]; !ok {
		return fmt.Errorf("%w: workflow edge references unknown from_node_id %q", ErrInvalidTaskConfig, edge.FromNodeID)
	}
	if _, ok := index.nodes[edge.ToNodeID]; !ok {
		return fmt.Errorf("%w: workflow edge references unknown to_node_id %q", ErrInvalidTaskConfig, edge.ToNodeID)
	}
	if nextID, exists := graph.next[edge.FromNodeID]; exists && nextID != edge.ToNodeID {
		return fmt.Errorf("%w: workflow branching is not supported", ErrInvalidTaskConfig)
	}
	graph.next[edge.FromNodeID] = edge.ToNodeID
	graph.outdegree[edge.FromNodeID]++
	graph.indegree[edge.ToNodeID]++
	return nil
}

func validateWorkflowDegrees(index workflowNodeIndex, graph workflowGraph) error {
	for id, node := range index.nodes {
		in := graph.indegree[id]
		out := graph.outdegree[id]
		switch node.Type {
		case workflowNodeTypeStart:
			if in != 0 || out != 1 {
				return fmt.Errorf("%w: start node must have in=0 and out=1", ErrInvalidTaskConfig)
			}
		case workflowNodeTypeEnd:
			if in != 1 || out != 0 {
				return fmt.Errorf("%w: end node must have in=1 and out=0", ErrInvalidTaskConfig)
			}
		default:
			if in != 1 || out != 1 {
				return fmt.Errorf("%w: workflow node %q must have in=1 and out=1", ErrInvalidTaskConfig, id)
			}
		}
	}
	return nil
}

func orderWorkflowNodes(index workflowNodeIndex, graph workflowGraph) ([]WorkflowNode, error) {
	ordered := make([]WorkflowNode, 0, len(index.nodes))
	seen := make(map[string]bool, len(index.nodes))
	current := index.startID
	for {
		if seen[current] {
			return nil, fmt.Errorf("%w: workflow must not contain cycles", ErrInvalidTaskConfig)
		}
		seen[current] = true
		ordered = append(ordered, index.nodes[current])
		nextID, ok := graph.next[current]
		if !ok {
			break
		}
		current = nextID
	}
	if current != index.endID {
		return nil, fmt.Errorf("%w: workflow does not reach end node", ErrInvalidTaskConfig)
	}
	if len(seen) != len(index.nodes) {
		return nil, fmt.Errorf("%w: workflow must be fully connected", ErrInvalidTaskConfig)
	}
	return ordered, nil
}

func (p workflowExecutionPlan) executableNodes() []WorkflowNode {
	if len(p.ordered) <= 2 {
		return nil
	}
	return append([]WorkflowNode(nil), p.ordered[1:len(p.ordered)-1]...)
}
