package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/tools"
)

func TestBuildRuntimeSystemPromptAppliesSystemPromptFiles(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	if _, err := bridgeconfig.UpdateSystemPromptFiles(promptsDir, bridgeconfig.SystemPromptUpdateRequest{
		CorePrompt: ptr("core block"),
	}); err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}

	registry := tools.NewRegistry()
	registry.Register(&catalogMockTool{name: "ask_human"})
	registry.Register(&catalogMockTool{name: "script_exec"})

	prompt, err := buildRuntimeSystemPrompt(bridgeconfig.Config{
		MaxTurns:    3,
		PromptsDir:  promptsDir,
		ProjectRoot: "/tmp/project",
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Allowlist: []string{"ask_human", "script_exec"},
		},
	}, registry)
	if err != nil {
		t.Fatalf("buildRuntimeSystemPrompt returned error: %v", err)
	}

	for _, snippet := range []string{"core block", "Project root: /tmp/project"} {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to contain %q, got %q", snippet, prompt)
		}
	}
	if strings.Count(prompt, "core block") != 1 {
		t.Fatalf("expected core job override to appear once, got %q", prompt)
	}
	if strings.Contains(prompt, "## Memory") {
		t.Fatalf("expected prompt without memory section when memory mode is disabled, got %q", prompt)
	}
}

func TestBuildRuntimeSystemPromptMemoryModeUsesDefaultPromptAndCreatesDayFile(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	promptsDir := filepath.Join(t.TempDir(), "prompts")

	registry := tools.NewRegistry()
	registry.Register(&catalogMockTool{name: "ask_human"})

	prompt, err := buildRuntimeSystemPrompt(bridgeconfig.Config{
		MaxTurns:          3,
		PromptsDir:        promptsDir,
		MemoryModeEnabled: true,
	}, registry)
	if err != nil {
		t.Fatalf("buildRuntimeSystemPrompt returned error: %v", err)
	}
	if !strings.Contains(prompt, "## Memory") {
		t.Fatalf("expected prompt to include memory section, got %q", prompt)
	}
	if !strings.Contains(prompt, "After completing one meaningful non-greeting task") {
		t.Fatalf("expected prompt to include default memory instructions, got %q", prompt)
	}

	dayFile, err := memoryDayFilePath(time.Now())
	if err != nil {
		t.Fatalf("memoryDayFilePath: %v", err)
	}
	if _, err := os.Stat(dayFile); err != nil {
		t.Fatalf("expected memory day file to exist: %s err=%v", dayFile, err)
	}
}

func TestBuildRuntimeSystemPromptMemoryModeUsesActiveLibraryOverride(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	library := []bridgeconfig.SystemPromptLibraryItem{
		{
			ID:          "memory-card",
			Name:        "Memory",
			InsertPoint: bridgeconfig.SystemPromptInsertPointMemory,
			Content:     "custom memory guidance",
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
		MaxTurns:          3,
		PromptsDir:        promptsDir,
		MemoryModeEnabled: true,
	}, registry)
	if err != nil {
		t.Fatalf("buildRuntimeSystemPrompt returned error: %v", err)
	}
	if !strings.Contains(prompt, "custom memory guidance") {
		t.Fatalf("expected prompt to include memory override, got %q", prompt)
	}
	if strings.Contains(prompt, "After completing one meaningful non-greeting task") {
		t.Fatalf("expected memory override to replace default memory instructions, got %q", prompt)
	}
}

func ptr(value string) *string {
	return &value
}
