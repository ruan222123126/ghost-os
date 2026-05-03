package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
)

func TestHandlePresetsGetPostPatchDelete(t *testing.T) {
	handler := newTestHandler(t, nil)
	promptsDir := os.Getenv("GHOST_PROMPTS_DIR")
	seedTransportPresetPromptLibrary(t, promptsDir)

	initial := serveRequest(handler, http.MethodGet, "/api/presets", "", nil)
	if initial.Code != http.StatusOK {
		t.Fatalf("unexpected initial GET status: got %d body=%s", initial.Code, initial.Body.String())
	}
	if payload := decodePresetListPayload(t, initial); len(payload) != 0 {
		t.Fatalf("expected empty preset list, got %+v", payload)
	}

	createResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/presets",
		`{"name":"Default","tool_allowlist":["web_search","script_exec","web_search"],"prompt_refs":{"rule":"rule-card","memory":"memory-card"}}`,
		nil,
	)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("unexpected POST status: got %d body=%s", createResp.Code, createResp.Body.String())
	}
	created := decodePresetPayload(t, createResp)
	if created.ID == "" {
		t.Fatalf("expected non-empty preset id, got %+v", created)
	}

	listResp := serveRequest(handler, http.MethodGet, "/api/presets", "", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected list GET status: got %d body=%s", listResp.Code, listResp.Body.String())
	}
	listed := decodePresetListPayload(t, listResp)
	if len(listed) != 1 {
		t.Fatalf("expected one preset, got %+v", listed)
	}
	if listed[0].PromptRefs.Rule != "rule-card" || listed[0].PromptRefs.Memory != "memory-card" {
		t.Fatalf("unexpected created prompt refs: %+v", listed[0].PromptRefs)
	}

	patchResp := serveRequest(
		handler,
		http.MethodPatch,
		"/api/presets/"+created.ID,
		`{"name":"Updated","tool_allowlist":["sfind"],"prompt_refs":{"core_job":"core-card"}}`,
		nil,
	)
	if patchResp.Code != http.StatusOK {
		t.Fatalf("unexpected PATCH status: got %d body=%s", patchResp.Code, patchResp.Body.String())
	}
	updated := decodePresetPayload(t, patchResp)
	if updated.Name != "Updated" {
		t.Fatalf("unexpected updated preset: %+v", updated)
	}
	if updated.PromptRefs.Rule != "" || updated.PromptRefs.Memory != "" || updated.PromptRefs.CoreJob != "core-card" {
		t.Fatalf("unexpected updated prompt refs: %+v", updated.PromptRefs)
	}

	deleteResp := serveRequest(handler, http.MethodDelete, "/api/presets/"+created.ID, "", nil)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("unexpected DELETE status: got %d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	deleted := decodePresetPayload(t, deleteResp)
	if deleted.ID != created.ID {
		t.Fatalf("unexpected deleted preset: %+v", deleted)
	}

	finalResp := serveRequest(handler, http.MethodGet, "/api/presets", "", nil)
	if finalResp.Code != http.StatusOK {
		t.Fatalf("unexpected final GET status: got %d body=%s", finalResp.Code, finalResp.Body.String())
	}
	if payload := decodePresetListPayload(t, finalResp); len(payload) != 0 {
		t.Fatalf("expected empty preset list after delete, got %+v", payload)
	}
}

func TestHandlePresetsRejectsUnknownTool(t *testing.T) {
	handler := newTestHandler(t, nil)
	promptsDir := os.Getenv("GHOST_PROMPTS_DIR")
	seedTransportPresetPromptLibrary(t, promptsDir)

	resp := serveRequest(
		handler,
		http.MethodPost,
		"/api/presets",
		`{"name":"Invalid","tool_allowlist":["not_exists"]}`,
		nil,
	)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("unexpected POST status: got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestHandlePresetsRejectsMissingPromptRef(t *testing.T) {
	handler := newTestHandler(t, nil)
	promptsDir := os.Getenv("GHOST_PROMPTS_DIR")
	seedTransportPresetPromptLibrary(t, promptsDir)

	resp := serveRequest(
		handler,
		http.MethodPost,
		"/api/presets",
		`{"name":"Invalid","prompt_refs":{"rule":"missing-card"}}`,
		nil,
	)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("unexpected POST status: got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestHandlePresetsRejectsWrongPromptInsertPoint(t *testing.T) {
	handler := newTestHandler(t, nil)
	promptsDir := os.Getenv("GHOST_PROMPTS_DIR")
	seedTransportPresetPromptLibrary(t, promptsDir)

	resp := serveRequest(
		handler,
		http.MethodPost,
		"/api/presets",
		`{"name":"Invalid","prompt_refs":{"memory":"rule-card"}}`,
		nil,
	)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("unexpected POST status: got %d body=%s", resp.Code, resp.Body.String())
	}
}

func decodePresetListPayload(t *testing.T, recorder *httptest.ResponseRecorder) []bridgeconfig.Preset {
	t.Helper()

	body := decodeResponseBody(t, recorder)
	raw, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var payload []bridgeconfig.Preset
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return payload
}

func decodePresetPayload(t *testing.T, recorder *httptest.ResponseRecorder) bridgeconfig.Preset {
	t.Helper()

	body := decodeResponseBody(t, recorder)
	raw, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var payload bridgeconfig.Preset
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return payload
}

func seedTransportPresetPromptLibrary(t *testing.T, promptsDir string) {
	t.Helper()

	library := []bridgeconfig.SystemPromptLibraryItem{
		{
			ID:          "rule-card",
			Name:        "Rule",
			InsertPoint: bridgeconfig.SystemPromptInsertPointRule,
			Content:     "rule content",
			Active:      true,
		},
		{
			ID:          "core-card",
			Name:        "Core",
			InsertPoint: bridgeconfig.SystemPromptInsertPointCoreJob,
			Content:     "core content",
			Active:      true,
		},
		{
			ID:          "memory-card",
			Name:        "Memory",
			InsertPoint: bridgeconfig.SystemPromptInsertPointMemory,
			Content:     "memory content",
			Active:      true,
		},
		{
			ID:          "context-a",
			Name:        "Context A",
			InsertPoint: bridgeconfig.SystemPromptInsertPointContext,
			Content:     "context alpha",
			Active:      true,
		},
		{
			ID:          "context-b",
			Name:        "Context B",
			InsertPoint: bridgeconfig.SystemPromptInsertPointContext,
			Content:     "context beta",
			Active:      true,
		},
	}
	if _, err := bridgeconfig.UpdateSystemPromptFiles(promptsDir, bridgeconfig.SystemPromptUpdateRequest{
		PromptLibrary: &library,
	}); err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}
}
