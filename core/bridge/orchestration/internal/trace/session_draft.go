package trace

import (
	"time"

	"ghost-os/bridge/orchestration/internal/trace/sessiondraft"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

func ProjectTurnDraft(sess *session.Session, event streaming.Event, at time.Time) bool {
	return sessiondraft.ProjectTurnDraft(sess, event, at)
}

func ProjectSessionRunState(sess *session.Session, event streaming.Event, at time.Time) bool {
	status, ok := runStatusFromEvent(event.Type)
	if !ok {
		return false
	}
	return sess.SetLastRunState(status, event.TraceID, at)
}

func runStatusFromEvent(eventType streaming.EventType) (session.RunStatus, bool) {
	switch eventType {
	case streaming.EventRunStarted:
		return session.RunStatusRunning, true
	case streaming.EventAwaitingHuman:
		return session.RunStatusAwaitingHuman, true
	case streaming.EventDone:
		return session.RunStatusSuccess, true
	case streaming.EventError:
		return session.RunStatusError, true
	default:
		return "", false
	}
}
