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
