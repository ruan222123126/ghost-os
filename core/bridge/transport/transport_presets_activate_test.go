package transport

import (
	"net/http"
	"os"
	"reflect"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
)

func TestHandlePresetActivateAppliesToolAndPromptConfig(t *testing.T) {
	handler := newTestHandler(t, nil)
	promptsDir := os.Getenv("GHOST_PROMPTS_DIR")
	seedTransportPresetPromptLibrary(t, promptsDir)

	createResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/presets",
		`{"name":"Research","tool_allowlist":["web_search","script_exec"],"prompt_refs":{"rule":"rule-card","core_job":"core-card","context":["context-b","context-a"]}}`,
		nil,
	)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("unexpected POST status: got %d body=%s", createResp.Code, createResp.Body.String())
	}
	created := decodePresetPayload(t, createResp)

	activateResp := serveRequest(
		handler,
		http.MethodPut,
		"/api/presets/"+created.ID+"/activate",
		"",
		nil,
	)
	if activateResp.Code != http.StatusOK {
		t.Fatalf("unexpected activate status: got %d body=%s", activateResp.Code, activateResp.Body.String())
	}
	activated := decodePresetPayload(t, activateResp)
	if activated.ID != created.ID {
		t.Fatalf("unexpected activated preset: got %+v want %+v", activated, created)
	}

	tools := listToolsFromResponse(t, serveRequest(handler, http.MethodGet, "/api/tools", "", nil))
	if !mustFindTool(t, tools, "script_exec").Enabled {
		t.Fatal("expected script_exec to be enabled after preset activation")
	}
	if !mustFindTool(t, tools, "web_search").Enabled {
		t.Fatal("expected web_search to be enabled after preset activation")
	}
	if mustFindTool(t, tools, "sfind").Enabled {
		t.Fatal("expected sfind to be disabled after preset activation")
	}

	promptsResp := serveRequest(handler, http.MethodGet, "/api/prompts/system", "", nil)
	if promptsResp.Code != http.StatusOK {
		t.Fatalf("unexpected prompts GET status: got %d body=%s", promptsResp.Code, promptsResp.Body.String())
	}
	prompts := decodeSystemPromptPayload(t, promptsResp)
	if got := activeContextIDs(prompts.PromptLibrary); !reflect.DeepEqual(got, []string{"context-b", "context-a"}) {
		t.Fatalf("unexpected active context ids: got %v want %v", got, []string{"context-b", "context-a"})
	}

	configResp := serveRequest(handler, http.MethodGet, "/api/config", "", nil)
	if configResp.Code != http.StatusOK {
		t.Fatalf("unexpected config GET status: got %d body=%s", configResp.Code, configResp.Body.String())
	}
	configBody := decodeResponseBody(t, configResp)
	configPayload, ok := configBody.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected config payload type: %T", configBody.Payload)
	}
	if configPayload["model_selection_enabled"] != false {
		t.Fatalf("expected model_selection_enabled=false after strict preset activation, got %v", configPayload["model_selection_enabled"])
	}
}

func activeContextIDs(library []bridgeconfig.SystemPromptLibraryItem) []string {
	ids := make([]string, 0, len(library))
	for _, item := range library {
		if item.InsertPoint == bridgeconfig.SystemPromptInsertPointContext && item.Active {
			ids = append(ids, item.ID)
		}
	}
	return ids
}
