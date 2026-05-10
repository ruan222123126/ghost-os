package orchestration

import (
	"context"
	"errors"
	"time"

	"ghost-os/bridge/agent"
	traceapp "ghost-os/bridge/orchestration/internal/app/trace"
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

var (
	ErrSessionInflight = traceapp.ErrSessionInflight
	ErrRunNotFound     = traceapp.ErrRunNotFound
	ErrRunCancelled    = traceapp.ErrRunCancelled
	ErrRunRegistryNil  = traceapp.ErrRunRegistryNil
)

const (
	sessionPushAssistantMessage = traceapp.SessionPushAssistantMessage
	sessionPushAwaitingHuman    = traceapp.SessionPushAwaitingHuman
	sessionPushRunStarted       = traceapp.SessionPushRunStarted
	sessionPushCompletionDelta  = traceapp.SessionPushCompletionDelta
	sessionPushToolCallStarted  = traceapp.SessionPushToolCallStarted
	sessionPushToolCallFinished = traceapp.SessionPushToolCallFinished
	sessionPushError            = traceapp.SessionPushError
	sessionPushDone             = traceapp.SessionPushDone
)

type RunHandle = traceapp.RunHandle
type RunRegistry = traceapp.RunRegistry
type sessionPushEventType = traceapp.SessionPushEventType
type sessionPushEvent = traceapp.SessionPushEvent
type sessionPushHub = traceapp.SessionPushHub
type streamTerminalBuffer = traceapp.StreamTerminalBuffer

func NewRunRegistry() *RunRegistry {
	return traceapp.NewRunRegistry()
}

func newSessionPushHub() *sessionPushHub {
	return traceapp.NewSessionPushHub()
}

func newSessionStreamBroadcastSink(sink streaming.Sink, hub *sessionPushHub) streaming.Sink {
	return traceapp.NewSessionStreamBroadcastSink(sink, hub)
}

func newSessionDraftCheckpointSink(
	sink streaming.Sink,
	sessionStore *session.Store,
	sess *session.Session,
) streaming.Sink {
	return traceapp.NewSessionDraftCheckpointSink(sink, sessionStore, sess)
}

func ensureEventSink(sink streaming.Sink) streaming.Sink {
	return traceapp.EnsureEventSink(sink)
}

func emitStreamEvent(ctx context.Context, sink streaming.Sink, event streaming.Event) error {
	return traceapp.EmitStreamEvent(ctx, sink, event)
}

func emitStreamErrorEvent(
	ctx context.Context,
	sink streaming.Sink,
	traceID string,
	turn int,
	stepID string,
	sessionID string,
	statusCode int,
	err error,
) error {
	return traceapp.EmitStreamErrorEvent(ctx, sink, traceID, turn, stepID, sessionID, statusCode, err)
}

type eventTurnTracker struct {
	inner *traceapp.EventTurnTracker
}

func newEventTurnTracker(sink streaming.Sink) *eventTurnTracker {
	return &eventTurnTracker{inner: traceapp.NewEventTurnTracker(sink)}
}

func (t *eventTurnTracker) Emit(ctx context.Context, event streaming.Event) (streaming.Event, error) {
	if t == nil || t.inner == nil {
		return event, errors.New("event turn tracker is not configured")
	}
	return t.inner.Emit(ctx, event)
}

func (t *eventTurnTracker) finalAssistantTurn() int {
	if t == nil || t.inner == nil {
		return 0
	}
	return t.inner.FinalAssistantTurn()
}

func newStreamTerminalBuffer(sink streaming.Sink) *streamTerminalBuffer {
	return traceapp.NewStreamTerminalBuffer(sink)
}

func newSessionStreamLifecyclePayloadBuilder(turn *sessionTurnState) agent.StreamLifecyclePayloadBuilder {
	return traceapp.NewSessionStreamLifecyclePayloadBuilder(turn.currentSessionID, parseSessionEndForStream)
}

func parseSessionEndForStream(response string) (string, bool, error) {
	normalized, sessionEnd, err := parseSessionEndSignal(response)
	return normalized, sessionEnd != nil, err
}

func projectSessionTurnDraft(sess *session.Session, event streaming.Event, at time.Time) bool {
	return sessionturn.ProjectTurnDraft(sess, event, at)
}
