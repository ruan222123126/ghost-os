package group

import (
	"fmt"
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
)

func validateNode(node Node) error {
	if node.ID == "" {
		return fmt.Errorf("%w: orchestration node id is required", bridgeTasks.ErrInvalidTaskConfig)
	}
	switch node.Type {
	case NodeTypeStart:
		return legacyBoundaryNodeError(node.Type)
	case NodeTypeGroup:
		return validateGroupNode(node)
	case NodeTypeAgent:
		return validateAgentNode(node)
	case NodeTypeEnd:
		return legacyBoundaryNodeError(node.Type)
	default:
		return fmt.Errorf("%w: unsupported orchestration node type %q", bridgeTasks.ErrInvalidTaskConfig, node.Type)
	}
}

func legacyBoundaryNodeError(nodeType string) error {
	return fmt.Errorf(
		"%w: orchestration node type %q is removed; run `bin/ghost-bridge migrate orchestrations`",
		bridgeTasks.ErrInvalidTaskConfig,
		nodeType,
	)
}

func validateGroupNode(node Node) error {
	if node.Group == nil || node.Agent != nil || node.Group.Title == "" {
		return fmt.Errorf("%w: orchestration group node %q requires title", bridgeTasks.ErrInvalidTaskConfig, node.ID)
	}
	if node.Group.MaxRounds <= 0 {
		return fmt.Errorf("%w: orchestration group node %q requires max_rounds > 0", bridgeTasks.ErrInvalidTaskConfig, node.ID)
	}
	return validateGroupSpeakingMode(node)
}

func validateGroupSpeakingMode(node Node) error {
	if !isSupportedSpeakingMode(node.Group.SpeakingMode) {
		return fmt.Errorf("%w: orchestration group node %q requires speaking_mode sequential|parallel|owner", bridgeTasks.ErrInvalidTaskConfig, node.ID)
	}
	if node.Group.SpeakingMode == SpeakingModeOwner && node.Group.OwnerAgentID == "" {
		return fmt.Errorf("%w: orchestration group node %q requires owner_agent_id in owner mode", bridgeTasks.ErrInvalidTaskConfig, node.ID)
	}
	if node.Group.SpeakingMode != SpeakingModeOwner && node.Group.OwnerAgentID != "" {
		node.Group.OwnerAgentID = ""
	}
	return nil
}

func isSupportedSpeakingMode(mode string) bool {
	return mode == SpeakingModeSequential ||
		mode == SpeakingModeParallel ||
		mode == SpeakingModeOwner
}

func validateAgentNode(node Node) error {
	if node.Agent == nil || node.Group != nil || node.Agent.Title == "" || node.Agent.Message == "" {
		return fmt.Errorf("%w: orchestration agent node %q requires title and message", bridgeTasks.ErrInvalidTaskConfig, node.ID)
	}
	return nil
}

func validateControlDegrees(graph graphData) error {
	if graph.groupCount == 0 {
		return validateMissingGroups(graph)
	}
	entries, exits, err := countGroupEndpoints(graph)
	if err != nil {
		return err
	}
	if entries != 1 {
		return fmt.Errorf("%w: orchestration requires exactly 1 entry group", bridgeTasks.ErrInvalidTaskConfig)
	}
	if exits != 1 {
		return fmt.Errorf("%w: orchestration requires exactly 1 exit group", bridgeTasks.ErrInvalidTaskConfig)
	}
	return nil
}

func validateMissingGroups(graph graphData) error {
	if len(graph.nodes) == 0 && graph.edgeCount == 0 {
		return nil
	}
	return fmt.Errorf("%w: orchestration requires at least 1 group node", bridgeTasks.ErrInvalidTaskConfig)
}

func countGroupEndpoints(graph graphData) (int, int, error) {
	entryCount := 0
	exitCount := 0
	for _, node := range graph.nodes {
		inDegree := graph.controlIn[node.ID]
		outDegree := graph.controlOut[node.ID]
		if err := validateNodeDegrees(graph, node, inDegree, outDegree); err != nil {
			return 0, 0, err
		}
		if node.Type == NodeTypeGroup && inDegree == 0 {
			entryCount++
		}
		if node.Type == NodeTypeGroup && outDegree == 0 {
			exitCount++
		}
	}
	return entryCount, exitCount, nil
}

