package transport

import (
	"net/http"
	"strings"
	"testing"
)

func TestHandleTasksRejectsRemovedSystemAction(t *testing.T) {
	handler := newTestHandler(t, nil)

	create := serveRequest(handler, http.MethodPost, "/api/tasks", `{
		"task_kind":"system_action",
		"interval_seconds":60
	}`, nil)
	if create.Code != http.StatusBadRequest {
		t.Fatalf("unexpected create status: got %d want %d", create.Code, http.StatusBadRequest)
	}
	body := decodeResponseBody(t, create)
	if !strings.Contains(body.Error, `unsupported task_kind "system_action"`) {
		t.Fatalf("unexpected error payload: %q", body.Error)
	}
}
