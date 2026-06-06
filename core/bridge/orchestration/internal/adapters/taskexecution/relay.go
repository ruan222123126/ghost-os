package taskexecution

import (
	apprelay "ghost-os/bridge/orchestration/internal/app/agentturn/relay"
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
)

func RelayTaskResult(result apprelay.Result) apptasks.RelayTaskResult {
	records := make([]apptasks.RelayRecordSnapshot, 0, len(result.Records))
	for _, record := range result.Records {
		records = append(records, apptasks.RelayRecordSnapshot{
			Round:          record.Round,
			Did:            record.Did,
			Remaining:      record.Remaining,
			FailedAttempts: append([]string(nil), record.FailedAttempts...),
			NextStep:       record.NextStep,
			Completed:      record.Completed,
			TraceID:        record.TraceID,
		})
	}
	if len(records) == 0 {
		records = nil
	}
	return apptasks.RelayTaskResult{
		Message:   result.Message,
		StoppedBy: result.StoppedBy,
		Records:   records,
		SessionID: result.SessionID,
	}
}
