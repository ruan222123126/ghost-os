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
	GlobalTemplate string `json:"global_template"`
	CorePrompt     string `json:"core_prompt"`
	ToolPrompt     string `json:"tool_prompt"`
	ToolKeySpec    string `json:"tool_key_spec"`
	RenderedPrompt string `json:"rendered_prompt"`
}

func TestHandleSystemPromptsGetAndPatch(t *testing.T) {
	handler := newTestHandler(t, nil)
	promptsDir := os.Getenv("GHOST_PROMPTS_DIR")

	if _, err := bridgeconfig.UpdateSystemPromptFiles(promptsDir, bridgeconfig.SystemPromptUpdateRequest{
		GlobalTemplate: ptr("BEGIN\n{{base_prompt}}\nEND\n{{core_prompt}}\n{{tool_prompt}}\n{{tool_key_spec}}"),
		CorePrompt:     ptr("core block"),
		ToolPrompt:     ptr("tool block"),
		ToolKeySpec:    ptr("key block"),
	}); err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}

	getResp := serveRequest(handler, http.MethodGet, "/api/prompts/system", "", nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("unexpected GET status: got %d body=%s", getResp.Code, getResp.Body.String())
	}
	got := decodeSystemPromptPayload(t, getResp)
	if got.GlobalTemplate != "BEGIN\n{{base_prompt}}\nEND\n{{core_prompt}}\n{{tool_prompt}}\n{{tool_key_spec}}" {
		t.Fatalf("unexpected template: %+v", got)
	}
	if !strings.Contains(got.RenderedPrompt, "core block") || !strings.Contains(got.RenderedPrompt, "tool block") || !strings.Contains(got.RenderedPrompt, "key block") {
		t.Fatalf("expected rendered prompt to include local injections, got %q", got.RenderedPrompt)
	}

	patchResp := serveRequest(handler, http.MethodPatch, "/api/prompts/system", `{"core_prompt":"patched core"}`, nil)
	if patchResp.Code != http.StatusOK {
		t.Fatalf("unexpected PATCH status: got %d body=%s", patchResp.Code, patchResp.Body.String())
	}
	updated := decodeSystemPromptPayload(t, patchResp)
	if updated.CorePrompt != "patched core" {
		t.Fatalf("unexpected updated payload: %+v", updated)
	}
	if !strings.Contains(updated.RenderedPrompt, "patched core") {
		t.Fatalf("expected rendered prompt to include patched core prompt, got %q", updated.RenderedPrompt)
	}
}

func TestHandleSystemPromptsRejectsEmptyPatch(t *testing.T) {
	handler := newTestHandler(t, nil)

	resp := serveRequest(handler, http.MethodPatch, "/api/prompts/system", `{}`, nil)
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
