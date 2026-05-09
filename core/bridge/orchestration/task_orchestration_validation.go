package orchestration

import (
	"fmt"
	"strings"
)

type orchestrationExecutionPlan struct {
	nodes        map[string]OrchestrationNode
	entryGroupID string
	controlNext  map[string]string
	groupMember  map[string][]string
}

type orchestrationGraphData struct {
	nodes         map[string]OrchestrationNode
	rawControlIn  map[string]int
	rawControlOut map[string]int
	controlIn     map[string]int
	controlOut    map[string]int
	controlNext   map[string]string
	groupMembers  map[string][]string
	startCount    int
	endCount      int
	groupCount    int
	edgeCount     int
}

func validateOrchestrationTaskDefinition(task *ScheduledTask) error {
	if task.Orchestration == nil {
		return fmt.Errorf("%w: orchestration is required for orchestration task", ErrInvalidTaskConfig)
	}
	if task.Name == "" {
		return fmt.Errorf("%w: orchestration name is required", ErrInvalidTaskConfig)
	}
	if task.Message != "" || task.SessionID != "" || task.Action != "" {
		return fmt.Errorf("%w: orchestration task does not allow message, session_id, or action", ErrInvalidTaskConfig)
	}
	if task.RuntimeOverrides != nil || task.Workflow != nil || len(task.ActionParams) > 0 {
		return fmt.Errorf("%w: orchestration task only allows name, orchestration, and schedule fields", ErrInvalidTaskConfig)
	}
	if err := normalizeOrchestrationDefinitionRuntimeOverrides(task.Orchestration); err != nil {
		return err
	}
	_, err := buildOrchestrationExecutionPlan(task.Orchestration)
	return err
}

func normalizeOrchestrationDefinitionRuntimeOverrides(definition *OrchestrationDefinition) error {
	if definition == nil {
		return nil
	}
	for index := range definition.Nodes {
		node := &definition.Nodes[index]
		if node.Type != orchestrationNodeTypeAgent || node.Agent == nil {
			continue
		}
		overrides, err := normalizeOrchestrationAgentRuntimeOverrides(node.Agent.RuntimeOverrides)
		if err != nil {
			return fmt.Errorf("%w: orchestration agent node %q %v", ErrInvalidTaskConfig, node.ID, err)
		}
		node.Agent.RuntimeOverrides = overrides
	}
	return nil
}

func buildOrchestrationExecutionPlan(definition *OrchestrationDefinition) (orchestrationExecutionPlan, error) {
	graph, err := buildOrchestrationGraphData(definition)
	if err != nil {
		return orchestrationExecutionPlan{}, err
	}
	if err := validateOrchestrationControlDegrees(graph); err != nil {
		return orchestrationExecutionPlan{}, err
	}
	if err := validateOrchestrationConnectivity(graph); err != nil {
		return orchestrationExecutionPlan{}, err
	}
	if err := validateOrchestrationMembers(graph); err != nil {
		return orchestrationExecutionPlan{}, err
	}
	return orchestrationExecutionPlan{
		nodes:        graph.nodes,
		entryGroupID: findOrchestrationEntryGroupID(graph),
		controlNext:  graph.controlNext,
		groupMember:  graph.groupMembers,
	}, nil
}

func buildOrchestrationGraphData(definition *OrchestrationDefinition) (orchestrationGraphData, error) {
	if definition == nil {
		return orchestrationGraphData{}, fmt.Errorf("%w: orchestration is required", ErrInvalidTaskConfig)
	}
	graph := orchestrationGraphData{
		nodes:         make(map[string]OrchestrationNode, len(definition.Nodes)),
		rawControlIn:  make(map[string]int, len(definition.Nodes)),
		rawControlOut: make(map[string]int, len(definition.Nodes)),
		controlIn:     make(map[string]int, len(definition.Nodes)),
		controlOut:    make(map[string]int, len(definition.Nodes)),
		controlNext:   make(map[string]string, len(definition.Nodes)),
		groupMembers:  make(map[string][]string),
	}
	for _, node := range definition.Nodes {
		if err := validateOrchestrationNode(node); err != nil {
			return orchestrationGraphData{}, err
		}
		if _, exists := graph.nodes[node.ID]; exists {
			return orchestrationGraphData{}, fmt.Errorf("%w: duplicate orchestration node id %q", ErrInvalidTaskConfig, node.ID)
		}
		graph.nodes[node.ID] = node
		switch node.Type {
		case orchestrationNodeTypeStart:
			graph.startCount++
		case orchestrationNodeTypeEnd:
			graph.endCount++
		case orchestrationNodeTypeGroup:
			graph.groupCount++
		}
	}
	if graph.startCount > 1 {
		return orchestrationGraphData{}, fmt.Errorf("%w: orchestration requires exactly 1 start node", ErrInvalidTaskConfig)
	}
	if graph.endCount > 1 {
		return orchestrationGraphData{}, fmt.Errorf("%w: orchestration requires exactly 1 end node", ErrInvalidTaskConfig)
	}
	if graph.startCount != graph.endCount {
		return orchestrationGraphData{}, fmt.Errorf("%w: orchestration start/end nodes must both be present or both be absent", ErrInvalidTaskConfig)
	}
	for _, edge := range definition.Edges {
		if err := addOrchestrationEdge(graph, edge); err != nil {
			return orchestrationGraphData{}, err
		}
	}
	return graph, nil
}

