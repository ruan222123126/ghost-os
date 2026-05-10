package orchestration

import (
	"strings"
	"time"

	"ghost-os/bridge/session"
	bridgeTasks "ghost-os/bridge/tasks"
)

func buildRelayAgentNodeResult(
	task ScheduledTask,
	execution bridgeTasks.ExecutionResult,
	result relayModeResult,
	startedAt time.Time,
	finishedAt time.Time,
) bridgeTasks.RunNodeResult {
	record := buildAgentMessageNodeResult(task, execution, startedAt, finishedAt)
	record.Output = relayAgentNodeOutput(execution, result)
	record.Preview = strings.TrimSpace(execution.ResponsePreview)
	record.Error = strings.TrimSpace(execution.Error)
	return record
}

func relayAgentNodeOutput(execution bridgeTasks.ExecutionResult, result relayModeResult) map[string]any {
	output := map[string]any{
		"session_id_output": strings.TrimSpace(execution.SessionIDOutput),
		"response_preview":  strings.TrimSpace(execution.ResponsePreview),
		"stopped_by":        strings.TrimSpace(result.StoppedBy),
		"relay_records":     relayRecordSnapshots(result.Records),
	}
	if strings.TrimSpace(execution.Error) != "" {
		output["error"] = strings.TrimSpace(execution.Error)
	}
	return output
}

func relayRecordSnapshots(records []session.RelayRecord) []map[string]any {
	if len(records) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		out = append(out, relayRecordSnapshot(record))
	}
	return out
}

func relayRecordSnapshot(record session.RelayRecord) map[string]any {
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
