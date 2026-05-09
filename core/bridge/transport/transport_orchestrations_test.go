package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type orchestrationResponsePayload struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	TaskKind      string `json:"task_kind"`
	Orchestration struct {
		Nodes []any `json:"nodes"`
		Edges []any `json:"edges"`
	} `json:"orchestration"`
}

func TestHandleOrchestrationsCreateAndGet(t *testing.T) {
	handler := newTestHandler(t, nil)

	create := serveRequest(handler, http.MethodPost, "/api/orchestrations", `{
		"task_kind":"orchestration",
		"name":"Morning orchestration",
		"orchestration":{"nodes":[],"edges":[]},
		"interval_seconds":60
	}`, nil)
	if create.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: got %d want %d", create.Code, http.StatusCreated)
	}
	created := decodeOrchestrationResponsePayload(t, create)
	if created.TaskKind != "orchestration" {
		t.Fatalf("unexpected created task kind: %#v", created)
	}
	if created.Name != "Morning orchestration" {
		t.Fatalf("unexpected created orchestration name: %#v", created)
	}

	get := serveRequest(handler, http.MethodGet, "/api/orchestrations/"+created.ID, "", nil)
	if get.Code != http.StatusOK {
		t.Fatalf("unexpected get status: got %d want %d", get.Code, http.StatusOK)
	}
	got := decodeOrchestrationResponsePayload(t, get)
	if got.ID != created.ID {
		t.Fatalf("unexpected orchestration id: got %q want %q", got.ID, created.ID)
	}
	if got.TaskKind != "orchestration" {
		t.Fatalf("unexpected get task kind: %#v", got)
	}
}

func decodeOrchestrationResponsePayload(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) orchestrationResponsePayload {
	t.Helper()
	body := decodeResponseBody(t, recorder)
	data, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal orchestration payload: %v", err)
	}
	var payload orchestrationResponsePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode orchestration payload: %v", err)
	}
	return payload
}
