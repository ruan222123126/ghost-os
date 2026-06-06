package tasks

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

const (
	DefaultRunLogRetention  = 100
	RunStatusRunning        = "running"
	RunStatusSuccess        = "success"
	RunStatusIncomplete     = "incomplete"
	RunStatusCancelled      = "cancelled"
	RunStatusError          = "error"
	RunStatusSkipped        = "skipped"
	RunStatusAwaitingHuman  = "awaiting_human"
	MaxResponsePreviewRunes = 240
)

var runIDCounter uint64

type RunNodeResult struct {
	NodeID       string    `json:"node_id"`
	NodeType     string    `json:"node_type"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	FinishedAt   time.Time `json:"finished_at,omitempty"`
	CompletedSeq int       `json:"completed_seq,omitempty"`
	BranchID     string    `json:"branch_id,omitempty"`
	Input        any       `json:"input,omitempty"`
	Output       any       `json:"output,omitempty"`
	Preview      string    `json:"preview,omitempty"`
	Error        string    `json:"error,omitempty"`
}

type RunLog struct {
	TaskID          string          `json:"task_id"`
	RunID           string          `json:"run_id"`
	TraceID         string          `json:"trace_id"`
	TaskKind        string          `json:"task_kind,omitempty"`
	Action          string          `json:"action,omitempty"`
	ScheduledAt     time.Time       `json:"scheduled_at"`
	StartedAt       time.Time       `json:"started_at,omitempty"`
	FinishedAt      time.Time       `json:"finished_at,omitempty"`
	Status          string          `json:"status"`
	SessionIDInput  string          `json:"session_id_input,omitempty"`
	SessionIDOutput string          `json:"session_id_output,omitempty"`
	ResponsePreview string          `json:"response_preview,omitempty"`
	NodeResults     []RunNodeResult `json:"node_results,omitempty"`
	RunCards        []RunCard       `json:"run_cards,omitempty"`
	Error           string          `json:"error,omitempty"`
}

func NormalizeRunLog(run *RunLog) {
	if run == nil {
		return
	}
	run.TaskID = strings.TrimSpace(run.TaskID)
	run.RunID = strings.TrimSpace(run.RunID)
	run.TraceID = strings.TrimSpace(run.TraceID)
	run.TaskKind = NormalizeKind(run.TaskKind)
	run.Action = strings.TrimSpace(run.Action)
	run.Status = strings.TrimSpace(run.Status)
	run.SessionIDInput = strings.TrimSpace(run.SessionIDInput)
	run.SessionIDOutput = strings.TrimSpace(run.SessionIDOutput)
	run.ResponsePreview = truncateRunes(strings.TrimSpace(run.ResponsePreview), MaxResponsePreviewRunes)
	run.NodeResults = CloneRunNodeResults(run.NodeResults)
	run.RunCards = CloneRunCards(run.RunCards)
	run.Error = strings.TrimSpace(run.Error)
	if run.Status == RunStatusCancelled {
		run.Error = ""
	}
	if !run.ScheduledAt.IsZero() {
		run.ScheduledAt = run.ScheduledAt.UTC()
	}
	if !run.StartedAt.IsZero() {
		run.StartedAt = run.StartedAt.UTC()
	}
	if !run.FinishedAt.IsZero() {
		run.FinishedAt = run.FinishedAt.UTC()
	}
}

func CloneRunNodeResults(input []RunNodeResult) []RunNodeResult {
	if len(input) == 0 {
		return nil
	}
	out := make([]RunNodeResult, len(input))
	for index, item := range input {
		out[index] = normalizeRunNodeResult(item)
	}
	return out
}

func normalizeRunNodeResult(input RunNodeResult) RunNodeResult {
	out := RunNodeResult{
		NodeID:       strings.TrimSpace(input.NodeID),
		NodeType:     strings.TrimSpace(input.NodeType),
		Status:       strings.TrimSpace(input.Status),
		StartedAt:    input.StartedAt,
		FinishedAt:   input.FinishedAt,
		CompletedSeq: input.CompletedSeq,
		BranchID:     strings.TrimSpace(input.BranchID),
		Input:        cloneJSONValue(input.Input),
		Output:       cloneJSONValue(input.Output),
		Preview:      strings.TrimSpace(input.Preview),
		Error:        strings.TrimSpace(input.Error),
	}
	if !out.StartedAt.IsZero() {
		out.StartedAt = out.StartedAt.UTC()
	}
	if !out.FinishedAt.IsZero() {
		out.FinishedAt = out.FinishedAt.UTC()
	}
	return out
}

func RunLogSortTime(run RunLog) time.Time {
	if !run.StartedAt.IsZero() {
		return run.StartedAt
	}
	return run.ScheduledAt
}

func NewRunID() string {
	return fmt.Sprintf("run-%d-%d", time.Now().UnixMilli(), atomic.AddUint64(&runIDCounter, 1))
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}
