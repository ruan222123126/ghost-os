package orchestration

import (
	"context"
	"errors"
	"time"

	"ghost-os/bridge/agent"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

var (
	ErrSessionInflight = internaltrace.ErrSessionInflight
	ErrRunNotFound     = internaltrace.ErrRunNotFound
	ErrRunCancelled    = internaltrace.ErrRunCancelled
	ErrRunRegistryNil  = internaltrace.ErrRunRegistryNil
)

const (
	sessionPushAssistantMessage    = internaltrace.SessionPushAssistantMessage
	sessionPushAwaitingHuman       = internaltrace.SessionPushAwaitingHuman
	sessionPushRunStarted          = internaltrace.SessionPushRunStarted
	sessionPushCompletionDelta     = internaltrace.SessionPushCompletionDelta
	sessionPushToolCallStarted     = internaltrace.SessionPushToolCallStarted
	sessionPushToolCallFinished    = internaltrace.SessionPushToolCallFinished
	sessionPushError               = internaltrace.SessionPushError
	sessionPushDone                = internaltrace.SessionPushDone
	sessionPushTaskRunCardStarted  = internaltrace.SessionPushTaskRunCardStarted
	sessionPushTaskRunCardEvent    = internaltrace.SessionPushTaskRunCardEvent
	sessionPushTaskRunCardFinished = internaltrace.SessionPushTaskRunCardFinished
)

type RunHandle = internaltrace.RunHandle
type RunRegistry = internaltrace.RunRegistry
type sessionPushEventType = internaltrace.SessionPushEventType
type sessionPushEvent = internaltrace.SessionPushEvent
type sessionPushHub = internaltrace.SessionPushHub
type streamTerminalBuffer = internaltrace.StreamTerminalBuffer

func NewRunRegistry() *RunRegistry {
	return internaltrace.NewRunRegistry()
}

func newSessionPushHub() *sessionPushHub {
	return internaltrace.NewSessionPushHub()
}

func newSessionStreamBroadcastSink(sink streaming.Sink, hub *sessionPushHub) streaming.Sink {
	return internaltrace.NewSessionStreamBroadcastSink(sink, hub)
}

func newSessionDraftCheckpointSink(
	sink streaming.Sink,
	sessionStore *session.Store,
	sess *session.Session,
) streaming.Sink {
	return internaltrace.NewSessionDraftCheckpointSink(sink, sessionStore, sess)
}

func ensureEventSink(sink streaming.Sink) streaming.Sink {
	return internaltrace.EnsureEventSink(sink)
}

func emitStreamEvent(ctx context.Context, sink streaming.Sink, event streaming.Event) error {
	return internaltrace.EmitStreamEvent(ctx, sink, event)
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
	return internaltrace.EmitStreamErrorEvent(ctx, sink, traceID, turn, stepID, sessionID, statusCode, err)
}

type eventTurnTracker struct {
	inner *internaltrace.EventTurnTracker
}

func newEventTurnTracker(sink streaming.Sink) *eventTurnTracker {
	return &eventTurnTracker{inner: internaltrace.NewEventTurnTracker(sink)}
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
	return internaltrace.NewStreamTerminalBuffer(sink)
}

func newSessionStreamLifecyclePayloadBuilder(turn *sessionTurnState) agent.StreamLifecyclePayloadBuilder {
	return internaltrace.NewSessionStreamLifecyclePayloadBuilder(turn.currentSessionID, parseSessionEndForStream)
}

func newSessionStreamLifecyclePayloadBuilderForSessionID(
	sessionID string,
) agent.StreamLifecyclePayloadBuilder {
	return internaltrace.NewSessionStreamLifecyclePayloadBuilder(
		func() string {
			return sessionID
		},
		parseSessionEndForStream,
	)
}

func parseSessionEndForStream(response string) (string, bool, error) {
	normalized, sessionEnd, err := parseSessionEndSignal(response)
	return normalized, sessionEnd != nil, err
}

func projectSessionTurnDraft(sess *session.Session, event streaming.Event, at time.Time) bool {
	return internaltrace.ProjectTurnDraft(sess, event, at)
}
