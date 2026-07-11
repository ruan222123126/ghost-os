package session

import (
	"strings"
	"time"
)

type RelayRecord struct {
	Round          int       `json:"round"`
	Did            string    `json:"did"`
	Remaining      string    `json:"remaining"`
	FailedAttempts []string  `json:"failed_attempts,omitempty"`
	NextStep       string    `json:"next_step"`
	Completed      bool      `json:"completed,omitempty"`
	TraceID        string    `json:"trace_id,omitempty"`
	RecordedAt     time.Time `json:"recorded_at,omitempty"`
	FinalChangeLog string    `json:"final_change_log,omitempty"`
}

type RelayRuntime struct {
	OriginalTask       string        `json:"original_task,omitempty"`
	StopPolicy         string        `json:"stop_policy,omitempty"`
	MaxRounds          int           `json:"max_rounds,omitempty"`
	ExecutionTimeoutMS *int          `json:"execution_timeout_ms,omitempty"`
	RoundCount         int           `json:"round_count,omitempty"`
	Status             string        `json:"status,omitempty"`
	StoppedBy          string        `json:"stopped_by,omitempty"`
	FinalMessage       string        `json:"final_message,omitempty"`
	FinalChangeLog     string        `json:"final_change_log,omitempty"`
	StartedAt          time.Time     `json:"started_at,omitempty"`
	UpdatedAt          time.Time     `json:"updated_at,omitempty"`
	Records            []RelayRecord `json:"records,omitempty"`
}

func (s *Session) StartRelayRuntime(task string, stopPolicy string, maxRounds int, executionTimeoutMS *int) {
	if s == nil {
		return
	}
	now := time.Now().UTC()
	s.RelayRuntime = &RelayRuntime{
		OriginalTask:       strings.TrimSpace(task),
		StopPolicy:         strings.TrimSpace(stopPolicy),
		MaxRounds:          maxRounds,
		ExecutionTimeoutMS: cloneOptionalInt(executionTimeoutMS),
		Status:             "running",
		StartedAt:          now,
		UpdatedAt:          now,
	}
	s.UpdatedAt = now
}

func (s *Session) AppendRelayRecord(record RelayRecord) {
	if s == nil {
		return
	}
	if s.RelayRuntime == nil {
		s.RelayRuntime = &RelayRuntime{}
	}
	record = normalizeRelayRecord(record)
	s.RelayRuntime.Records = append(s.RelayRuntime.Records, record)
	s.RelayRuntime.RoundCount = len(s.RelayRuntime.Records)
	s.RelayRuntime.UpdatedAt = record.RecordedAt
	s.RelayRuntime.Status = "running"
	s.UpdatedAt = record.RecordedAt
}

func (s *Session) FinishRelayRuntime(status string, stoppedBy string, finalMessage string, finalChangeLog string) {
	if s == nil || s.RelayRuntime == nil {
		return
	}
	now := time.Now().UTC()
	s.RelayRuntime.Status = strings.TrimSpace(status)
	s.RelayRuntime.StoppedBy = strings.TrimSpace(stoppedBy)
	s.RelayRuntime.FinalMessage = strings.TrimSpace(finalMessage)
	s.RelayRuntime.FinalChangeLog = strings.TrimSpace(finalChangeLog)
	s.RelayRuntime.RoundCount = len(s.RelayRuntime.Records)
	s.RelayRuntime.UpdatedAt = now
	s.UpdatedAt = now
}

func normalizeRelayRecord(record RelayRecord) RelayRecord {
	record.Did = strings.TrimSpace(record.Did)
	record.Remaining = strings.TrimSpace(record.Remaining)
	record.FailedAttempts = normalizeRelayRecordAttempts(record.FailedAttempts)
	record.NextStep = strings.TrimSpace(record.NextStep)
	record.TraceID = strings.TrimSpace(record.TraceID)
	record.FinalChangeLog = strings.TrimSpace(record.FinalChangeLog)
	if record.RecordedAt.IsZero() {
		record.RecordedAt = time.Now().UTC()
	}
	return record
}

func normalizeRelayRecordAttempts(input []string) []string {
	out := make([]string, 0, len(input))
	for _, raw := range input {
		value := strings.TrimSpace(raw)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func cloneOptionalInt(input *int) *int {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func cloneRelayRuntime(raw *RelayRuntime) *RelayRuntime {
	if raw == nil {
		return nil
	}
	cloned := *raw
	cloned.ExecutionTimeoutMS = cloneOptionalInt(raw.ExecutionTimeoutMS)
	if len(raw.Records) > 0 {
		cloned.Records = make([]RelayRecord, 0, len(raw.Records))
		for _, record := range raw.Records {
			cloned.Records = append(cloned.Records, cloneRelayRecord(record))
		}
	}
	return &cloned
}

func cloneRelayRecord(record RelayRecord) RelayRecord {
	record.FailedAttempts = append([]string(nil), record.FailedAttempts...)
	return record
}
