package result

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	bridgeTasks "ghost-os/bridge/tasks"
)

type NodeRecord struct {
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

type NodeResultRecorder struct {
	mu           sync.Mutex
	completedSeq int
	results      []bridgeTasks.RunNodeResult
}

func NewNodeResultRecorder(capacity int) *NodeResultRecorder {
	initialCapacity := capacity
	if initialCapacity < 0 {
		initialCapacity = 0
	}
	return &NodeResultRecorder{
		results: make([]bridgeTasks.RunNodeResult, 0, initialCapacity),
	}
}

func (r *NodeResultRecorder) Record(record NodeRecord) {
	if r == nil {
		return
	}
	startedAt := record.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	r.appendResult(record, startedAt, time.Now().UTC())
}

func (r *NodeResultRecorder) appendResult(
	record NodeRecord,
	startedAt time.Time,
	finishedAt time.Time,
) {
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
		Input:        JSONSnapshot(record.Input),
		Output:       JSONSnapshot(record.Output),
		Preview:      strings.TrimSpace(record.Preview),
		Error:        strings.TrimSpace(record.Error),
	})
}

func (r *NodeResultRecorder) Snapshot() []bridgeTasks.RunNodeResult {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return bridgeTasks.CloneRunNodeResults(r.results)
}

func JSONSnapshot(value any) any {
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
