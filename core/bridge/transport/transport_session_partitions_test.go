package transport

import (
	"net/http"
	"reflect"
	"testing"

	"ghost-os/bridge/session"
)

func TestHandleSessionPartitionsExactRouteReturnsEmptyState(t *testing.T) {
	handler := newTestHandler(t, nil)

	recorder := serveRequest(handler, http.MethodGet, "/api/sessions/partitions", "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}

	payload := decodeSessionPartitionPayload(t, decodeResponseBody(t, recorder).Payload)
	expectSessionPartitionPayload(t, payload, map[string]any{
		"version":     float64(1),
		"partitions":  []any{},
		"assignments": map[string]any{},
	})
}

func TestHandleSessionPartitionsPutAndGetRoundTrip(t *testing.T) {
	handler, sessionStore := newTestHandlerWithStore(t, nil)
	saveSessionForTransportPartitionTest(t, sessionStore, "session-a")
	saveSessionForTransportPartitionTest(t, sessionStore, "session-b")

	requestBody := `{
		"version": 1,
		"partitions": [
			{"id":"work","name":"Work"},
			{"id":"personal","name":"Personal"}
		],
		"assignments": {
			"session-a":"work",
			"session-b":"personal"
		},
		"trace_id":"trace-session-partitions-put"
	}`
	putRecorder := serveRequest(handler, http.MethodPut, "/api/sessions/partitions", requestBody, nil)
	if putRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected PUT status: got %d want %d body=%s", putRecorder.Code, http.StatusOK, putRecorder.Body.String())
	}

	expected := map[string]any{
		"version": float64(1),
		"partitions": []any{
			map[string]any{"id": "work", "name": "Work"},
			map[string]any{"id": "personal", "name": "Personal"},
		},
		"assignments": map[string]any{
			"session-a": "work",
			"session-b": "personal",
		},
	}
	expectSessionPartitionPayload(t, decodeSessionPartitionPayload(t, decodeResponseBody(t, putRecorder).Payload), expected)

	getRecorder := serveRequest(handler, http.MethodGet, "/api/sessions/partitions", "", nil)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected GET status: got %d want %d body=%s", getRecorder.Code, http.StatusOK, getRecorder.Body.String())
	}
	expectSessionPartitionPayload(t, decodeSessionPartitionPayload(t, decodeResponseBody(t, getRecorder).Payload), expected)
}

func TestHandleSessionPartitionsPutNormalizesDirtyPayload(t *testing.T) {
	handler, sessionStore := newTestHandlerWithStore(t, nil)
	saveSessionForTransportPartitionTest(t, sessionStore, "session-a")
	saveSessionForTransportPartitionTest(t, sessionStore, "session-b")

	requestBody := `{
		"version": 1,
		"partitions": [
			{"id":"work","name":"Work"},
			{"id":"work","name":"Duplicate"},
			{"id":"__unclassified__","name":"Reserved"},
			{"id":"personal","name":"Personal"}
		],
		"assignments": {
			"session-a":"work",
			"session-b":"missing",
			"session-missing":"work"
		}
	}`
	recorder := serveRequest(handler, http.MethodPut, "/api/sessions/partitions", requestBody, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	expectSessionPartitionPayload(t, decodeSessionPartitionPayload(t, decodeResponseBody(t, recorder).Payload), map[string]any{
		"version": float64(1),
		"partitions": []any{
			map[string]any{"id": "work", "name": "Work"},
			map[string]any{"id": "personal", "name": "Personal"},
		},
		"assignments": map[string]any{
			"session-a": "work",
		},
	})
}

func saveSessionForTransportPartitionTest(t *testing.T, store *session.Store, id string) {
	t.Helper()
	sess := session.NewSession("system")
	sess.ID = id
	if err := store.Save(sess); err != nil {
		t.Fatalf("save session %s: %v", id, err)
	}
}

func decodeSessionPartitionPayload(t *testing.T, payload any) map[string]any {
	t.Helper()
	decoded, ok := payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", payload)
	}
	return decoded
}

func expectSessionPartitionPayload(t *testing.T, got map[string]any, want map[string]any) {
	t.Helper()
	if got["version"] != want["version"] {
		t.Fatalf("unexpected version: got %v want %v", got["version"], want["version"])
	}
	if !reflectValueEqual(got["partitions"], want["partitions"]) {
		t.Fatalf("unexpected partitions: got=%v want=%v", got["partitions"], want["partitions"])
	}
	if !reflectValueEqual(got["assignments"], want["assignments"]) {
		t.Fatalf("unexpected assignments: got=%v want=%v", got["assignments"], want["assignments"])
	}
}

func reflectValueEqual(a any, b any) bool {
	return reflect.DeepEqual(a, b)
}
