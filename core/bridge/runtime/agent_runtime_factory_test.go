package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgentRuntimeFactoryRegistersToolSearchWhenEnabled(t *testing.T) {
	setupRuntimeFactoryTestEnv(t)
	t.Setenv("GHOST_TOOL_SEARCH_ENABLED", "true")
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	if deps.registry.Get("sfind") == nil {
		t.Fatal("expected sfind to be registered when tool search is enabled")
	}
}

func TestAgentRuntimeFactoryLoadsToolPromptOverridesFromFiles(t *testing.T) {
	const (
		testDirPerm  = 0o755
		testFilePerm = 0o600
	)

	tempDir := setupRuntimeFactoryTestEnv(t)
	promptsDir := filepath.Join(tempDir, "prompts")
	toolPromptsDir := filepath.Join(promptsDir, "tools")
	if err := os.MkdirAll(toolPromptsDir, testDirPerm); err != nil {
		t.Fatalf("MkdirAll(%s): %v", toolPromptsDir, err)
	}
	promptPath := filepath.Join(toolPromptsDir, "script_exec.md")
	if err := os.WriteFile(promptPath, []byte("from prompt file"), testFilePerm); err != nil {
		t.Fatalf("WriteFile(%s): %v", promptPath, err)
	}
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	if got := deps.cfg.ToolSelector.PromptOverrides["script_exec"]; got != "from prompt file" {
		t.Fatalf("unexpected script_exec override: got %q want %q", got, "from prompt file")
	}
}
