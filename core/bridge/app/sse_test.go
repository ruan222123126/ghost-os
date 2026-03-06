package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"ghost-os/bridge/agent"
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

	err := sink.Emit(context.Background(), agent.AgentEvent{
		TraceID: "trace-123",
		Turn:    0,
		Type:    agent.EventRunStarted,
		Payload: map[string]any{"session_id": "session-1"},
		At:      time.Unix(0, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("Emit returned error: %v", err)
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
}

func TestSSEEventSinkHonorsContextCancellation(t *testing.T) {
	recorder := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	sink := newSSEEventSink(recorder, recorder, "trace-123")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := sink.Emit(ctx, agent.NewEvent("trace-123", 0, "", agent.EventDone, map[string]any{"ok": true}))
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
			if err := sink.Emit(context.Background(), agent.NewEvent("trace-123", turn, "", agent.EventDone, map[string]any{"turn": turn})); err != nil {
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

func decodeSSEDataLine(t *testing.T, body string) agent.AgentEvent {
	t.Helper()

	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var event agent.AgentEvent
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &event); err != nil {
			t.Fatalf("decode sse data: %v", err)
		}
		return event
	}

	t.Fatalf("missing data line in body: %q", body)
	return agent.AgentEvent{}
}
