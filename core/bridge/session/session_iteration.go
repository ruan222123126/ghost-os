package session

import (
	"strings"
	"time"
)

// IterationRecord captures the minimal handoff state between fresh-memory pro/prox agents.
type IterationRecord struct {
	Iteration      int       `json:"iteration"`
	Did            string    `json:"did"`
	Remaining      string    `json:"remaining"`
	Completed      bool      `json:"completed,omitempty"`
	TraceID        string    `json:"trace_id,omitempty"`
	RecordedAt     time.Time `json:"recorded_at,omitempty"`
	FinalChangeLog string    `json:"final_change_log,omitempty"`
}

// IterationRuntime stores the latest pro/prox orchestration state without polluting chat history.
type IterationRuntime struct {
	Mode           string            `json:"mode,omitempty"`
	OriginalTask   string            `json:"original_task,omitempty"`
	MaxIterations  int               `json:"max_iterations,omitempty"`
	Unlimited      bool              `json:"unlimited,omitempty"`
	IterationCount int               `json:"iteration_count,omitempty"`
	Status         string            `json:"status,omitempty"`
	StoppedBy      string            `json:"stopped_by,omitempty"`
	FinalMessage   string            `json:"final_message,omitempty"`
	FinalChangeLog string            `json:"final_change_log,omitempty"`
	StartedAt      time.Time         `json:"started_at,omitempty"`
	UpdatedAt      time.Time         `json:"updated_at,omitempty"`
	Records        []IterationRecord `json:"records,omitempty"`
}

// StartIterationRuntime resets the hidden pro/prox iteration state for a new run.
func (s *Session) StartIterationRuntime(mode string, originalTask string, maxIterations int, unlimited bool) {
	if s == nil {
		return
	}

	now := time.Now().UTC()
	s.IterationRuntime = &IterationRuntime{
		Mode:          strings.TrimSpace(mode),
		OriginalTask:  strings.TrimSpace(originalTask),
		MaxIterations: maxIterations,
		Unlimited:     unlimited,
		Status:        "running",
		StartedAt:     now,
		UpdatedAt:     now,
	}
	s.UpdatedAt = now
}

// AppendIterationRecord records one fresh-memory handoff step.
func (s *Session) AppendIterationRecord(record IterationRecord) {
	if s == nil {
		return
	}
	if s.IterationRuntime == nil {
		s.IterationRuntime = &IterationRuntime{}
	}

	record = normalizeIterationRecord(record)
	s.IterationRuntime.Records = append(s.IterationRuntime.Records, record)
	s.IterationRuntime.IterationCount = len(s.IterationRuntime.Records)
	s.IterationRuntime.UpdatedAt = record.RecordedAt
	s.IterationRuntime.Status = "running"
	s.UpdatedAt = record.RecordedAt
}

// FinishIterationRuntime marks the latest pro/prox run as completed, cancelled, or errored.
func (s *Session) FinishIterationRuntime(status string, stoppedBy string, finalMessage string, finalChangeLog string) {
	if s == nil || s.IterationRuntime == nil {
		return
	}

	now := time.Now().UTC()
	s.IterationRuntime.Status = strings.TrimSpace(status)
	s.IterationRuntime.StoppedBy = strings.TrimSpace(stoppedBy)
	s.IterationRuntime.FinalMessage = strings.TrimSpace(finalMessage)
	s.IterationRuntime.FinalChangeLog = strings.TrimSpace(finalChangeLog)
	s.IterationRuntime.IterationCount = len(s.IterationRuntime.Records)
	s.IterationRuntime.UpdatedAt = now
	s.UpdatedAt = now
}

func normalizeIterationRecord(record IterationRecord) IterationRecord {
	record.Did = strings.TrimSpace(record.Did)
	record.Remaining = strings.TrimSpace(record.Remaining)
	record.TraceID = strings.TrimSpace(record.TraceID)
	record.FinalChangeLog = strings.TrimSpace(record.FinalChangeLog)
	if record.RecordedAt.IsZero() {
		record.RecordedAt = time.Now().UTC()
	}
	return record
}
