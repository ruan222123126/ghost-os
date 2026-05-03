package transport

import (
	"net/http"
	"os"
	"reflect"
	"testing"
)

func TestHandlePresetsNormalizesContextPromptRefs(t *testing.T) {
	handler := newTestHandler(t, nil)
	promptsDir := os.Getenv("GHOST_PROMPTS_DIR")
	seedTransportPresetPromptLibrary(t, promptsDir)

	resp := serveRequest(
		handler,
		http.MethodPost,
		"/api/presets",
		`{"name":"Context","prompt_refs":{"context":[" context-b ","context-a","context-b","","context-a"]}}`,
		nil,
	)
	if resp.Code != http.StatusCreated {
		t.Fatalf("unexpected POST status: got %d body=%s", resp.Code, resp.Body.String())
	}

	payload := decodePresetPayload(t, resp)
	if !reflect.DeepEqual(payload.PromptRefs.Context, []string{"context-b", "context-a"}) {
		t.Fatalf("unexpected context refs: %+v", payload.PromptRefs)
	}
}

func TestHandlePresetsRejectsContextPromptWithWrongInsertPoint(t *testing.T) {
	handler := newTestHandler(t, nil)
	promptsDir := os.Getenv("GHOST_PROMPTS_DIR")
	seedTransportPresetPromptLibrary(t, promptsDir)

	resp := serveRequest(
		handler,
		http.MethodPost,
		"/api/presets",
		`{"name":"Invalid","prompt_refs":{"context":["rule-card"]}}`,
		nil,
	)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("unexpected POST status: got %d body=%s", resp.Code, resp.Body.String())
	}
}
