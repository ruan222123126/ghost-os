package trace

import "testing"

func TestSessionPushHubReplaysTraceWithoutLiveEventGap(t *testing.T) {
	hub := NewSessionPushHub()
	hub.Publish(SessionPushEvent{
		ID:        "trace-replay:000001",
		Type:      SessionPushRunStarted,
		TraceID:   "trace-replay",
		SessionID: "session-replay",
		Payload:   map[string]any{"session_id": "session-replay"},
	})
	hub.Publish(SessionPushEvent{
		ID:        "trace-replay:000002",
		Type:      SessionPushCompletionDelta,
		TraceID:   "trace-replay",
		SessionID: "session-replay",
		Payload:   map[string]any{"kind": "text", "text": "part"},
	})

	sessionID, replay, notifications, unsubscribe, ok := hub.SubscribeTrace(
		"trace-replay",
		"trace-replay:000001",
	)
	defer unsubscribe()
	if !ok {
		t.Fatal("expected replay subscription")
	}
	if sessionID != "session-replay" {
		t.Fatalf("unexpected session id: got %q", sessionID)
	}
	if len(replay) != 1 || replay[0].ID != "trace-replay:000002" {
		t.Fatalf("unexpected initial replay: %+v", replay)
	}

	hub.Publish(SessionPushEvent{
		ID:        "trace-replay:000003",
		Type:      SessionPushDone,
		TraceID:   "trace-replay",
		SessionID: "session-replay",
		Payload:   map[string]any{"session_ended": false},
	})
	<-notifications

	liveReplay := hub.ReplayTraceAfter("trace-replay", "trace-replay:000002")
	if len(liveReplay) != 1 || liveReplay[0].ID != "trace-replay:000003" {
		t.Fatalf("unexpected live replay: %+v", liveReplay)
	}
}

func TestSessionPushHubReplacesPreviousTraceReplayForSession(t *testing.T) {
	hub := NewSessionPushHub()
	hub.Publish(SessionPushEvent{
		ID:        "trace-old:000001",
		Type:      SessionPushRunStarted,
		TraceID:   "trace-old",
		SessionID: "session-replay",
	})
	hub.Publish(SessionPushEvent{
		ID:        "trace-new:000001",
		Type:      SessionPushRunStarted,
		TraceID:   "trace-new",
		SessionID: "session-replay",
	})

	if replay := hub.ReplayTraceAfter("trace-old", ""); len(replay) != 0 {
		t.Fatalf("expected old trace replay to be removed: %+v", replay)
	}
	if replay := hub.ReplayTraceAfter("trace-new", ""); len(replay) != 1 {
		t.Fatalf("expected new trace replay: %+v", replay)
	}
}
