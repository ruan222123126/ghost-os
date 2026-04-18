package orchestration

import (
	"fmt"
	"strings"
)

type workflowGraph struct {
	outgoing  map[string][]string
	incoming  map[string][]string
	indegree  map[string]int
	outdegree map[string]int
}

type workflowGraphBuilder struct {
	index   workflowNodeIndex
	graph   workflowGraph
	edgeSet map[string]bool
}

func newWorkflowGraphBuilder(index workflowNodeIndex, edgeCount int) workflowGraphBuilder {
	graph := workflowGraph{
		outgoing:  make(map[string][]string, len(index.nodes)),
		incoming:  make(map[string][]string, len(index.nodes)),
		indegree:  make(map[string]int, len(index.nodes)),
		outdegree: make(map[string]int, len(index.nodes)),
	}
	for nodeID := range index.nodes {
		graph.outgoing[nodeID] = nil
		graph.incoming[nodeID] = nil
		graph.indegree[nodeID] = 0
		graph.outdegree[nodeID] = 0
	}
	return workflowGraphBuilder{
		index:   index,
		graph:   graph,
		edgeSet: make(map[string]bool, edgeCount),
	}
}

func buildWorkflowGraph(definition *WorkflowDefinition, index workflowNodeIndex) (workflowGraph, error) {
	builder := newWorkflowGraphBuilder(index, len(definition.Edges))
	for _, edge := range definition.Edges {
		if err := builder.connect(edge); err != nil {
			return workflowGraph{}, err
		}
	}
	return builder.graph, nil
}

func (b *workflowGraphBuilder) connect(edge WorkflowEdge) error {
	fromID := strings.TrimSpace(edge.FromNodeID)
	toID := strings.TrimSpace(edge.ToNodeID)
	if fromID == "" || toID == "" {
		return fmt.Errorf("%w: workflow edge endpoints are required", ErrInvalidTaskConfig)
	}
	if fromID == toID {
		return fmt.Errorf("%w: workflow does not allow self-loop edge %q", ErrInvalidTaskConfig, fromID)
	}
	if _, ok := b.index.nodes[fromID]; !ok {
		return fmt.Errorf("%w: workflow edge references unknown from_node_id %q", ErrInvalidTaskConfig, fromID)
	}
	if _, ok := b.index.nodes[toID]; !ok {
		return fmt.Errorf("%w: workflow edge references unknown to_node_id %q", ErrInvalidTaskConfig, toID)
	}
	key := fromID + "->" + toID
	if b.edgeSet[key] {
		return fmt.Errorf("%w: duplicate workflow edge %q", ErrInvalidTaskConfig, key)
	}
	b.edgeSet[key] = true
	b.graph.outgoing[fromID] = append(b.graph.outgoing[fromID], toID)
	b.graph.incoming[toID] = append(b.graph.incoming[toID], fromID)
	b.graph.outdegree[fromID]++
	b.graph.indegree[toID]++
	return nil
}

func validateWorkflowDegrees(index workflowNodeIndex, graph workflowGraph) error {
	for nodeID, node := range index.nodes {
		in := graph.indegree[nodeID]
		out := graph.outdegree[nodeID]
		if err := validateWorkflowNodeDegree(nodeID, node, in, out); err != nil {
			return err
		}
	}
	return nil
}

func validateWorkflowNodeDegree(nodeID string, node WorkflowNode, in int, out int) error {
	switch node.Type {
	case workflowNodeTypeStart:
		if in != 0 || out < 1 {
			return fmt.Errorf("%w: start node must have in=0 and out>=1", ErrInvalidTaskConfig)
		}
	case workflowNodeTypeEnd:
		if in < 1 || out != 0 {
			return fmt.Errorf("%w: end node must have in>=1 and out=0", ErrInvalidTaskConfig)
		}
	case workflowNodeTypeIf, workflowNodeTypeLoop:
		if in < 1 || out != 2 {
			return fmt.Errorf("%w: workflow node %q must have in>=1 and out=2", ErrInvalidTaskConfig, nodeID)
		}
	default:
		if in < 1 || out != 1 {
			return fmt.Errorf("%w: workflow node %q must have in>=1 and out=1", ErrInvalidTaskConfig, nodeID)
		}
	}
	return nil
}

