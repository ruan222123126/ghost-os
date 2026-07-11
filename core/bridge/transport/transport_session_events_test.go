package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

func TestHandleSessionEventsStreamsAssistantMessage(t *testing.T) {
	executor := func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		return "sent to phone", "session-push", nil
	}
	handler, service, _ := newTestHandlerWithService(t, executor, nil)
	recorder, cancel, done := startSessionEventRequest(handler, "session-push")

	waitForSessionPushSubscriber(t, service.SessionPushHub(), "session-push")
	response := serveRequest(handler, http.MethodPost, "/api/agent", `{"message":"hello"}`, map[string]string{"Content-Type": "application/json"})
	if response.Code != http.StatusOK {
		cancel()
		<-done
		t.Fatalf("unexpected send status: got %d want %d", response.Code, http.StatusOK)
	}

	cancel()
	<-done

	events := decodeSessionPushEvents(t, recorder)
	if len(events) != 1 {
		t.Fatalf("unexpected event count: got %d want %d", len(events), 1)
	}
	if events[0].Type != bridgeorchestration.SessionPushAssistantMessage {
		t.Fatalf("unexpected event type: got %q want %q", events[0].Type, bridgeorchestration.SessionPushAssistantMessage)
	}
	payload, ok := events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", events[0].Payload)
	}
	if payload["message"] != "sent to phone" {
		t.Fatalf("unexpected message payload: got %v want %q", payload["message"], "sent to phone")
	}
}

func TestHandleSessionEventsStreamsAwaitingHuman(t *testing.T) {
	executor := func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		return "", "session-await", &agent.ErrAwaitingHuman{
			QuestionID: "q-1",
			Prompt:     "Approve?",
		}
	}
	handler, service, _ := newTestHandlerWithService(t, executor, nil)
	recorder, cancel, done := startSessionEventRequest(handler, "session-await")

	waitForSessionPushSubscriber(t, service.SessionPushHub(), "session-await")
	response := serveRequest(handler, http.MethodPost, "/api/agent", `{"message":"hello"}`, map[string]string{"Content-Type": "application/json"})
	if response.Code != http.StatusAccepted {
		cancel()
		<-done
		t.Fatalf("unexpected send status: got %d want %d", response.Code, http.StatusAccepted)
	}

	cancel()
	<-done

	events := decodeSessionPushEvents(t, recorder)
	if len(events) != 1 {
		t.Fatalf("unexpected event count: got %d want %d", len(events), 1)
	}
	if events[0].Type != bridgeorchestration.SessionPushAwaitingHuman {
		t.Fatalf("unexpected event type: got %q want %q", events[0].Type, bridgeorchestration.SessionPushAwaitingHuman)
	}
	payload, ok := events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", events[0].Payload)
	}
	if payload["question_id"] != "q-1" {
		t.Fatalf("unexpected question payload: got %v want %q", payload["question_id"], "q-1")
	}
}

func TestHandleSessionEventsEmitsPendingQuestionSnapshot(t *testing.T) {
	handler, service, sessionStore := newTestHandlerWithService(t, nil, nil)
	sess := session.NewSession("system")
	sess.ID = "session-pending"
	sess.AddPendingQuestion("q-pending", session.PendingHumanQuestion{
		Prompt:    "Need a choice",
		TraceID:   "trace-pending",
		CreatedAt: time.Unix(10, 0).UTC(),
	})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}
	_ = service

	recorder, cancel, done := startSessionEventRequest(handler, sess.ID)
	waitForBodyContains(t, recorder, "q-pending")
	cancel()
	<-done

	events := decodeSessionPushEvents(t, recorder)
	if len(events) != 1 {
		t.Fatalf("unexpected event count: got %d want %d", len(events), 1)
	}
	if events[0].TraceID != "trace-pending" {
		t.Fatalf("unexpected trace id: got %q want %q", events[0].TraceID, "trace-pending")
	}
}