func validateOrchestrationNode(node OrchestrationNode) error {
	if node.ID == "" {
		return fmt.Errorf("%w: orchestration node id is required", ErrInvalidTaskConfig)
	}
	switch node.Type {
	case orchestrationNodeTypeStart:
		if node.Group != nil || node.Agent != nil {
			return fmt.Errorf("%w: orchestration node %q payload does not match type %q", ErrInvalidTaskConfig, node.ID, node.Type)
		}
	case orchestrationNodeTypeGroup:
		if node.Group == nil || node.Agent != nil || node.Group.Title == "" {
			return fmt.Errorf("%w: orchestration group node %q requires title", ErrInvalidTaskConfig, node.ID)
		}
		if node.Group.MaxRounds <= 0 {
			return fmt.Errorf("%w: orchestration group node %q requires max_rounds > 0", ErrInvalidTaskConfig, node.ID)
		}
		if node.Group.SpeakingMode != orchestrationModeSequential &&
			node.Group.SpeakingMode != orchestrationModeParallel &&
			node.Group.SpeakingMode != orchestrationModeOwner {
			return fmt.Errorf("%w: orchestration group node %q requires speaking_mode sequential|parallel|owner", ErrInvalidTaskConfig, node.ID)
		}
		if node.Group.SpeakingMode == orchestrationModeOwner && node.Group.OwnerAgentID == "" {
			return fmt.Errorf("%w: orchestration group node %q requires owner_agent_id in owner mode", ErrInvalidTaskConfig, node.ID)
		}
		if node.Group.SpeakingMode != orchestrationModeOwner && node.Group.OwnerAgentID != "" {
			node.Group.OwnerAgentID = ""
		}
	case orchestrationNodeTypeAgent:
		if node.Agent == nil || node.Group != nil || node.Agent.Title == "" || node.Agent.Message == "" {
			return fmt.Errorf("%w: orchestration agent node %q requires title and message", ErrInvalidTaskConfig, node.ID)
		}
	case orchestrationNodeTypeEnd:
		if node.Group != nil || node.Agent != nil {
			return fmt.Errorf("%w: orchestration node %q payload does not match type %q", ErrInvalidTaskConfig, node.ID, node.Type)
		}
	default:
		return fmt.Errorf("%w: unsupported orchestration node type %q", ErrInvalidTaskConfig, node.Type)
	}
	return nil
}

func addOrchestrationEdge(graph orchestrationGraphData, edge OrchestrationEdge) error {
	fromNode, ok := graph.nodes[edge.FromNodeID]
	if !ok {
		return fmt.Errorf("%w: orchestration edge references unknown from_node_id %q", ErrInvalidTaskConfig, edge.FromNodeID)
	}
	toNode, ok := graph.nodes[edge.ToNodeID]
	if !ok {
		return fmt.Errorf("%w: orchestration edge references unknown to_node_id %q", ErrInvalidTaskConfig, edge.ToNodeID)
	}
	if edge.FromNodeID == edge.ToNodeID {
		return fmt.Errorf("%w: orchestration does not allow self-loop edge %q", ErrInvalidTaskConfig, edge.FromNodeID)
	}
	graph.edgeCount++
	switch edge.Kind {
	case orchestrationEdgeKindControl:
		if !isValidOrchestrationControlEdge(fromNode.Type, toNode.Type) {
			return fmt.Errorf("%w: orchestration control edge %q -> %q is invalid", ErrInvalidTaskConfig, edge.FromNodeID, edge.ToNodeID)
		}
		graph.rawControlOut[edge.FromNodeID]++
		graph.rawControlIn[edge.ToNodeID]++
		if fromNode.Type == orchestrationNodeTypeGroup && toNode.Type == orchestrationNodeTypeGroup {
			if _, exists := graph.controlNext[edge.FromNodeID]; exists {
				return fmt.Errorf("%w: orchestration node %q allows only one control outgoing edge", ErrInvalidTaskConfig, edge.FromNodeID)
			}
			graph.controlNext[edge.FromNodeID] = edge.ToNodeID
			graph.controlOut[edge.FromNodeID]++
			graph.controlIn[edge.ToNodeID]++
		}
	case orchestrationEdgeKindMember:
		if fromNode.Type != orchestrationNodeTypeAgent || toNode.Type != orchestrationNodeTypeGroup {
			return fmt.Errorf("%w: orchestration member edge must be agent -> group", ErrInvalidTaskConfig)
		}
		graph.groupMembers[edge.ToNodeID] = append(graph.groupMembers[edge.ToNodeID], edge.FromNodeID)
	default:
		return fmt.Errorf("%w: unsupported orchestration edge kind %q", ErrInvalidTaskConfig, edge.Kind)
	}
	return nil
}

