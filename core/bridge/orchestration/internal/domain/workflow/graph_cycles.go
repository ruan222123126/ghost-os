package workflow

import (
	"fmt"
	"strings"

	taskdefs "ghost-os/bridge/taskdefs"
)

type cycleState struct {
	color    map[string]int
	stack    []string
	position map[string]int
}

func validateCycles(index NodeIndex, graph Graph) error {
	state := cycleState{
		color:    make(map[string]int, len(index.nodes)),
		stack:    make([]string, 0, len(index.nodes)),
		position: make(map[string]int, len(index.nodes)),
	}
	for nodeID := range index.nodes {
		if state.color[nodeID] != 0 {
			continue
		}
		if err := walkCycle(nodeID, index, graph, &state); err != nil {
			return err
		}
	}
	return nil
}

func walkCycle(nodeID string, index NodeIndex, graph Graph, state *cycleState) error {
	state.enter(nodeID)
	for _, nextID := range graph.outgoing[nodeID] {
		if err := visitNextCycleNode(nextID, index, graph, state); err != nil {
			return err
		}
	}
	state.leave(nodeID)
	return nil
}

func visitNextCycleNode(nextID string, index NodeIndex, graph Graph, state *cycleState) error {
	if state.color[nextID] == 0 {
		if err := walkCycle(nextID, index, graph, state); err != nil {
			return err
		}
	}
	if state.color[nextID] != 1 {
		return nil
	}
	cycle := append([]string(nil), state.stack[state.position[nextID]:]...)
	if cycleContainsLoopNode(cycle, index) {
		return nil
	}
	return fmt.Errorf("%w: workflow cycle must include a loop node: %s", taskdefs.ErrInvalidTaskConfig, strings.Join(cycle, " -> "))
}

func (s *cycleState) enter(nodeID string) {
	s.color[nodeID] = 1
	s.position[nodeID] = len(s.stack)
	s.stack = append(s.stack, nodeID)
}

func (s *cycleState) leave(nodeID string) {
	s.color[nodeID] = 2
	delete(s.position, nodeID)
	s.stack = s.stack[:len(s.stack)-1]
}

func cycleContainsLoopNode(cycle []string, index NodeIndex) bool {
	for _, nodeID := range cycle {
		if index.nodes[nodeID].Type == NodeTypeLoop {
			return true
		}
	}
	return false
}