func TestHandleSessionEventsBroadcastsStreamProgress(t *testing.T) {
	streamExecutor := func(
		ctx context.Context,
		_ string,
		_ string,
		traceID string,
		_ bridgeconfig.Store,
		_ *session.Store,
		sink streaming.Sink,
	) (string, string, error) {
		type runStartedPayload struct {
			SessionID string `json:"session_id"`
		}
		type messagePayload struct {
			Text string `json:"text"`
		}
		type donePayload struct {
			SessionEnded bool `json:"session_ended"`
		}

		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, "session-stream-push", 0, "", streaming.EventRunStarted, runStartedPayload{
			SessionID: "session-stream-push",
		})); err != nil {
			return "", "", err
		}
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, "session-stream-push", 0, mustAppToolStepID(t, 0, 0), streaming.EventToolCallStarted, map[string]any{
			"tool":         "list_files",
			"tool_call_id": "call-stream-1",
		})); err != nil {
			return "", "", err
		}
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, "session-stream-push", 0, mustAppToolStepID(t, 0, 0), streaming.EventToolCallFinished, map[string]any{
			"tool":         "list_files",
			"tool_call_id": "call-stream-1",
			"status":       "success",
		})); err != nil {
			return "", "", err
		}
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, "session-stream-push", 1, mustAppAssistantStepID(t, 1), streaming.EventMessage, messagePayload{
			Text: "stream finished",
		})); err != nil {
			return "", "", err
		}
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, "session-stream-push", 1, "", streaming.EventDone, donePayload{
			SessionEnded: false,
		})); err != nil {
			return "", "", err
		}
		return "stream finished", "session-stream-push", nil
	}
	handler, service, _ := newTestHandlerWithService(t, nil, streamExecutor)
	recorder, cancel, done := startSessionEventRequest(handler, "session-stream-push")

	waitForSessionPushSubscriber(t, service.SessionPushHub(), "session-stream-push")
	response := serveRequest(handler, http.MethodPost, "/api/agent/stream", `{"message":"hello"}`, map[string]string{"Content-Type": "application/json"})
	if response.Code != http.StatusOK {
		cancel()
		<-done
		t.Fatalf("unexpected send status: got %d want %d body=%s", response.Code, http.StatusOK, response.Body.String())
	}

	waitForBodyContains(t, recorder, "stream finished")
	cancel()
	<-done

	events := decodeSessionPushEvents(t, recorder)
	if len(events) != 5 {
		t.Fatalf("unexpected event count: got %d want %d", len(events), 5)
	}
	if events[0].Type != bridgeorchestration.SessionPushRunStarted {
		t.Fatalf("unexpected first event type: got %q want %q", events[0].Type, bridgeorchestration.SessionPushRunStarted)
	}
	if events[1].Type != bridgeorchestration.SessionPushToolCallStarted {
		t.Fatalf("unexpected second event type: got %q want %q", events[1].Type, bridgeorchestration.SessionPushToolCallStarted)
	}
	if events[2].Type != bridgeorchestration.SessionPushToolCallFinished {
		t.Fatalf("unexpected third event type: got %q want %q", events[2].Type, bridgeorchestration.SessionPushToolCallFinished)
	}
	if events[3].Type != bridgeorchestration.SessionPushDone {
		t.Fatalf("unexpected fourth event type: got %q want %q", events[3].Type, bridgeorchestration.SessionPushDone)
	}
	if events[4].Type != bridgeorchestration.SessionPushAssistantMessage {
		t.Fatalf("unexpected fifth event type: got %q want %q", events[4].Type, bridgeorchestration.SessionPushAssistantMessage)
	}
}

