package workflow

import (
	"fmt"
	"strings"

	taskdefs "ghost-os/bridge/taskdefs"
)

func validateIfNode(node Node) error {
	if !IsIfOperatorSupported(node.If.Operator) {
		return fmt.Errorf("%w: workflow if node %q uses unsupported operator %q", taskdefs.ErrInvalidTaskConfig, node.ID, node.If.Operator)
	}
	if strings.TrimSpace(node.If.TrueNodeID) == "" || strings.TrimSpace(node.If.FalseNodeID) == "" {
		return fmt.Errorf("%w: workflow if node %q requires true_node_id and false_node_id", taskdefs.ErrInvalidTaskConfig, node.ID)
	}
	if node.If.TrueNodeID == node.If.FalseNodeID {
		return fmt.Errorf("%w: workflow if node %q true_node_id and false_node_id must differ", taskdefs.ErrInvalidTaskConfig, node.ID)
	}
	if RequiresIfValue(node.If.Operator) && strings.TrimSpace(node.If.Value) == "" {
		return fmt.Errorf("%w: workflow if node %q requires value for operator %q", taskdefs.ErrInvalidTaskConfig, node.ID, node.If.Operator)
	}
	return nil
}

func validateLoopNode(node Node) error {
	if node.Loop.MaxIterations <= 0 {
		return fmt.Errorf("%w: workflow loop node %q requires max_iterations > 0", taskdefs.ErrInvalidTaskConfig, node.ID)
	}
	if strings.TrimSpace(node.Loop.BodyNodeID) == "" || strings.TrimSpace(node.Loop.ExitNodeID) == "" {
		return fmt.Errorf("%w: workflow loop node %q requires body_node_id and exit_node_id", taskdefs.ErrInvalidTaskConfig, node.ID)
	}
	if node.Loop.BodyNodeID == node.Loop.ExitNodeID {
		return fmt.Errorf("%w: workflow loop node %q body_node_id and exit_node_id must differ", taskdefs.ErrInvalidTaskConfig, node.ID)
	}
	return nil
}

func IsIfOperatorSupported(operator string) bool {
	switch strings.TrimSpace(operator) {
	case IfOperatorEquals,
		IfOperatorNotEquals,
		IfOperatorContains,
		IfOperatorNotContains,
		IfOperatorIsEmpty,
		IfOperatorNotEmpty:
		return true
	default:
		return false
	}
}

func RequiresIfValue(operator string) bool {
	switch strings.TrimSpace(operator) {
	case IfOperatorIsEmpty, IfOperatorNotEmpty:
		return false
	default:
		return true
	}
}

func countNodes(index NodeIndex, want string) int {
	count := 0
	for _, node := range index.nodes {
		if node.Type == want {
			count++
		}
	}
	return count
}
