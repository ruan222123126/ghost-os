package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type systemTaskResponsePayload struct {
	ID           string         `json:"id"`
	TaskKind     string         `json:"task_kind"`
	Action       string         `json:"action,omitempty"`
	ActionParams map[string]any `json:"action_params,omitempty"`
	CronExpr     string         `json:"cron_expr,omitempty"`
}

func TestHandleTasksCreateGetAndUpdateSystemAction(t *testing.T) {
	handler := newTestHandler(t, nil)

	create := serveRequest(handler, http.MethodPost, "/api/tasks", `{
		"task_kind":"system_action",
		"action":"RSS_INBOX_POLL",
		"action_params":{"max_items_per_feed":5,"ai_batch_size":2},
		"interval_seconds":60
	}`, nil)
	if create.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: got %d want %d", create.Code, http.StatusCreated)
	}
	created := decodeSystemTaskResponsePayload(t, create)
	if created.TaskKind != "system_action" || created.Action != "RSS_INBOX_POLL" {
		t.Fatalf("unexpected created payload: %#v", created)
	}

	get := serveRequest(handler, http.MethodGet, "/api/tasks/"+created.ID, "", nil)
	if get.Code != http.StatusOK {
		t.Fatalf("unexpected get status: got %d want %d", get.Code, http.StatusOK)
	}
	got := decodeSystemTaskResponsePayload(t, get)
	if got.ActionParams["max_items_per_feed"] != float64(5) {
		t.Fatalf("unexpected get payload: %#v", got)
	}

	update := serveRequest(handler, http.MethodPatch, "/api/tasks/"+created.ID, `{
		"action":"RSS_BRIEFING_BUILD",
		"action_params":{"group_limit":3,"item_limit":8},
		"cron_expr":"*/10 * * * *"
	}`, nil)
	if update.Code != http.StatusOK {
		t.Fatalf("unexpected update status: got %d want %d", update.Code, http.StatusOK)
	}
	updated := decodeSystemTaskResponsePayload(t, update)
	if updated.Action != "RSS_BRIEFING_BUILD" || updated.CronExpr != "*/10 * * * *" {
		t.Fatalf("unexpected updated payload: %#v", updated)
	}
	if updated.ActionParams["group_limit"] != float64(3) {
		t.Fatalf("unexpected updated action params: %#v", updated.ActionParams)
	}
}

func decodeSystemTaskResponsePayload(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) systemTaskResponsePayload {
	t.Helper()
	body := decodeResponseBody(t, recorder)
	data, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal system task payload: %v", err)
	}
	var payload systemTaskResponsePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode system task payload: %v", err)
	}
	return payload
}