func TestHandleRunEventsReplaysAfterLastEventIDThroughTerminalEvent(t *testing.T) {
	handler, service, _ := newTestHandlerWithService(t, nil, nil)
	hub := service.SessionPushHub()
	hub.Publish(bridgeorchestration.SessionPushEvent{
		ID:        "trace-resume:000001",
		Type:      bridgeorchestration.SessionPushRunStarted,
		TraceID:   "trace-resume",
		SessionID: "session-resume",
		Payload:   map[string]any{"session_id": "session-resume"},
	})
	hub.Publish(bridgeorchestration.SessionPushEvent{
		ID:        "trace-resume:000002",
		Type:      bridgeorchestration.SessionPushCompletionDelta,
		TraceID:   "trace-resume",
		SessionID: "session-resume",
		Payload:   map[string]any{"kind": "text", "text": "continued"},
	})
	hub.Publish(bridgeorchestration.SessionPushEvent{
		ID:        "trace-resume:000003",
		Type:      bridgeorchestration.SessionPushDone,
		TraceID:   "trace-resume",
		SessionID: "session-resume",
		Payload:   map[string]any{"session_ended": false},
	})

	response := serveRequest(
		handler,
		http.MethodGet,
		"/api/runs/trace-resume/events",
		"",
		map[string]string{"Last-Event-ID": "trace-resume:000001"},
	)
	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d body=%s", response.Code, response.Body.String())
	}
	events := decodeAgentSSEEvents(t, response.Body.String())
	if len(events) != 2 {
		t.Fatalf("unexpected replay event count: got %d events=%+v", len(events), events)
	}
	if events[0].ID != "trace-resume:000002" || events[0].Type != streaming.EventCompletionDelta {
		t.Fatalf("unexpected replay delta: %+v", events[0])
	}
	if events[1].ID != "trace-resume:000003" || events[1].Type != streaming.EventDone {
		t.Fatalf("unexpected replay terminal event: %+v", events[1])
	}
}

func TestHandleRunEventsRejectsUnknownTrace(t *testing.T) {
	handler, _, _ := newTestHandlerWithService(t, nil, nil)

	response := serveRequest(handler, http.MethodGet, "/api/runs/missing/events", "", nil)

	if response.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: got %d want %d", response.Code, http.StatusNotFound)
	}
}

func decodeAgentSSEEvents(t *testing.T, body string) []streaming.Event {
	t.Helper()
	events := make([]streaming.Event, 0)
	for _, block := range strings.Split(body, "\n\n") {
		for _, line := range strings.Split(block, "\n") {
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			var event streaming.Event
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
				t.Fatalf("decode resumed agent event: %v", err)
			}
			events = append(events, event)
		}
	}
	return events
}

type decodedSessionPushEvent struct {
	ID        string                                   `json:"id"`
	Type      bridgeorchestration.SessionPushEventType `json:"type"`
	TraceID   string                                   `json:"trace_id"`
	SessionID string                                   `json:"session_id"`
	Payload   any                                      `json:"payload"`
}

func startSessionEventRequest(handler http.Handler, sessionID string) (*httptest.ResponseRecorder, context.CancelFunc, <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodGet, "/api/sessions/"+sessionID+"/events", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(recorder, request)
		close(done)
	}()
	return recorder, cancel, done
}

func waitForSessionPushSubscriber(t *testing.T, hub *bridgeorchestration.SessionPushHub, sessionID string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hub.SubscriberCount(sessionID) > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for subscriber on session %q", sessionID)
}

func waitForBodyContains(t *testing.T, recorder *httptest.ResponseRecorder, fragment string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(recorder.Body.String(), fragment) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for response body to contain %q; body=%q", fragment, recorder.Body.String())
}

func decodeSessionPushEvents(t *testing.T, recorder *httptest.ResponseRecorder) []decodedSessionPushEvent {
	t.Helper()
	trimmed := strings.TrimSpace(recorder.Body.String())
	if trimmed == "" {
		return nil
	}

	blocks := strings.Split(trimmed, "\n\n")
	events := make([]decodedSessionPushEvent, 0, len(blocks))
	for _, block := range blocks {
		if strings.TrimSpace(block) == "" || strings.HasPrefix(block, ":") {
			continue
		}
		var dataLine string
		for _, line := range strings.Split(block, "\n") {
			if strings.HasPrefix(line, "data: ") {
				dataLine = strings.TrimPrefix(line, "data: ")
				break
			}
		}
		if dataLine == "" {
			continue
		}
		var event decodedSessionPushEvent
		if err := json.Unmarshal([]byte(dataLine), &event); err != nil {
			t.Fatalf("decode session push event: %v", err)
		}
		events = append(events, event)
	}
	return events
}
