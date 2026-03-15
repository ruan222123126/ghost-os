package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"ghost-os/bridge/streaming"
)

type sseEventSink struct {
	w        http.ResponseWriter
	flusher  http.Flusher
	traceID  string
	sequence int
	mu       sync.Mutex
}

func newSSEEventSink(w http.ResponseWriter, flusher http.Flusher, traceID string) *sseEventSink {
	return &sseEventSink{
		w:       w,
		flusher: flusher,
		traceID: strings.TrimSpace(traceID),
	}
}

func (s *sseEventSink) Emit(ctx context.Context, event streaming.Event) (streaming.Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return event, ctx.Err()
	default:
	}

	if strings.TrimSpace(event.TraceID) == "" {
		event.TraceID = s.traceID
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}

	s.sequence++
	eventID, err := streaming.FormatEventID(event.TraceID, s.sequence)
	if err != nil {
		return event, err
	}
	event.ID = eventID

	data, err := json.Marshal(event)
	if err != nil {
		return event, err
	}

	if _, err := fmt.Fprintf(s.w, "id: %s\n", event.ID); err != nil {
		return event, err
	}
	if _, err := fmt.Fprintf(s.w, "event: %s\n", event.Type); err != nil {
		return event, err
	}
	if _, err := fmt.Fprintf(s.w, "data: %s\n\n", data); err != nil {
		return event, err
	}

	s.flusher.Flush()
	return event, nil
}
