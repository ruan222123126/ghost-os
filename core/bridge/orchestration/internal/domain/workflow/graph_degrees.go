package workflow

import (
	"fmt"

	taskdefs "ghost-os/bridge/taskdefs"
)

func validateDegrees(index NodeIndex, graph Graph) error {
	for nodeID, node := range index.nodes {
		in := graph.indegree[nodeID]
		out := graph.outdegree[nodeID]
		if err := validateNodeDegree(nodeID, node, in, out); err != nil {
			return err
		}
	}
	return nil
}

func validateNodeDegree(nodeID string, node Node, in int, out int) error {
	switch node.Type {
	case NodeTypeStart:
		if in != 0 || out < 1 {
			return fmt.Errorf("%w: start node must have in=0 and out>=1", taskdefs.ErrInvalidTaskConfig)
		}
	case NodeTypeEnd:
		if in < 1 || out != 0 {
			return fmt.Errorf("%w: end node must have in>=1 and out=0", taskdefs.ErrInvalidTaskConfig)
		}
	case NodeTypeIf, NodeTypeLoop:
		if in < 1 || out != 2 {
			return fmt.Errorf("%w: workflow node %q must have in>=1 and out=2", taskdefs.ErrInvalidTaskConfig, nodeID)
		}
	default:
		if in < 1 || out != 1 {
			return fmt.Errorf("%w: workflow node %q must have in>=1 and out=1", taskdefs.ErrInvalidTaskConfig, nodeID)
		}
	}
	return nil
}

func validateControlTargets(index NodeIndex, graph Graph) error {
	for nodeID, node := range index.nodes {
		if node.Type == NodeTypeIf {
			if err := validateIfTargets(nodeID, node, index, graph); err != nil {
				return err
			}
		}
		if node.Type == NodeTypeLoop {
			if err := validateLoopTargets(nodeID, node, index, graph); err != nil {
				return err
			}
		}
	}
	return nil
}
