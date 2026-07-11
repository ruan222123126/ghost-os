package sessiondraft

import (
	"strings"
	"time"

	bridgesession "ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

const (
	draftAssistantOrderPrefix = "assistant:"
	draftThinkingOrderPrefix  = "thinking:"
	draftToolOrderPrefix      = "tool:"

	draftAssistantSegmentPrefix = "stream-segment:assistant:"
	draftThinkingSegmentPrefix  = "stream-segment:thinking:"

	draftToolPendingStatus = "pending"
	draftToolRunningStatus = "running"
	draftToolSuccessStatus = "success"
	draftToolErrorStatus   = "error"
)

func ProjectTurnDraft(
	sess *bridgesession.Session,
	event streaming.Event,
	at time.Time,
) bool {
	switch event.Type {
	case streaming.EventRunStarted:
		return projectSessionTurnDraftRunStarted(sess, event, at)
	case streaming.EventCompletionDelta:
		return projectSessionTurnDraftCompletionDelta(sess, event, at)
	case streaming.EventToolCallStarted:
		return projectSessionTurnDraftToolStarted(sess, event, at)
	case streaming.EventToolCallFinished:
		return projectSessionTurnDraftToolFinished(sess, event, at)
	case streaming.EventAwaitingHuman:
		return projectSessionTurnDraftAwaitingHuman(sess, event, at)
	case streaming.EventError:
		return projectSessionTurnDraftError(sess, event, at)
	case streaming.EventDone:
		return clearSessionTurnDraft(sess, event, at)
	default:
		return false
	}
}

func ensureSessionTurnDraft(
	sess *bridgesession.Session,
	event streaming.Event,
	at time.Time,
) (*bridgesession.TurnDraft, bool) {
	if sess == nil || strings.TrimSpace(event.TraceID) == "" {
		return nil, false
	}

	if sess.TurnDraft != nil &&
		strings.TrimSpace(sess.TurnDraft.TraceID) == strings.TrimSpace(event.TraceID) &&
		sess.TurnDraft.Turn == event.Turn {
		return sess.TurnDraft, false
	}

	sess.TurnDraft = &bridgesession.TurnDraft{
		TraceID: strings.TrimSpace(event.TraceID),
		Turn:    event.Turn,
		Status:  bridgesession.TurnDraftStatusStreaming,
		ToolTagState: &bridgesession.TurnDraftToolTagState{
			Mode:        "normal",
			NextCallSeq: 1,
		},
	}
	sess.UpdatedAt = at.UTC()
	return sess.TurnDraft, true
}

func clearSessionTurnDraft(sess *bridgesession.Session, event streaming.Event, at time.Time) bool {
	if sess == nil || sess.TurnDraft == nil {
		return false
	}
	if strings.TrimSpace(sess.TurnDraft.TraceID) != strings.TrimSpace(event.TraceID) {
		return false
	}
	if sess.TurnDraft.Turn != event.Turn {
		return false
	}
	return sess.ClearTurnDraft(at.UTC())
}
