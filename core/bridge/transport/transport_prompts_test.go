package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	bridgetools "ghost-os/bridge/tools"
)

type systemPromptResponsePayload struct {
	CorePrompt      string                                 `json:"core_prompt"`
	RenderedPrompt  string                                 `json:"rendered_prompt"`
	PromptLibrary   []bridgeconfig.SystemPromptLibraryItem `json:"prompt_library"`
	ToolDefinitions []systemPromptToolDefinitionPayload    `json:"tool_definitions"`
}

type systemPromptToolDefinitionPayload struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

func TestHandleSystemPromptsGetAndPatch(t *testing.T) {
	t.Setenv("GHOST_TOOL_SEARCH_ENABLED", "true")

	handler := newTestHandler(t, nil)
	promptsDir := os.Getenv("GHOST_PROMPTS_DIR")

	if _, err := bridgeconfig.UpdateSystemPromptFiles(promptsDir, bridgeconfig.SystemPromptUpdateRequest{
		CorePrompt: ptr("core block"),
	}); err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}
	for _, toolName := range []string{
		"ask_human",
		"list_files",
		"read_file",
		"search_files",
		"write_file",
		"apply_diff",
		"bash_exec",
		"codex_cli",
		"screen_control",
		"script_exec",
		"sfind",
		"web_search",
	} {
		enableToolForPromptPreviewTest(t, handler, toolName)
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
	for _, snippet := range []string{"Role:", "Job:", "Skills:", "Skill Context:", "Context:", "core block"} {
		if !strings.Contains(got.RenderedPrompt, snippet) {
			t.Fatalf("expected rendered prompt to include %q, got %q", snippet, got.RenderedPrompt)
		}
	}
	if strings.Contains(got.RenderedPrompt, "## Dynamic Tool State") {
		t.Fatalf("expected rendered prompt to exclude legacy dynamic tool section, got %q", got.RenderedPrompt)
	}
	if strings.Count(got.RenderedPrompt, "core block") != 1 {
		t.Fatalf("expected rendered prompt to include core job once, got %q", got.RenderedPrompt)
	}
	assertExpectedSystemPromptToolDefinitions(t, got.ToolDefinitions)

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
	for _, snippet := range []string{"Role:", "Job:", "Skills:", "Skill Context:", "Context:", "patched core"} {
		if !strings.Contains(updated.RenderedPrompt, snippet) {
			t.Fatalf("expected updated rendered prompt to include %q, got %q", snippet, updated.RenderedPrompt)
		}
	}
	if strings.Count(updated.RenderedPrompt, "patched core") != 1 {
		t.Fatalf("expected patched core job to render once, got %q", updated.RenderedPrompt)
	}
	assertExpectedSystemPromptToolDefinitions(t, updated.ToolDefinitions)
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

func TestHandleSystemPromptsPromptLibraryRuleUpdatesRenderedPrompt(t *testing.T) {
	handler := newTestHandler(t, nil)

	resp := serveRequest(
		handler,
		http.MethodPatch,
		"/api/prompts/system",
		`{"prompt_library":[{"id":"rule-card","name":"Rule","insert_point":"rule","content":"You are the patched rule.","active":true}]}`,
		nil,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected PATCH status: got %d body=%s", resp.Code, resp.Body.String())
	}

	payload := decodeSystemPromptPayload(t, resp)
	if len(payload.PromptLibrary) != 1 || payload.PromptLibrary[0].InsertPoint != bridgeconfig.SystemPromptInsertPointRule {
		t.Fatalf("expected rule prompt card, got %+v", payload.PromptLibrary)
	}
	if !strings.Contains(payload.RenderedPrompt, "You are the patched rule.") {
		t.Fatalf("expected rendered prompt to include rule override, got %q", payload.RenderedPrompt)
	}
	if strings.Contains(payload.RenderedPrompt, "Ghost-OS bridge agent (digital twin execution layer).") {
		t.Fatalf("expected default rule to be replaced, got %q", payload.RenderedPrompt)
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

func enableToolForPromptPreviewTest(t *testing.T, handler http.Handler, toolName string) {
	t.Helper()

	resp := serveRequest(
		handler,
		http.MethodPatch,
		"/api/tools/"+toolName,
		`{"enabled":true}`,
		nil,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("enable tool %q: status=%d body=%s", toolName, resp.Code, resp.Body.String())
	}
}

func hasPromptToolDefinition(items []systemPromptToolDefinitionPayload, toolName string) bool {
	for _, item := range items {
		if item.Name == toolName {
			return true
		}
	}
	return false
}

func promptToolDefinitionDescription(items []systemPromptToolDefinitionPayload, toolName string) (string, bool) {
	for _, item := range items {
		if item.Name == toolName {
			return item.Description, true
		}
	}
	return "", false
}

func assertExpectedSystemPromptToolDefinitions(t *testing.T, items []systemPromptToolDefinitionPayload) {
	t.Helper()

	for toolName, definition := range expectedSystemPromptToolDefinitions() {
		assertSystemPromptToolDefinition(t, items, toolName, definition.description, definition.parameters)
	}
}

func expectedSystemPromptToolDefinitions() map[string]struct {
	description string
	parameters  map[string]any
} {
	listFilesPrompt := mustToolBasePrompt("list_files")
	readFilePrompt := mustToolBasePrompt("read_file")
	searchFilesPrompt := mustToolBasePrompt("search_files")
	writeFilePrompt := mustToolBasePrompt("write_file")
	applyDiffPrompt := mustToolBasePrompt("apply_diff")
	bashExecPrompt := mustToolBasePrompt("bash_exec")
	scriptExecPrompt := mustToolBasePrompt("script_exec")

	return map[string]struct {
		description string
		parameters  map[string]any
	}{
		"ask_human": {
			description: "Block and ask user for input. If 'options' are provided, the final option MUST set allow_custom=true.",
			parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prompt": map[string]any{
						"type": "string",
					},
					"selection_mode": map[string]any{
						"type": "string",
						"enum": []any{"single", "multiple"},
					},
					"options": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"label": map[string]any{
									"type": "string",
								},
								"allow_custom": map[string]any{
									"type": "boolean",
								},
							},
							"required":             []any{"label"},
							"additionalProperties": false,
						},
					},
				},
				"required":             []any{"prompt"},
				"additionalProperties": false,
			},
		},
		"apply_diff": {
			description: applyDiffPrompt,
			parameters:  mustToolSchema(bridgetools.NewApplyDiffTool(nil)),
		},
		"bash_exec": {
			description: bashExecPrompt,
			parameters:  mustToolSchema(bridgetools.NewBashExecTool(nil)),
		},
		"codex_cli": {
			description: "Async codex runner. Rules: 'prompt' required for start/resume. 'session_id' required for resume/status (pass command_id here for status). Omit model/sandbox to use local Codex config. Fork is interactive-only in Codex CLI 0.130.0. DO NOT use 'exec'.",
			parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"op": map[string]any{
						"type": "string",
						"enum": []any{"start", "resume", "status"},
					},
					"prompt": map[string]any{
						"type": "string",
					},
					"session_id": map[string]any{
						"type": "string",
					},
					"cwd": map[string]any{
						"type": "string",
					},
					"output_path": map[string]any{
						"type": "string",
					},
					"model": map[string]any{
						"type":        "string",
						"description": "Optional. If omitted, Codex uses local config.",
					},
					"sandbox": map[string]any{
						"type":        "string",
						"enum":        []any{"read-only", "workspace-write", "danger-full-access"},
						"description": "Optional. If omitted, Codex uses local config.",
					},
					"skip_git_repo_check": map[string]any{
						"type":        "boolean",
						"description": "Default: true",
					},
					"json": map[string]any{
						"type":        "boolean",
						"description": "Default: true",
					},
					"wait_ms_before_async": map[string]any{
						"type": "integer",
					},
					"wait_duration_seconds": map[string]any{
						"type": "integer",
					},
					"output_character_count": map[string]any{
						"type": "integer",
					},
				},
				"required":             []any{"op"},
				"additionalProperties": false,
			},
		},
		"list_files": {
			description: listFilesPrompt,
			parameters:  mustToolSchema(bridgetools.NewListFilesTool(nil)),
		},
		"read_file": {
			description: readFilePrompt,
			parameters:  mustToolSchema(bridgetools.NewReadFileTool(nil)),
		},
		"search_files": {
			description: searchFilesPrompt,
			parameters:  mustToolSchema(bridgetools.NewSearchFilesTool(nil)),
		},
		"screen_control": {
			description: "Screen control. Mode 'atomic' (screenshot/OCR/click) or 'agent' (goal-driven execution).",
			parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"mode": map[string]any{
						"type": "string",
						"enum": []any{"atomic"},
					},
					"action": map[string]any{
						"type": "string",
						"enum": []any{"screenshot", "find_text", "find_icon", "click_icon", "mouse_position", "text_input"},
					},
					"params": map[string]any{
						"type": "object",
					},
					"display_id": map[string]any{
						"type":    "integer",
						"minimum": float64(0),
					},
				},
				"required":             []any{"action"},
				"additionalProperties": false,
			},
		},
		"script_exec": {
			description: scriptExecPrompt,
			parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"script": map[string]any{
						"type": "string",
					},
					"timeout_ms": map[string]any{
						"type": "integer",
					},
					"max_memory_mb": map[string]any{
						"type": "integer",
					},
				},
				"required":             []any{"script"},
				"additionalProperties": false,
			},
		},
		"sfind": {
			description: "Manage dynamic skills from SKILL.md. 'search' finds them, 'load' applies them immediately for this session, 'unload' removes them, 'list' shows current state. Use ONLY when visible tools are insufficient.",
			parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{
						"type": "string",
						"enum": []any{"search", "load", "unload", "list"},
					},
					"query": map[string]any{
						"type": "string",
					},
					"skill_names": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
					},
				},
				"required":             []any{"action"},
				"additionalProperties": false,
			},
		},
		"web_search": {
			description: "Search the web for current information.",
			parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"max_results": map[string]any{
						"type":    "integer",
						"minimum": float64(1),
						"maximum": float64(10),
					},
					"provider": map[string]any{
						"type": "string",
						"enum": []any{"tavily", "exa"},
					},
					"query": map[string]any{
						"type": "string",
					},
				},
				"required":             []any{"provider", "query"},
				"additionalProperties": false,
			},
		},
		"write_file": {
			description: writeFilePrompt,
			parameters:  mustToolSchema(bridgetools.NewWriteFileTool(nil)),
		},
	}
}

