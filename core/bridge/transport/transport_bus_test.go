package transport

import (
	"net/http"
	"strings"
	"testing"
)

func TestHandleBusMethodNotAllowed(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodGet, "/api/bus", "", nil)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	body := decodeResponseBody(t, recorder)
	if body.Status != "error" {
		t.Fatalf("unexpected status field: got %q want %q", body.Status, "error")
	}
}

func TestHandleBusMissingFields(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodPost, "/api/bus", `{"params":{},"trace_id":"trace-1"}`, nil)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusBadRequest)
	}
	body := decodeResponseBody(t, recorder)
	if body.Error != "action is required" {
		t.Fatalf("unexpected error: got %q want %q", body.Error, "action is required")
	}
}

func TestHandleBusInvalidAction(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"UNKNOWN","params":{},"trace_id":"trace-unknown"}`, nil)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusBadRequest)
	}
	if recorder.Header().Get("X-Trace-ID") != "trace-unknown" {
		t.Fatalf("missing trace header: got %q want %q", recorder.Header().Get("X-Trace-ID"), "trace-unknown")
	}
	body := decodeResponseBody(t, recorder)
	if !strings.Contains(body.Error, "unsupported action") {
		t.Fatalf("unexpected error: %q", body.Error)
	}
}

func TestHandleBusMissingTraceID(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"TASK_LIST","params":{}}`, nil)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusBadRequest)
	}
	body := decodeResponseBody(t, recorder)
	if body.Error != "trace_id is required" {
		t.Fatalf("unexpected error: got %q want %q", body.Error, "trace_id is required")
	}
}