func validateWorkflowControlTargets(index workflowNodeIndex, graph workflowGraph) error {
	for nodeID, node := range index.nodes {
		switch node.Type {
		case workflowNodeTypeIf:
			if err := validateWorkflowIfTargets(nodeID, node, index, graph); err != nil {
				return err
			}
		case workflowNodeTypeLoop:
			if err := validateWorkflowLoopTargets(nodeID, node, index, graph); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateWorkflowIfTargets(nodeID string, node WorkflowNode, index workflowNodeIndex, graph workflowGraph) error {
	if sourceID := strings.TrimSpace(node.If.SourceNodeID); sourceID != "" {
		if _, ok := index.nodes[sourceID]; !ok {
			return fmt.Errorf("%w: workflow if node %q references unknown source_node_id %q", ErrInvalidTaskConfig, nodeID, sourceID)
		}
	}
	if _, ok := index.nodes[node.If.TrueNodeID]; !ok {
		return fmt.Errorf("%w: workflow if node %q references unknown true_node_id %q", ErrInvalidTaskConfig, nodeID, node.If.TrueNodeID)
	}
	if _, ok := index.nodes[node.If.FalseNodeID]; !ok {
		return fmt.Errorf("%w: workflow if node %q references unknown false_node_id %q", ErrInvalidTaskConfig, nodeID, node.If.FalseNodeID)
	}
	if !workflowOutgoingMatchesTargets(graph.outgoing[nodeID], node.If.TrueNodeID, node.If.FalseNodeID) {
		return fmt.Errorf("%w: workflow if node %q outgoing edges must match true_node_id/false_node_id", ErrInvalidTaskConfig, nodeID)
	}
	return nil
}

func validateWorkflowLoopTargets(nodeID string, node WorkflowNode, index workflowNodeIndex, graph workflowGraph) error {
	if _, ok := index.nodes[node.Loop.BodyNodeID]; !ok {
		return fmt.Errorf("%w: workflow loop node %q references unknown body_node_id %q", ErrInvalidTaskConfig, nodeID, node.Loop.BodyNodeID)
	}
	if _, ok := index.nodes[node.Loop.ExitNodeID]; !ok {
		return fmt.Errorf("%w: workflow loop node %q references unknown exit_node_id %q", ErrInvalidTaskConfig, nodeID, node.Loop.ExitNodeID)
	}
	if !workflowOutgoingMatchesTargets(graph.outgoing[nodeID], node.Loop.BodyNodeID, node.Loop.ExitNodeID) {
		return fmt.Errorf("%w: workflow loop node %q outgoing edges must match body_node_id/exit_node_id", ErrInvalidTaskConfig, nodeID)
	}
	return nil
}

func workflowOutgoingMatchesTargets(outgoing []string, firstID string, secondID string) bool {
	if len(outgoing) != 2 {
		return false
	}
	firstSeen := false
	secondSeen := false
	for _, nodeID := range outgoing {
		if nodeID == firstID {
			firstSeen = true
		}
		if nodeID == secondID {
			secondSeen = true
		}
	}
	return firstSeen && secondSeen
}

func validateWorkflowConnectivity(index workflowNodeIndex, graph workflowGraph) error {
	fromStart := traverseWorkflowGraph(index.startID, graph.outgoing)
	if len(fromStart) != len(index.nodes) {
		return fmt.Errorf("%w: workflow must be fully connected from start node", ErrInvalidTaskConfig)
	}
	toEnd := traverseWorkflowGraph(index.endID, graph.incoming)
	for nodeID := range index.nodes {
		if !toEnd[nodeID] {
			return fmt.Errorf("%w: workflow node %q cannot reach end node", ErrInvalidTaskConfig, nodeID)
		}
	}
	return validateWorkflowLoopCycles(index, graph)
}

func validateWorkflowLoopCycles(index workflowNodeIndex, graph workflowGraph) error {
	for nodeID, node := range index.nodes {
		if node.Type != workflowNodeTypeLoop {
			continue
		}
		if workflowPathExists(graph.outgoing, node.Loop.BodyNodeID, nodeID) {
			continue
		}
		return fmt.Errorf("%w: workflow loop node %q body path must return to the loop node", ErrInvalidTaskConfig, nodeID)
	}
	return nil
}

func traverseWorkflowGraph(startID string, adjacency map[string][]string) map[string]bool {
	seen := make(map[string]bool, len(adjacency))
	stack := []string{startID}
	for len(stack) > 0 {
		last := len(stack) - 1
		nodeID := stack[last]
		stack = stack[:last]
		if seen[nodeID] {
			continue
		}
		seen[nodeID] = true
		for _, nextID := range adjacency[nodeID] {
			if !seen[nextID] {
				stack = append(stack, nextID)
			}
		}
	}
	return seen
}

func workflowPathExists(adjacency map[string][]string, startID string, targetID string) bool {
	seen := make(map[string]bool, len(adjacency))
	stack := []string{startID}
	for len(stack) > 0 {
		last := len(stack) - 1
		nodeID := stack[last]
		stack = stack[:last]
		if seen[nodeID] {
			continue
		}
		if nodeID == targetID {
			return true
		}
		seen[nodeID] = true
		for _, nextID := range adjacency[nodeID] {
			if !seen[nextID] {
				stack = append(stack, nextID)
			}
		}
	}
	return false
}

type workflowCycleState struct {
	color    map[string]int
	stack    []string
	position map[string]int
}

func validateWorkflowCycles(index workflowNodeIndex, graph workflowGraph) error {
	state := workflowCycleState{
		color:    make(map[string]int, len(index.nodes)),
		stack:    make([]string, 0, len(index.nodes)),
		position: make(map[string]int, len(index.nodes)),
	}
	for nodeID := range index.nodes {
		if state.color[nodeID] != 0 {
			continue
		}
		if err := walkWorkflowCycle(nodeID, index, graph, &state); err != nil {
			return err
		}
	}
	return nil
}

func walkWorkflowCycle(nodeID string, index workflowNodeIndex, graph workflowGraph, state *workflowCycleState) error {
	state.color[nodeID] = 1
	state.position[nodeID] = len(state.stack)
	state.stack = append(state.stack, nodeID)
	for _, nextID := range graph.outgoing[nodeID] {
		if state.color[nextID] == 0 {
			if err := walkWorkflowCycle(nextID, index, graph, state); err != nil {
				return err
			}
		}
		if state.color[nextID] == 1 {
			cycle := append([]string(nil), state.stack[state.position[nextID]:]...)
			if workflowCycleContainsLoopNode(cycle, index) {
				continue
			}
			return fmt.Errorf("%w: workflow cycle must include a loop node: %s", ErrInvalidTaskConfig, strings.Join(cycle, " -> "))
		}
	}
	state.color[nodeID] = 2
	delete(state.position, nodeID)
	state.stack = state.stack[:len(state.stack)-1]
	return nil
}

func workflowCycleContainsLoopNode(cycle []string, index workflowNodeIndex) bool {
	for _, nodeID := range cycle {
		if index.nodes[nodeID].Type == workflowNodeTypeLoop {
			return true
		}
	}
	return false
}