func mustToolBasePrompt(toolName string) string {
	prompt, ok := bridgeconfig.ToolBasePrompt(toolName)
	if !ok {
		panic("missing tool base prompt: " + toolName)
	}
	return prompt
}

func mustToolSchema(tool bridgetools.Tool) map[string]any {
	raw := tool.Parameters()
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		panic("decode tool schema: " + err.Error())
	}
	return schema
}

func assertSystemPromptToolDefinition(
	t *testing.T,
	items []systemPromptToolDefinitionPayload,
	toolName string,
	wantDescription string,
	wantParameters map[string]any,
) {
	t.Helper()

	for _, item := range items {
		if item.Name != toolName {
			continue
		}
		if item.Description != wantDescription {
			t.Fatalf("unexpected %s description: got %q want %q", toolName, item.Description, wantDescription)
		}
		if !reflect.DeepEqual(item.Parameters, wantParameters) {
			gotParameters, err := json.Marshal(item.Parameters)
			if err != nil {
				t.Fatalf("marshal %s parameters: %v", toolName, err)
			}
			wantParametersJSON, err := json.Marshal(wantParameters)
			if err != nil {
				t.Fatalf("marshal expected %s parameters: %v", toolName, err)
			}
			t.Fatalf("unexpected %s parameters: got %s want %s", toolName, gotParameters, wantParametersJSON)
		}
		return
	}
	t.Fatalf("expected tool_definitions to include %s, got %+v", toolName, items)
}
