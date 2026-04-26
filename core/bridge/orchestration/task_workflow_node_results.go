package orchestration

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	bridgeTasks "ghost-os/bridge/tasks"
)

type workflowNodeRecord struct {
	NodeID    string
	NodeType  string
	Status    string
	StartedAt time.Time
	BranchID  string
	Input     any
	Output    any
	Preview   string
	Error     string
}

type workflowNodeResultRecorder struct {
	mu           sync.Mutex
	completedSeq int
	results      []bridgeTasks.RunNodeResult
}

func newWorkflowNodeResultRecorder(capacity int) *workflowNodeResultRecorder {
	initialCapacity := capacity
	if initialCapacity < 0 {
		initialCapacity = 0
	}
	return &workflowNodeResultRecorder{
		results: make([]bridgeTasks.RunNodeResult, 0, initialCapacity),
	}
}

func (r *workflowNodeResultRecorder) record(record workflowNodeRecord) {
	if r == nil {
		return
	}
	startedAt := record.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	finishedAt := time.Now().UTC()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.completedSeq++
	r.results = append(r.results, bridgeTasks.RunNodeResult{
		NodeID:       strings.TrimSpace(record.NodeID),
		NodeType:     strings.TrimSpace(record.NodeType),
		Status:       strings.TrimSpace(record.Status),
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		CompletedSeq: r.completedSeq,
		BranchID:     strings.TrimSpace(record.BranchID),
		Input:        workflowJSONSnapshot(record.Input),
		Output:       workflowJSONSnapshot(record.Output),
		Preview:      strings.TrimSpace(record.Preview),
		Error:        strings.TrimSpace(record.Error),
	})
}

func (r *workflowNodeResultRecorder) snapshot() []bridgeTasks.RunNodeResult {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return bridgeTasks.CloneRunNodeResults(r.results)
}

func workflowNodeInputSnapshot(node WorkflowNode) any {
	return workflowJSONSnapshot(node)
}

func workflowNodeInputFromOutcome(node WorkflowNode, outcome workflowNodeOutcome) any {
	if outcome.inputSnapshot != nil {
		return workflowJSONSnapshot(outcome.inputSnapshot)
	}
	return workflowNodeInputSnapshot(node)
}

func workflowNodeOutputSnapshot(nextNodeID string, outcome workflowNodeOutcome) any {
	output := map[string]any{}
	if strings.TrimSpace(nextNodeID) != "" {
		output["next_node_id"] = strings.TrimSpace(nextNodeID)
	}
	if strings.TrimSpace(outcome.sessionID) != "" {
		output["session_id_output"] = strings.TrimSpace(outcome.sessionID)
	}
	if strings.TrimSpace(outcome.preview) != "" {
		output["response_preview"] = strings.TrimSpace(outcome.preview)
	}
	if outcome.outputValue != nil {
		output["output"] = workflowJSONSnapshot(outcome.outputValue)
	} else if strings.TrimSpace(outcome.outputText) != "" {
		output["output"] = strings.TrimSpace(outcome.outputText)
	}
	if outcome.err != nil {
		output["error"] = outcome.err.Error()
	}
	if len(output) == 0 {
		return nil
	}
	return output
}

func workflowEndNodeOutputSnapshot() any {
	return map[string]any{"reached_end": true}
}

func workflowJSONSnapshot(value any) any {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return value
	}
	return decoded
}
