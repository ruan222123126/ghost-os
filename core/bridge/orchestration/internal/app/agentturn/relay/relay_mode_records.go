package relay

import (
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/session"
)

func relayRecordFromHandoff(round int, traceID string, handoffErr *agent.ErrIterationHandoff) session.RelayRecord {
	return session.RelayRecord{
		Round:          round,
		Did:            handoffErr.Did,
		Remaining:      handoffErr.Remaining,
		FailedAttempts: append([]string(nil), handoffErr.FailedAttempts...),
		NextStep:       handoffErr.NextStep,
		Completed:      handoffErr.Completed,
		TraceID:        traceID,
		RecordedAt:     time.Now().UTC(),
		FinalChangeLog: handoffErr.FinalChangeLog,
	}
}
