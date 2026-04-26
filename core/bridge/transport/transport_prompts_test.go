package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
)

type systemPromptResponsePayload struct {
	CorePrompt     string                                 `json:"core_prompt"`
	RenderedPrompt string                                 `json:"rendered_prompt"`
	PromptLibrary  []bridgeconfig.SystemPromptLibraryItem `json:"prompt_library"`
}

func TestHandleSystemPromptsGetAndPatch(t *testing.T) {
	handler := newTestHandler(t, nil)
	promptsDir := os.Getenv("GHOST_PROMPTS_DIR")

	if _, err := bridgeconfig.UpdateSystemPromptFiles(promptsDir, bridgeconfig.SystemPromptUpdateRequest{
		CorePrompt: ptr("core block"),
	}); err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}

	getResp := serveRequest(handler, http.MethodGet, "/api/prompts/system", "", nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("unexpected GET status: got %d body=%s", getResp.Code, getResp.Body.String())
	}
	got := decodeSystemPromptPayload(t, getResp)
	if got.CorePrompt != "core block" {
		t.Fatalf("unexpected payload: %+v", got)
	}
	if len(got.PromptLibrary) != 1 || !got.PromptLibrary[0].Active {
		t.Fatalf("expected single active prompt library card, got %+v", got.PromptLibrary)
	}
	if !strings.Contains(got.RenderedPrompt, "core block") {
		t.Fatalf("expected rendered prompt to include core job override, got %q", got.RenderedPrompt)
	}
	if strings.Count(got.RenderedPrompt, "core block") != 1 {
		t.Fatalf("expected rendered prompt to include core job once, got %q", got.RenderedPrompt)
	}

	patchResp := serveRequest(handler, http.MethodPatch, "/api/prompts/system", `{"core_prompt":"patched core"}`, nil)
	if patchResp.Code != http.StatusOK {
		t.Fatalf("unexpected PATCH status: got %d body=%s", patchResp.Code, patchResp.Body.String())
	}
	updated := decodeSystemPromptPayload(t, patchResp)
	if updated.CorePrompt != "patched core" {
		t.Fatalf("unexpected updated payload: %+v", updated)
	}
	if len(updated.PromptLibrary) != 1 || updated.PromptLibrary[0].Content != "patched core" {
		t.Fatalf("expected updated prompt library to match patched core, got %+v", updated.PromptLibrary)
	}
	if !strings.Contains(updated.RenderedPrompt, "patched core") {
		t.Fatalf("expected rendered prompt to include patched core job, got %q", updated.RenderedPrompt)
	}
	if strings.Count(updated.RenderedPrompt, "patched core") != 1 {
		t.Fatalf("expected patched core job to render once, got %q", updated.RenderedPrompt)
	}
}

func TestHandleSystemPromptsRejectsEmptyPatch(t *testing.T) {
	handler := newTestHandler(t, nil)

	resp := serveRequest(handler, http.MethodPatch, "/api/prompts/system", `{}`, nil)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("unexpected PATCH status: got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestHandleSystemPromptsRejectsPatchWithCorePromptAndPromptLibrary(t *testing.T) {
	handler := newTestHandler(t, nil)

	resp := serveRequest(
		handler,
		http.MethodPatch,
		"/api/prompts/system",
		`{"core_prompt":"patched","prompt_library":[{"id":"card-a","name":"A","insert_point":"core_job","content":"patched","active":true}]}`,
		nil,
	)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("unexpected PATCH status: got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestHandleSystemPromptsReturnsInternalErrorForInvalidPromptRoot(t *testing.T) {
	handler := newTestHandler(t, nil)
	promptsDir := os.Getenv("GHOST_PROMPTS_DIR")

	if err := os.RemoveAll(promptsDir); err != nil {
		t.Fatalf("RemoveAll(%s): %v", promptsDir, err)
	}
	if err := os.WriteFile(promptsDir, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("WriteFile(%s): %v", promptsDir, err)
	}

	resp := serveRequest(handler, http.MethodGet, "/api/prompts/system", "", nil)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected GET status: got %d body=%s", resp.Code, resp.Body.String())
	}
}

func decodeSystemPromptPayload(t *testing.T, recorder *httptest.ResponseRecorder) systemPromptResponsePayload {
	t.Helper()

	body := decodeResponseBody(t, recorder)
	raw, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var payload systemPromptResponsePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return payload
}

func ptr(value string) *string {
	return &value
}
