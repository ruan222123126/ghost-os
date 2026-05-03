package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type toolResponsePayload struct {
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	PromptOverride  string `json:"prompt_override,omitempty"`
	SandboxMemoryMB *int   `json:"sandbox_memory_mb,omitempty"`
	InputSchema     any    `json:"input_schema,omitempty"`
}

func TestHandleToolsListAndPatch(t *testing.T) {
	handler := newTestHandler(t, nil)

	initial := listToolsFromResponse(t, serveRequest(handler, http.MethodGet, "/api/tools", "", nil))
	initialScriptExec := mustFindTool(t, initial, "script_exec")
	if initialScriptExec.Enabled {
		t.Fatal("expected script_exec to be disabled before allowlist update")
	}

	update := serveRequest(
		handler,
		http.MethodPatch,
		"/api/tools/script_exec",
		`{"enabled":true,"prompt_override":"use only when needed","sandbox_memory_mb":384}`,
		nil,
	)
	if update.Code != http.StatusOK {
		t.Fatalf("unexpected patch status: got %d body=%s", update.Code, update.Body.String())
	}

	afterEnable := listToolsFromResponse(t, serveRequest(handler, http.MethodGet, "/api/tools", "", nil))
	updatedScriptExec := mustFindTool(t, afterEnable, "script_exec")
	if !updatedScriptExec.Enabled {
		t.Fatal("expected script_exec to be enabled after patch")
	}
	if updatedScriptExec.PromptOverride != "use only when needed" {
		t.Fatalf("unexpected prompt override: %q", updatedScriptExec.PromptOverride)
	}
	if updatedScriptExec.SandboxMemoryMB == nil || *updatedScriptExec.SandboxMemoryMB != 384 {
		t.Fatalf("unexpected sandbox memory: %+v", updatedScriptExec.SandboxMemoryMB)
	}
	assertToolPromptFile(t, "script_exec", "use only when needed")

	reset := serveRequest(
		handler,
		http.MethodPatch,
		"/api/tools/script_exec",
		`{"enabled":false,"prompt_override":"","sandbox_memory_mb":256}`,
		nil,
	)
	if reset.Code != http.StatusOK {
		t.Fatalf("unexpected reset status: got %d body=%s", reset.Code, reset.Body.String())
	}
	afterDisable := listToolsFromResponse(t, serveRequest(handler, http.MethodGet, "/api/tools", "", nil))
	resetScriptExec := mustFindTool(t, afterDisable, "script_exec")
	if resetScriptExec.Enabled {
		t.Fatal("expected script_exec to be disabled after reset")
	}
	if resetScriptExec.SandboxMemoryMB == nil || *resetScriptExec.SandboxMemoryMB != 256 {
		t.Fatalf("unexpected reset sandbox memory: %+v", resetScriptExec.SandboxMemoryMB)
	}
	assertToolPromptFile(t, "script_exec", "")
}

func TestHandleToolPatchValidation(t *testing.T) {
	handler := newTestHandler(t, nil)

	unknown := serveRequest(
		handler,
		http.MethodPatch,
		"/api/tools/not_exists",
		`{"enabled":false}`,
		nil,
	)
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unexpected status for unknown tool: got %d body=%s", unknown.Code, unknown.Body.String())
	}

	invalidBody := serveRequest(
		handler,
		http.MethodPatch,
		"/api/tools/script_exec",
		`{}`,
		nil,
	)
	if invalidBody.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status for invalid body: got %d body=%s", invalidBody.Code, invalidBody.Body.String())
	}

	invalidMemory := serveRequest(
		handler,
		http.MethodPatch,
		"/api/tools/web_search",
		`{"sandbox_memory_mb":384}`,
		nil,
	)
	if invalidMemory.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status for invalid sandbox memory patch: got %d body=%s", invalidMemory.Code, invalidMemory.Body.String())
	}
}

func TestHandleToolsListIncludesInputSchema(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodGet, "/api/tools", "", nil)
	assertToolsListHasNoNullInputSchema(t, recorder)
	items := listToolsFromResponse(t, recorder)
	assertToolSchemaHasProperty(t, mustFindTool(t, items, "script_exec"), "script")
	assertToolSchemaHasProperty(t, mustFindTool(t, items, "search_files"), "query")
	assertToolSchemaHasProperty(t, mustFindTool(t, items, "write_file"), "content")
	assertToolSchemaHasProperty(t, mustFindTool(t, items, "bash_exec"), "command")
}

func assertToolsListHasNoNullInputSchema(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	body := decodeResponseBody(t, recorder)
	payload, ok := body.Payload.([]any)
	if !ok {
		t.Fatalf("unexpected tools payload type: %T", body.Payload)
	}

	for index, entry := range payload {
		record, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("unexpected tools payload entry %d type: %T", index, entry)
		}
		inputSchemaValue, hasInputSchema := record["input_schema"]
		if hasInputSchema && inputSchemaValue == nil {
			t.Fatalf("expected tools payload[%d].input_schema to be omitted, got null", index)
		}
	}
}

func listToolsFromResponse(t *testing.T, recorder *httptest.ResponseRecorder) []toolResponsePayload {
	t.Helper()

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected list status: got %d body=%s", recorder.Code, recorder.Body.String())
	}
	body := decodeResponseBody(t, recorder)
	raw, err := json.Marshal(body.Payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	var tools []toolResponsePayload
	if err := json.Unmarshal(raw, &tools); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return tools
}

func mustFindTool(t *testing.T, items []toolResponsePayload, name string) toolResponsePayload {
	t.Helper()

	for _, item := range items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("tool not found: %s", name)
	return toolResponsePayload{}
}

func assertToolSchemaHasProperty(t *testing.T, tool toolResponsePayload, property string) {
	t.Helper()

	schema, ok := tool.InputSchema.(map[string]any)
	if !ok {
		t.Fatalf("expected %s input_schema object, got %T", tool.Name, tool.InputSchema)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected %s schema.properties object, got %T", tool.Name, schema["properties"])
	}
	if _, ok := properties[property]; !ok {
		t.Fatalf("expected %s schema to contain %q, got %v", tool.Name, property, properties)
	}
}

func assertToolPromptFile(t *testing.T, name string, want string) {
	t.Helper()

	path := filepath.Join(strings.TrimSpace(os.Getenv("GHOST_PROMPTS_DIR")), "tools", name+".md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if got := strings.TrimSpace(string(raw)); got != want {
		t.Fatalf("unexpected prompt file content for %s: got %q want %q", name, got, want)
	}
}
