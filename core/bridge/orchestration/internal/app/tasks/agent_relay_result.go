package tasks

import (
	"context"
	"errors"
	"strings"

	sharedtext "ghost-os/bridge/orchestration/internal/shared/text"
	"ghost-os/bridge/taskdefs"
)

const RelayStopMaxRounds = "max_rounds"

type RelayTaskRunner interface {
	ExecuteRelayTask(
		ctx context.Context,
		task taskdefs.ScheduledTask,
		traceID string,
	) (RelayTaskResult, error)
}

type RelayTaskResult struct {
	Message   string
	StoppedBy string
	Records   []RelayRecordSnapshot
	SessionID string
}

type RelayRecordSnapshot struct {
	Round          int
	Did            string
	Remaining      string
	FailedAttempts []string
	NextStep       string
	Completed      bool
	TraceID        string
}

type RelayPreflightError struct {
	Err error
}

func (e RelayPreflightError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e RelayPreflightError) Unwrap() error {
	return e.Err
}

func NewRelayPreflightError(err error) error {
	if err == nil {
		return nil
	}
	return RelayPreflightError{Err: err}
}

func IsRelayPreflightError(err error) bool {
	var preflight RelayPreflightError
	return errors.As(err, &preflight)
}

func ExecutionResultFromRelayOutcome(result RelayTaskResult, err error) taskdefs.ExecutionResult {
	if err != nil {
		return taskdefs.ExecutionResult{Status: taskdefs.RunStatusError, Error: err.Error()}
	}
	status := taskdefs.RunStatusSuccess
	if result.StoppedBy == RelayStopMaxRounds {
		status = taskdefs.RunStatusIncomplete
	}
	return taskdefs.ExecutionResult{
		Status:          status,
		SessionIDOutput: strings.TrimSpace(result.SessionID),
		ResponsePreview: sharedtext.TruncateRunes(strings.TrimSpace(result.Message), taskdefs.MaxResponsePreviewRunes),
	}
}

func BuildRelayAgentMessageNodeResult(
	task taskdefs.ScheduledTask,
	execution taskdefs.ExecutionResult,
	result RelayTaskResult,
	timestamps NodeResultTimestamps,
) taskdefs.RunNodeResult {
	record := BuildAgentMessageNodeResult(task, execution, timestamps)
	record.Output = RelayAgentMessageNodeOutput(execution, result)
	record.Preview = strings.TrimSpace(execution.ResponsePreview)
	record.Error = strings.TrimSpace(execution.Error)
	return record
}

func RelayAgentMessageNodeOutput(
	execution taskdefs.ExecutionResult,
	result RelayTaskResult,
) map[string]any {
	output := map[string]any{
		"session_id_output": strings.TrimSpace(execution.SessionIDOutput),
		"response_preview":  strings.TrimSpace(execution.ResponsePreview),
		"stopped_by":        strings.TrimSpace(result.StoppedBy),
		"relay_records":     RelayRecordSnapshots(result.Records),
	}
	if strings.TrimSpace(execution.Error) != "" {
		output["error"] = strings.TrimSpace(execution.Error)
	}
	return output
}

func RelayRecordSnapshots(records []RelayRecordSnapshot) []map[string]any {
	if len(records) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, RelayRecordSnapshotPayload(record))
	}
	return out
}

func RelayRecordSnapshotPayload(record RelayRecordSnapshot) map[string]any {
	return map[string]any{
		"round":           record.Round,
		"did":             strings.TrimSpace(record.Did),
		"remaining":       strings.TrimSpace(record.Remaining),
		"failed_attempts": append([]string(nil), record.FailedAttempts...),
		"next_step":       strings.TrimSpace(record.NextStep),
		"completed":       record.Completed,
		"trace_id":        strings.TrimSpace(record.TraceID),
	}
}
