package group

import (
	"fmt"

	bridgeTasks "ghost-os/bridge/tasks"
)

type graphData struct {
	nodes         map[string]Node
	rawControlIn  map[string]int
	rawControlOut map[string]int
	controlIn     map[string]int
	controlOut    map[string]int
	controlNext   map[string]string
	groupMembers  map[string][]string
	groupCount    int
	edgeCount     int
}

func buildGraphData(definition *Definition) (graphData, error) {
	if definition == nil {
		return graphData{}, fmt.Errorf("%w: orchestration is required", bridgeTasks.ErrInvalidTaskConfig)
	}
	graph := newGraphData(len(definition.Nodes))
	for _, node := range definition.Nodes {
		if err := addNode(&graph, node); err != nil {
			return graphData{}, err
		}
	}
	for _, edge := range definition.Edges {
		if err := addEdge(graph, edge); err != nil {
			return graphData{}, err
		}
	}
	return graph, nil
}

func newGraphData(nodeCount int) graphData {
	return graphData{
		nodes:         make(map[string]Node, nodeCount),
		rawControlIn:  make(map[string]int, nodeCount),
		rawControlOut: make(map[string]int, nodeCount),
		controlIn:     make(map[string]int, nodeCount),
		controlOut:    make(map[string]int, nodeCount),
		controlNext:   make(map[string]string, nodeCount),
		groupMembers:  make(map[string][]string),
	}
}

func addNode(graph *graphData, node Node) error {
	if err := validateNode(node); err != nil {
		return err
	}
	if _, exists := graph.nodes[node.ID]; exists {
		return fmt.Errorf("%w: duplicate orchestration node id %q", bridgeTasks.ErrInvalidTaskConfig, node.ID)
	}
	graph.nodes[node.ID] = node
	countNodeType(graph, node.Type)
	return nil
}

func countNodeType(graph *graphData, nodeType string) {
	switch nodeType {
	case NodeTypeGroup:
		graph.groupCount++
	}
}

func (g graphData) plan() Plan {
	return Plan{
		Nodes:        g.nodes,
		EntryGroupID: findEntryGroupID(g),
		ControlNext:  g.controlNext,
		GroupMembers: g.groupMembers,
	}
}