func validateNodeDegrees(graph graphData, node Node, inDegree int, outDegree int) error {
	rawInDegree := graph.rawControlIn[node.ID]
	rawOutDegree := graph.rawControlOut[node.ID]
	switch node.Type {
	case NodeTypeGroup:
		return validateGroupDegrees(node.ID, inDegree, outDegree)
	case NodeTypeAgent:
		return validateAgentDegrees(node.ID, rawInDegree, rawOutDegree)
	default:
		return nil
	}
}

func validateGroupDegrees(nodeID string, inDegree int, outDegree int) error {
	if inDegree > 1 || outDegree > 1 {
		return fmt.Errorf("%w: orchestration group node %q must have in<=1 and out<=1", bridgeTasks.ErrInvalidTaskConfig, nodeID)
	}
	return nil
}

func validateAgentDegrees(nodeID string, rawInDegree int, rawOutDegree int) error {
	if rawInDegree != 0 || rawOutDegree != 0 {
		return fmt.Errorf("%w: orchestration agent node %q cannot participate in control flow", bridgeTasks.ErrInvalidTaskConfig, nodeID)
	}
	return nil
}

func validateConnectivity(graph graphData) error {
	if graph.groupCount == 0 {
		return nil
	}
	controlSeen, err := collectReachableGroups(graph)
	if err != nil {
		return err
	}
	return validateAllGroupsReachable(graph, controlSeen)
}

func collectReachableGroups(graph graphData) (map[string]bool, error) {
	currentID := findEntryGroupID(graph)
	controlSeen := make(map[string]bool, graph.groupCount)
	for currentID != "" {
		if controlSeen[currentID] {
			return nil, fmt.Errorf("%w: orchestration control flow contains a cycle", bridgeTasks.ErrInvalidTaskConfig)
		}
		controlSeen[currentID] = true
		currentID = graph.controlNext[currentID]
	}
	return controlSeen, nil
}

func validateAllGroupsReachable(graph graphData, controlSeen map[string]bool) error {
	for _, node := range graph.nodes {
		if node.Type != NodeTypeGroup {
			continue
		}
		if !controlSeen[node.ID] {
			return fmt.Errorf("%w: orchestration node %q is disconnected from control flow", bridgeTasks.ErrInvalidTaskConfig, node.ID)
		}
	}
	return nil
}

func validateMembers(graph graphData) error {
	for _, node := range graph.nodes {
		if node.Type != NodeTypeGroup {
			continue
		}
		if err := validateGroupMembers(graph.groupMembers[node.ID], node); err != nil {
			return err
		}
	}
	return nil
}

func validateGroupMembers(members []string, node Node) error {
	if len(members) == 0 {
		return fmt.Errorf("%w: orchestration group node %q requires at least one member", bridgeTasks.ErrInvalidTaskConfig, node.ID)
	}
	if node.Group.SpeakingMode != SpeakingModeOwner {
		return nil
	}
	return validateOwnerMember(members, node)
}

func validateOwnerMember(members []string, node Node) error {
	ownerID := strings.TrimSpace(node.Group.OwnerAgentID)
	if ownerID == "" {
		return fmt.Errorf("%w: orchestration group node %q requires owner_agent_id in owner mode", bridgeTasks.ErrInvalidTaskConfig, node.ID)
	}
	if !containsMemberID(members, ownerID) {
		return fmt.Errorf("%w: orchestration group node %q owner_agent_id %q must be an existing member", bridgeTasks.ErrInvalidTaskConfig, node.ID, ownerID)
	}
	return nil
}

func containsMemberID(members []string, target string) bool {
	for _, memberID := range members {
		if strings.TrimSpace(memberID) == target {
			return true
		}
	}
	return false
}

func findEntryGroupID(graph graphData) string {
	for _, node := range graph.nodes {
		if node.Type == NodeTypeGroup && graph.controlIn[node.ID] == 0 {
			return node.ID
		}
	}
	return ""
}
