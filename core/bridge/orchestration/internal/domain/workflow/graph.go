package workflow

import (
	"fmt"
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
)

type graphBuilder struct {
	index   NodeIndex
	graph   Graph
	edgeSet map[string]bool
}

func newGraphBuilder(index NodeIndex, edgeCount int) graphBuilder {
	graph := Graph{
		outgoing:  make(map[string][]string, len(index.nodes)),
		incoming:  make(map[string][]string, len(index.nodes)),
		indegree:  make(map[string]int, len(index.nodes)),
		outdegree: make(map[string]int, len(index.nodes)),
	}
	for nodeID := range index.nodes {
		graph.outgoing[nodeID] = nil
		graph.incoming[nodeID] = nil
	}
	return graphBuilder{index: index, graph: graph, edgeSet: make(map[string]bool, edgeCount)}
}

func buildGraph(definition *Definition, index NodeIndex) (Graph, error) {
	builder := newGraphBuilder(index, len(definition.Edges))
	for _, edge := range definition.Edges {
		if err := builder.connect(edge); err != nil {
			return Graph{}, err
		}
	}
	return builder.graph, nil
}

func (b *graphBuilder) connect(edge Edge) error {
	fromID := strings.TrimSpace(edge.FromNodeID)
	toID := strings.TrimSpace(edge.ToNodeID)
	if fromID == "" || toID == "" {
		return fmt.Errorf("%w: workflow edge endpoints are required", bridgeTasks.ErrInvalidTaskConfig)
	}
	if fromID == toID {
		return fmt.Errorf("%w: workflow does not allow self-loop edge %q", bridgeTasks.ErrInvalidTaskConfig, fromID)
	}
	if _, ok := b.index.nodes[fromID]; !ok {
		return fmt.Errorf("%w: workflow edge references unknown from_node_id %q", bridgeTasks.ErrInvalidTaskConfig, fromID)
	}
	if _, ok := b.index.nodes[toID]; !ok {
		return fmt.Errorf("%w: workflow edge references unknown to_node_id %q", bridgeTasks.ErrInvalidTaskConfig, toID)
	}
	return b.connectKnownNodes(fromID, toID)
}

func (b *graphBuilder) connectKnownNodes(fromID string, toID string) error {
	key := fromID + "->" + toID
	if b.edgeSet[key] {
		return fmt.Errorf("%w: duplicate workflow edge %q", bridgeTasks.ErrInvalidTaskConfig, key)
	}
	b.edgeSet[key] = true
	b.graph.outgoing[fromID] = append(b.graph.outgoing[fromID], toID)
	b.graph.incoming[toID] = append(b.graph.incoming[toID], fromID)
	b.graph.outdegree[fromID]++
	b.graph.indegree[toID]++
	return nil
}

func validateGraph(index NodeIndex, graph Graph) error {
	if err := validateDegrees(index, graph); err != nil {
		return err
	}
	if err := validateControlTargets(index, graph); err != nil {
		return err
	}
	if err := validateConnectivity(index, graph); err != nil {
		return err
	}
	return validateCycles(index, graph)
}
