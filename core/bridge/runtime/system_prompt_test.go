package runtime

import (
	"path/filepath"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/tools"
)

func TestBuildRuntimeSystemPromptAppliesSystemPromptFiles(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	if _, err := bridgeconfig.UpdateSystemPromptFiles(promptsDir, bridgeconfig.SystemPromptUpdateRequest{
		GlobalTemplate: ptr("BEGIN\n{{base_prompt}}\nEND\n{{core_prompt}}\n{{tool_prompt}}\n{{tool_key_spec}}"),
		CorePrompt:     ptr("core block"),
		ToolPrompt:     ptr("tool block"),
		ToolKeySpec:    ptr("key block"),
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

	for _, snippet := range []string{"BEGIN", "END", "core block", "tool block", "key block", "Project root: /tmp/project"} {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to contain %q, got %q", snippet, prompt)
		}
	}
}

func ptr(value string) *string {
	return &value
}
