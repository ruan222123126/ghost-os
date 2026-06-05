package runcards

import (
	"context"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
)

func TestStreamSinkBackfillsSourceSessionIDFromEvent(t *testing.T) {
	ctx, recorder, writer := newRecorderForTest(t)
	handle, err := recorder.StartCard(ctx, StartInput{
		Kind:      bridgeTasks.RunCardKindWorkflowAgent,
		Title:     "agent-node",
		StartedAt: time.Unix(1_700_100_000, 0),
	})
	if err != nil {
		t.Fatalf("start card: %v", err)
	}

	event := mustStreamEvent(t, "trace-1", "source-session", streaming.EventRunStarted)
	if _, err := NewStreamSink(handle).Emit(ctx, event); err != nil {
		t.Fatalf("emit event: %v", err)
	}

	snapshot := recorder.Snapshot()
	if got := snapshot[0].SourceSessionID; got != "source-session" {
		t.Fatalf("unexpected snapshot source session: got %q", got)
	}
	if len(snapshot[0].SourceEvents) != 1 {
		t.Fatalf("unexpected snapshot source events: %#v", snapshot[0].SourceEvents)
	}
	latest := writer.latest()
	if got := latest[0].SourceSessionID; got != "source-session" {
		t.Fatalf("unexpected persisted source session: got %q", got)
	}
	if len(latest[0].SourceEvents) != 1 {
		t.Fatalf("unexpected persisted source events: %#v", latest[0].SourceEvents)
	}
}

func TestStreamSinkRejectsMismatchedSourceSessionID(t *testing.T) {
	ctx, recorder, writer := newRecorderForTest(t)
	handle, err := recorder.StartCard(ctx, StartInput{
		Kind:            bridgeTasks.RunCardKindWorkflowAgent,
		Title:           "agent-node",
		SourceSessionID: "source-a",
	})
	if err != nil {
		t.Fatalf("start card: %v", err)
	}
	writesBefore := writer.count()

	event := mustStreamEvent(t, "trace-1", "source-b", streaming.EventCompletionDelta)
	_, err = NewStreamSink(handle).Emit(ctx, event)
	if err == nil || !strings.Contains(err.Error(), "source_session_id mismatch") {
		t.Fatalf("unexpected emit error: %v", err)
	}
	if got := recorder.Snapshot()[0].SourceSessionID; got != "source-a" {
		t.Fatalf("source session changed after mismatch: got %q", got)
	}
	if got := writer.count(); got != writesBefore {
		t.Fatalf("unexpected persist count after mismatch: got %d want %d", got, writesBefore)
	}
}

func TestStreamSinkMergesCompletionDeltaBeforePersist(t *testing.T) {
	ctx, recorder, writer := newRecorderForTest(t)
	now := time.Unix(1_700_100_000, 0).UTC()
	recorder.currentTime = func() time.Time { return now }
	handle, err := recorder.StartCard(ctx, StartInput{
		Kind:      bridgeTasks.RunCardKindWorkflowAgent,
		Title:     "agent-node",
		StartedAt: now,
	})
	if err != nil {
		t.Fatalf("start card: %v", err)
	}

	runStarted := mustStreamEvent(t, "trace-1", "source-session", streaming.EventRunStarted)
	runStarted.Payload = map[string]any{"session_id": "source-session"}
	if _, err := NewStreamSink(handle).Emit(ctx, runStarted); err != nil {
		t.Fatalf("emit run_started: %v", err)
	}

	now = now.Add(100 * time.Millisecond)
	textA := mustStreamEvent(t, "trace-1", "source-session", streaming.EventCompletionDelta)
	textA.StepID = "turn-0001-assistant"
	textA.Payload = map[string]any{"kind": "text", "text": "hel"}
	if _, err := NewStreamSink(handle).Emit(ctx, textA); err != nil {
		t.Fatalf("emit textA: %v", err)
	}

	now = now.Add(100 * time.Millisecond)
	textB := mustStreamEvent(t, "trace-1", "source-session", streaming.EventCompletionDelta)
	textB.StepID = "turn-0001-assistant"
	textB.Payload = map[string]any{"kind": "text", "text": "lo"}
	if _, err := NewStreamSink(handle).Emit(ctx, textB); err != nil {
		t.Fatalf("emit textB: %v", err)
	}

	if got := writer.count(); got != 2 {
		t.Fatalf("unexpected persist count before terminal event: got %d want 2", got)
	}

	now = now.Add(50 * time.Millisecond)
	done := mustStreamEvent(t, "trace-1", "source-session", streaming.EventDone)
	done.Payload = map[string]any{"session_id": "source-session"}
	if _, err := NewStreamSink(handle).Emit(ctx, done); err != nil {
		t.Fatalf("emit done: %v", err)
	}

	latest := writer.latest()
	if len(latest) != 1 || len(latest[0].SourceEvents) != 3 {
		t.Fatalf("unexpected persisted source events: %#v", latest)
	}
	delta := latest[0].SourceEvents[1]
	if got := delta.Type; got != string(streaming.EventCompletionDelta) {
		t.Fatalf("unexpected merged event type: %q", got)
	}
	if got := delta.Payload["text"]; got != "hello" {
		t.Fatalf("unexpected merged text payload: %#v", delta.Payload)
	}
}

func newRecorderForTest(
	t *testing.T,
) (context.Context, *Recorder, *recordingProgressWriter) {
	t.Helper()

	writer := &recordingProgressWriter{}
	ctx := bridgeTasks.WithRunSession(context.Background(), bridgeTasks.RunSession{
		SessionID:      "display-session",
		RunID:          "run-live",
		ProgressWriter: writer,
	})
	recorder, err := NewRecorder(ctx, trace.NewSessionPushHub())
	if err != nil {
		t.Fatalf("new recorder: %v", err)
	}
	return ctx, recorder, writer
}

func mustStreamEvent(
	t *testing.T,
	traceID string,
	sessionID string,
	eventType streaming.EventType,
) streaming.Event {
	t.Helper()

	event, err := streaming.NewEvent(traceID, sessionID, 0, "", eventType, nil)
	if err != nil {
		t.Fatalf("new event: %v", err)
	}
	return event
}

type recordingProgressWriter struct {
	updates []bridgeTasks.RunningRunLogUpdate
}

func (w *recordingProgressWriter) WriteRunningRunLog(
	update bridgeTasks.RunningRunLogUpdate,
) error {
	w.updates = append(w.updates, bridgeTasks.RunningRunLogUpdate{
		RunCards: bridgeTasks.CloneRunCards(update.RunCards),
	})
	return nil
}

func (w *recordingProgressWriter) latest() []bridgeTasks.RunCard {
	if len(w.updates) == 0 {
		return nil
	}
	return bridgeTasks.CloneRunCards(w.updates[len(w.updates)-1].RunCards)
}

func (w *recordingProgressWriter) count() int {
	return len(w.updates)
}
