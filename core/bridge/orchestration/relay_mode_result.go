package orchestration

import (
	"context"
	"errors"

	"ghost-os/bridge/session"
)

func (r relayModeRunner) finishErroredRun(sess *session.Session, runErr error) error {
	status := relayModeStatusError
	stoppedBy := relayModeStopError
	if errors.Is(runErr, context.Canceled) {
		status = relayModeStatusCancelled
		stoppedBy = relayModeStopCancelled
	}
	sess.FinishRelayRuntime(status, stoppedBy, "", "")
	return r.sessionStore.Save(sess)
}

func relayResultStatus(stoppedBy string) string {
	switch stoppedBy {
	case relayModeStopCompleted:
		return relayModeStatusCompleted
	case relayModeStopMaxRounds:
		return relayModeStatusIncomplete
	case relayModeStopCancelled:
		return relayModeStatusCancelled
	default:
		return relayModeStatusError
	}
}

func cloneRelayRecords(runtime *session.RelayRuntime) []session.RelayRecord {
	if runtime == nil || len(runtime.Records) == 0 {
		return nil
	}
	out := make([]session.RelayRecord, len(runtime.Records))
	copy(out, runtime.Records)
	return out
}