func validateOrchestrationControlDegrees(graph orchestrationGraphData) error {
	if graph.groupCount == 0 {
		if len(graph.nodes) == 0 && graph.edgeCount == 0 {
			return nil
		}
		return fmt.Errorf("%w: orchestration requires at least 1 group node", ErrInvalidTaskConfig)
	}
	entryCount := 0
	exitCount := 0
	for _, node := range graph.nodes {
		inDegree := graph.controlIn[node.ID]
		outDegree := graph.controlOut[node.ID]
		rawInDegree := graph.rawControlIn[node.ID]
		rawOutDegree := graph.rawControlOut[node.ID]
		switch node.Type {
		case orchestrationNodeTypeStart:
			if rawInDegree != 0 || rawOutDegree != 1 {
				return fmt.Errorf("%w: orchestration start node must have in=0 and out=1", ErrInvalidTaskConfig)
			}
		case orchestrationNodeTypeEnd:
			if rawInDegree != 1 || rawOutDegree != 0 {
				return fmt.Errorf("%w: orchestration end node must have in=1 and out=0", ErrInvalidTaskConfig)
			}
		case orchestrationNodeTypeGroup:
			if inDegree > 1 || outDegree > 1 {
				return fmt.Errorf("%w: orchestration group node %q must have in<=1 and out<=1", ErrInvalidTaskConfig, node.ID)
			}
			if inDegree == 0 {
				entryCount++
			}
			if outDegree == 0 {
				exitCount++
			}
		case orchestrationNodeTypeAgent:
			if rawInDegree != 0 || rawOutDegree != 0 {
				return fmt.Errorf("%w: orchestration agent node %q cannot participate in control flow", ErrInvalidTaskConfig, node.ID)
			}
		}
	}
	if entryCount != 1 {
		return fmt.Errorf("%w: orchestration requires exactly 1 entry group", ErrInvalidTaskConfig)
	}
	if exitCount != 1 {
		return fmt.Errorf("%w: orchestration requires exactly 1 exit group", ErrInvalidTaskConfig)
	}
	return nil
}

func validateOrchestrationConnectivity(graph orchestrationGraphData) error {
	if graph.groupCount == 0 {
		return nil
	}
	currentID := findOrchestrationEntryGroupID(graph)
	if currentID == "" {
		return nil
	}
	controlSeen := make(map[string]bool, graph.groupCount)
	for currentID != "" {
		if controlSeen[currentID] {
			return fmt.Errorf("%w: orchestration control flow contains a cycle", ErrInvalidTaskConfig)
		}
		controlSeen[currentID] = true
		currentID = graph.controlNext[currentID]
	}
	for _, node := range graph.nodes {
		if node.Type != orchestrationNodeTypeGroup {
			continue
		}
		if !controlSeen[node.ID] {
			return fmt.Errorf("%w: orchestration node %q is disconnected from control flow", ErrInvalidTaskConfig, node.ID)
		}
	}
	return nil
}

func validateOrchestrationMembers(graph orchestrationGraphData) error {
	for _, node := range graph.nodes {
		if node.Type != orchestrationNodeTypeGroup {
			continue
		}
		members := graph.groupMembers[node.ID]
		if len(members) == 0 {
			return fmt.Errorf("%w: orchestration group node %q requires at least one member", ErrInvalidTaskConfig, node.ID)
		}
		if node.Group.SpeakingMode != orchestrationModeOwner {
			continue
		}
		ownerID := strings.TrimSpace(node.Group.OwnerAgentID)
		if ownerID == "" {
			return fmt.Errorf("%w: orchestration group node %q requires owner_agent_id in owner mode", ErrInvalidTaskConfig, node.ID)
		}
		if !containsOrchestrationMemberID(members, ownerID) {
			return fmt.Errorf("%w: orchestration group node %q owner_agent_id %q must be an existing member", ErrInvalidTaskConfig, node.ID, ownerID)
		}
	}
	return nil
}

func containsOrchestrationMemberID(members []string, target string) bool {
	for _, memberID := range members {
		if strings.TrimSpace(memberID) == target {
			return true
		}
	}
	return false
}

func isValidOrchestrationControlEdge(fromType string, toType string) bool {
	switch fromType {
	case orchestrationNodeTypeStart:
		return toType == orchestrationNodeTypeGroup || toType == orchestrationNodeTypeEnd
	case orchestrationNodeTypeGroup:
		return toType == orchestrationNodeTypeGroup || toType == orchestrationNodeTypeEnd
	default:
		return false
	}
}

func findOrchestrationEntryGroupID(graph orchestrationGraphData) string {
	for _, node := range graph.nodes {
		if node.Type == orchestrationNodeTypeGroup && graph.controlIn[node.ID] == 0 {
			return node.ID
		}
	}
	return ""
}
