package orchestration

import "fmt"

func (p workflowExecutionPlan) node(nodeID string) (WorkflowNode, bool) {
	node, ok := p.nodes[nodeID]
	return node, ok
}

func (p workflowExecutionPlan) nextNodeIDs(nodeID string) []string {
	return append([]string(nil), p.graph.outgoing[nodeID]...)
}

func (p workflowExecutionPlan) singleNextNodeID(nodeID string) (string, error) {
	nextIDs := p.nextNodeIDs(nodeID)
	if len(nextIDs) != 1 {
		return "", fmt.Errorf("%w: workflow node %q must have exactly one outgoing edge", ErrInvalidTaskConfig, nodeID)
	}
	return nextIDs[0], nil
}

func (p workflowExecutionPlan) needsRuntimeDependencies() bool {
	for _, node := range p.nodes {
		if node.Type == workflowNodeTypeTool || node.Type == workflowNodeTypeLLM {
			return true
		}
	}
	return false
}

func (p workflowExecutionPlan) needsService() bool {
	for _, node := range p.nodes {
		if node.Type == workflowNodeTypeTool || node.Type == workflowNodeTypeLLM || node.Type == workflowNodeTypeAgent {
			return true
		}
	}
	return false
}
