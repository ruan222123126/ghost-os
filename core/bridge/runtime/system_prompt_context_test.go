package runtime

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/tools"
)

func TestBuildRuntimeSystemPromptIncludesActiveContextCardsInSavedOrder(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
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
			Content:     "first custom context",
			Active:      true,
		},
		{
			ID:          "context-b",
			Name:        "Context B",
			InsertPoint: bridgeconfig.SystemPromptInsertPointContext,
			Content:     "second custom context",
			Active:      true,
		},
	}
	if _, err := bridgeconfig.UpdateSystemPromptFiles(promptsDir, bridgeconfig.SystemPromptUpdateRequest{
		PromptLibrary: &library,
	}); err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}

	registry := tools.NewRegistry()
	registry.Register(&catalogMockTool{name: "ask_human"})

	prompt, err := buildRuntimeSystemPrompt(bridgeconfig.Config{
		MaxTurns:    7,
		PromptsDir:  promptsDir,
		ProjectRoot: "/tmp/context-project",
	}, registry)
	if err != nil {
		t.Fatalf("buildRuntimeSystemPrompt returned error: %v", err)
	}

	contextSection := "Context:\nOS: " + goruntime.GOOS
	for _, snippet := range []string{
		contextSection,
		"Root: /tmp/context-project",
		"Max turns: 7",
		"first custom context",
		"second custom context",
	} {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to include %q, got %q", snippet, prompt)
		}
	}
	firstIndex := strings.Index(prompt, "first custom context")
	secondIndex := strings.Index(prompt, "second custom context")
	if firstIndex < 0 || secondIndex < 0 || firstIndex > secondIndex {
		t.Fatalf("expected active context cards to keep saved order, got %q", prompt)
	}
}

func TestBuildRuntimeSystemPromptFailsWithoutContextPlaceholder(t *testing.T) {
	tempDir := t.TempDir()
	promptsDir := filepath.Join(tempDir, "prompts")
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

	registry := tools.NewRegistry()
	registry.Register(&catalogMockTool{name: "ask_human"})

	_, err := buildRuntimeSystemPrompt(bridgeconfig.Config{
		MaxTurns:    3,
		PromptsPath: configPath,
		PromptsDir:  promptsDir,
	}, registry)
	if err == nil {
		t.Fatal("expected context placeholder error, got nil")
	}
	if !strings.Contains(err.Error(), "{{context}}") {
		t.Fatalf("unexpected error: %v", err)
	}
}
