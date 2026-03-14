package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"ghost-os/bridge/streaming"
)

type flushRecorder struct {
	*httptest.ResponseRecorder
}

func (f *flushRecorder) Flush() {
	f.ResponseRecorder.Flush()
}

func TestSSEEventSinkFormatsEvents(t *testing.T) {
	recorder := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	sink := newSSEEventSink(recorder, recorder, "trace-123")

	event, err := sink.Emit(context.Background(), streaming.Event{
		Turn:      0,
		Type:      streaming.EventRunStarted,
		SessionID: "session-1",
		Payload:   map[string]any{"session_id": "session-1"},
	})
	if err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}
	if event.TraceID != "trace-123" {
		t.Fatalf("unexpected canonical trace_id: got %q want %q", event.TraceID, "trace-123")
	}
	if event.ID != "trace-123:000001" {
		t.Fatalf("unexpected canonical event id: got %q want %q", event.ID, "trace-123:000001")
	}
	if event.At.IsZero() {
		t.Fatal("expected canonical event time to be set")
	}

	body := recorder.Body.String()
	if !strings.Contains(body, "id: trace-123:000001\n") {
		t.Fatalf("missing event id in body: %q", body)
	}
	if !strings.Contains(body, "event: run_started\n") {
		t.Fatalf("missing event name in body: %q", body)
	}
	if !strings.Contains(body, "data: ") {
		t.Fatalf("missing data line in body: %q", body)
	}
	if got := decodeSSEDataLine(t, body); !reflect.DeepEqual(got, event) {
		t.Fatalf("unexpected SSE payload event: got %+v want %+v", got, event)
	}
}

func TestSSEEventSinkHonorsContextCancellation(t *testing.T) {
	recorder := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	sink := newSSEEventSink(recorder, recorder, "trace-123")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := sink.Emit(ctx, mustAppEvent(t, "trace-123", "", 0, "", streaming.EventDone, map[string]any{"ok": true}))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected error: got %v want %v", err, context.Canceled)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty body, got %q", recorder.Body.String())
	}
}

func TestSSEEventSinkSerializesConcurrentEmitCalls(t *testing.T) {
	recorder := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	sink := newSSEEventSink(recorder, recorder, "trace-123")

	var wg sync.WaitGroup
	for index := 0; index < 8; index++ {
		wg.Add(1)
		go func(turn int) {
			defer wg.Done()
			if _, err := sink.Emit(context.Background(), mustAppEvent(t, "trace-123", "", turn, "", streaming.EventDone, map[string]any{"turn": turn})); err != nil {
				t.Errorf("Emit returned error: %v", err)
			}
		}(index)
	}
	wg.Wait()

	if got := strings.Count(recorder.Body.String(), "event: done\n"); got != 8 {
		t.Fatalf("unexpected event count in body: got %d want %d", got, 8)
	}

	blocks := strings.Split(strings.TrimSpace(recorder.Body.String()), "\n\n")
	if len(blocks) != 8 {
		t.Fatalf("unexpected block count: got %d want %d", len(blocks), 8)
	}
	lastBlock := blocks[len(blocks)-1]
	if !strings.Contains(lastBlock, "id: trace-123:000008\n") {
		t.Fatalf("unexpected final event id: %q", lastBlock)
	}
}

func TestSessionStreamBroadcastSinkPublishesCanonicalEventFromInnerSink(t *testing.T) {
	recorder := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	sseSink := newSSEEventSink(recorder, recorder, "trace-canonical")
	hub := newSessionPushHub()
	streamSink := newSessionStreamBroadcastSink(sseSink, hub)

	subscription, unsubscribe := hub.Subscribe("session-canonical")
	defer unsubscribe()

	emitted, err := streamSink.Emit(context.Background(), mustAppEvent(
		t,
		"",
		"session-canonical",
		1,
		"",
		streaming.EventDone,
		map[string]any{"session_ended": false},
	))
	if err != nil {
		t.Fatalf("Emit returned error: %v", err)
	}
	if emitted.TraceID != "trace-canonical" {
		t.Fatalf("unexpected canonical trace_id: got %q want %q", emitted.TraceID, "trace-canonical")
	}
	if emitted.ID != "trace-canonical:000001" {
		t.Fatalf("unexpected canonical event id: got %q want %q", emitted.ID, "trace-canonical:000001")
	}
	if emitted.At.IsZero() {
		t.Fatal("expected canonical event time to be set")
	}

	sseEvent := decodeSSEDataLine(t, recorder.Body.String())
	if sseEvent.ID != emitted.ID || sseEvent.TraceID != emitted.TraceID || sseEvent.SessionID != emitted.SessionID || sseEvent.Turn != emitted.Turn || sseEvent.Type != emitted.Type || !sseEvent.At.Equal(emitted.At) {
		t.Fatalf("unexpected SSE envelope event: got %+v want %+v", sseEvent, emitted)
	}
	payload, ok := sseEvent.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected SSE payload type: %T", sseEvent.Payload)
	}
	if payload["session_ended"] != false {
		t.Fatalf("unexpected SSE payload: %+v", payload)
	}

	select {
	case pushed := <-subscription:
		if pushed.ID != emitted.ID {
			t.Fatalf("unexpected push event id: got %q want %q", pushed.ID, emitted.ID)
		}
		if pushed.TraceID != emitted.TraceID {
			t.Fatalf("unexpected push trace_id: got %q want %q", pushed.TraceID, emitted.TraceID)
		}
		if !pushed.At.Equal(emitted.At) {
			t.Fatalf("unexpected push time: got %s want %s", pushed.At, emitted.At)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for pushed event")
	}
}

func decodeSSEDataLine(t *testing.T, body string) streaming.Event {
	t.Helper()

	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var event streaming.Event
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
			t.Fatalf("decode sse data: %v", err)
		}
		return event
	}

	t.Fatalf("missing data line in body: %q", body)
	return streaming.Event{}
}
