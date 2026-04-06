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

func TestHandleBusTaskSystemListAction(t *testing.T) {
	handler := newTestHandler(t, nil)

	create := serveRequest(handler, http.MethodPost, "/api/tasks", `{
		"task_kind":"system_action",
		"action":"RSS_INBOX_POLL",
		"action_params":{"max_items_per_feed":3},
		"interval_seconds":60
	}`, nil)
	if create.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: got %d want %d", create.Code, http.StatusCreated)
	}

	list := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"TASK_SYSTEM_LIST","params":{},"trace_id":"trace-system-list"}`, nil)
	if list.Code != http.StatusOK {
		t.Fatalf("unexpected list status: got %d want %d", list.Code, http.StatusOK)
	}

	body := decodeResponseBody(t, list)
	items, ok := body.Payload.([]any)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", body.Payload)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected payload length: got %d want %d", len(items), 1)
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload item: %#v", items[0])
	}
	if first["task_kind"] != "system_action" {
		t.Fatalf("unexpected task_kind: %#v", first["task_kind"])
	}
}
