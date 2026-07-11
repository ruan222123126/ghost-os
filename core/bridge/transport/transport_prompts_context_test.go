package transport

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
)

func TestHandleSystemPromptsRendersActiveContextCardsInSavedOrder(t *testing.T) {
	handler := newTestHandler(t, nil)

	resp := serveRequest(
		handler,
		http.MethodPatch,
		"/api/prompts/system",
		`{"prompt_library":[{"id":"core-card","name":"Core","insert_point":"core_job","content":"core guidance","active":true},{"id":"context-a","name":"Context A","insert_point":"context","content":"first custom context","active":true},{"id":"context-b","name":"Context B","insert_point":"context","content":"second custom context","active":true}]}`,
		nil,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected PATCH status: got %d body=%s", resp.Code, resp.Body.String())
	}

	payload := decodeSystemPromptPayload(t, resp)
	for _, snippet := range []string{
		"Context:",
		"first custom context",
		"second custom context",
	} {
		if !strings.Contains(payload.RenderedPrompt, snippet) {
			t.Fatalf("expected rendered prompt to include %q, got %q", snippet, payload.RenderedPrompt)
		}
	}
	firstIndex := strings.Index(payload.RenderedPrompt, "first custom context")
	secondIndex := strings.Index(payload.RenderedPrompt, "second custom context")
	if firstIndex < 0 || secondIndex < 0 || firstIndex > secondIndex {
		t.Fatalf("expected active context cards to keep saved order, got %q", payload.RenderedPrompt)
	}
}

func TestHandleSystemPromptsFailsWhenTemplateMissingContextPlaceholder(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "legacy-prompts.yaml")
	content := `version: "1.0"
system:
  default: |
    Role: {{rule}}

    Job:
    {{core_job}}

    Context: OS: {{os_type}} | Root: {{project_root}} | Max turns: {{max_turns}}

  rule: |
    Legacy rule.

  core_job: |
    Legacy core.
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", configPath, err)
	}
	t.Setenv("GHOST_PROMPTS_PATH", configPath)

	handler := newTestHandler(t, nil)
	promptsDir := os.Getenv("GHOST_PROMPTS_DIR")
	library := []bridgeconfig.SystemPromptLibraryItem{
		{
			ID:          "core-card",
			Name:        "Core",
			InsertPoint: bridgeconfig.SystemPromptInsertPointCoreJob,
			Content:     "core guidance",
			Active:      true,
		},
		{
			ID:          "context-a",
			Name:        "Context A",
			InsertPoint: bridgeconfig.SystemPromptInsertPointContext,
			Content:     "context alpha",
			Active:      true,
		},
	}
	if _, err := bridgeconfig.UpdateSystemPromptFiles(promptsDir, bridgeconfig.SystemPromptUpdateRequest{
		PromptLibrary: &library,
	}); err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}

	resp := serveRequest(handler, http.MethodGet, "/api/prompts/system", "", nil)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected GET status: got %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "{{context}}") {
		t.Fatalf("expected error body to mention missing context placeholder, got %s", resp.Body.String())
	}
}
