package transport

import (
	"net/http"
	"strings"
	"testing"
)

func TestHandleTasksRejectsRSSSystemAction(t *testing.T) {
	handler := newTestHandler(t, nil)

	create := serveRequest(handler, http.MethodPost, "/api/tasks", `{
		"task_kind":"system_action",
		"action":"RSS_INBOX_POLL",
		"action_params":{"max_items_per_feed":5,"ai_batch_size":2},
		"interval_seconds":60
	}`, nil)
	if create.Code != http.StatusBadRequest {
		t.Fatalf("unexpected create status: got %d want %d", create.Code, http.StatusBadRequest)
	}
	body := decodeResponseBody(t, create)
	if !strings.Contains(body.Error, `unsupported system action "RSS_INBOX_POLL"`) {
		t.Fatalf("unexpected error payload: %q", body.Error)
	}
}
