package orchestration

import "fmt"

const (
	workflowNodeTypeStart = "start"
	workflowNodeTypeEnd   = "end"
)

type workflowNodeIndex struct {
	types   map[string]string
	startID string
	endID   string
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
	return validateWorkflowDefinition(task.Workflow)
}

func validateWorkflowDefinition(definition *WorkflowDefinition) error {
	index, err := buildWorkflowNodeIndex(definition)
	if err != nil {
		return err
	}
	if len(definition.Edges) != 1 {
		return fmt.Errorf("%w: workflow requires exactly 1 edge", ErrInvalidTaskConfig)
	}
	return validateWorkflowEdge(definition.Edges[0], index)
}

func buildWorkflowNodeIndex(definition *WorkflowDefinition) (workflowNodeIndex, error) {
	index := workflowNodeIndex{types: make(map[string]string, len(definition.Nodes))}
	for _, node := range definition.Nodes {
		if node.ID == "" {
			return workflowNodeIndex{}, fmt.Errorf("%w: workflow node id is required", ErrInvalidTaskConfig)
		}
		if _, exists := index.types[node.ID]; exists {
			return workflowNodeIndex{}, fmt.Errorf("%w: duplicate workflow node id %q", ErrInvalidTaskConfig, node.ID)
		}
		if node.Type != workflowNodeTypeStart && node.Type != workflowNodeTypeEnd {
			return workflowNodeIndex{}, fmt.Errorf("%w: unsupported workflow node type %q", ErrInvalidTaskConfig, node.Type)
		}
		index.types[node.ID] = node.Type
		if node.Type == workflowNodeTypeStart {
			index.startID = node.ID
		}
		if node.Type == workflowNodeTypeEnd {
			index.endID = node.ID
		}
	}
	if index.startID == "" || countWorkflowNodeType(index.types, workflowNodeTypeStart) != 1 {
		return workflowNodeIndex{}, fmt.Errorf("%w: workflow requires exactly 1 start node", ErrInvalidTaskConfig)
	}
	if index.endID == "" || countWorkflowNodeType(index.types, workflowNodeTypeEnd) != 1 {
		return workflowNodeIndex{}, fmt.Errorf("%w: workflow requires exactly 1 end node", ErrInvalidTaskConfig)
	}
	return index, nil
}

func countWorkflowNodeType(nodeTypes map[string]string, want string) int {
	count := 0
	for _, nodeType := range nodeTypes {
		if nodeType == want {
			count++
		}
	}
	return count
}

func validateWorkflowEdge(edge WorkflowEdge, index workflowNodeIndex) error {
	if edge.FromNodeID == "" || edge.ToNodeID == "" {
		return fmt.Errorf("%w: workflow edge endpoints are required", ErrInvalidTaskConfig)
	}
	if edge.FromNodeID == edge.ToNodeID {
		return fmt.Errorf("%w: workflow does not allow self-loop edge %q", ErrInvalidTaskConfig, edge.FromNodeID)
	}
	fromType, ok := index.types[edge.FromNodeID]
	if !ok {
		return fmt.Errorf("%w: workflow edge references unknown from_node_id %q", ErrInvalidTaskConfig, edge.FromNodeID)
	}
	toType, ok := index.types[edge.ToNodeID]
	if !ok {
		return fmt.Errorf("%w: workflow edge references unknown to_node_id %q", ErrInvalidTaskConfig, edge.ToNodeID)
	}
	if fromType != workflowNodeTypeStart {
		return fmt.Errorf("%w: start node must be the only edge source", ErrInvalidTaskConfig)
	}
	if toType != workflowNodeTypeEnd {
		return fmt.Errorf("%w: end node must be the only edge target", ErrInvalidTaskConfig)
	}
	return nil
}
