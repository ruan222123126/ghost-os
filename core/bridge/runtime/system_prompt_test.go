package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

func TestBuildRuntimeSystemPromptAppliesSystemPromptFiles(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	projectRoot := t.TempDir()
	if _, err := bridgeconfig.UpdateSystemPromptFiles(promptsDir, bridgeconfig.SystemPromptUpdateRequest{
		CorePrompt: ptr("core block"),
	}); err != nil {
		t.Fatalf("UpdateSystemPromptFiles: %v", err)
	}
	skillDir := filepath.Join(projectRoot, ".agents", "skills", "release")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", skillDir, err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: release_flow\ndescription: release skill\n---\nRun release flow.\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}

	registry := tools.NewRegistry()
	registry.Register(&catalogMockTool{name: "ask_human"})
	registry.Register(&catalogMockTool{name: "script_exec"})

	prompt, err := buildRuntimeSystemPrompt(bridgeconfig.Config{
		MaxTurns:    3,
		PromptsDir:  promptsDir,
		ProjectRoot: projectRoot,
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Allowlist: []string{"ask_human", "script_exec"},
		},
	}, registry)
	if err != nil {
		t.Fatalf("buildRuntimeSystemPrompt returned error: %v", err)
	}

	for _, snippet := range []string{
		"Role:",
		"Job:",
		"Skills:",
		"Skill Context:",
		"core block",
		"release_flow: release skill",
		"Root: " + projectRoot,
	} {
		if !strings.Contains(prompt, snippet) {
			t.Fatalf("expected prompt to contain %q, got %q", snippet, prompt)
		}
	}
	if strings.Count(prompt, "core block") != 1 {
		t.Fatalf("expected core job override to appear once, got %q", prompt)
	}
	if strings.Contains(prompt, "## Dynamic Tool State") {
		t.Fatalf("expected prompt without legacy dynamic tool section, got %q", prompt)
	}
}

func TestBuildSystemPromptForSessionIncludesLoadedSkillBody(t *testing.T) {
	projectRoot := t.TempDir()
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	skillDir := filepath.Join(projectRoot, ".agents", "skills", "release")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", skillDir, err)
	}
	body := "Release playbook body."
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("---\nname: release_flow\ndescription: release skill\n---\n"+body+"\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}

	sess := session.NewSession("")
	sess.AdvanceToolTurn(3)
	sess.EnsureDynamicSkillLoaded("release_flow", "sfind")

	registry := tools.NewRegistry()
	registry.Register(&catalogMockTool{name: "sfind"})
	prompt, err := buildSystemPromptForSession(
		bridgeconfig.Config{
			MaxTurns:    3,
			PromptsDir:  promptsDir,
			ProjectRoot: projectRoot,
			ToolSearch:  bridgeconfig.ToolSearchConfig{IdleTurns: 3},
		},
		registry,
		sess,
		3,
	)
	if err != nil {
		t.Fatalf("buildSystemPromptForSession returned error: %v", err)
	}
	if !strings.Contains(prompt, "### Skill `release_flow`") {
		t.Fatalf("expected prompt to include loaded skill heading, got %q", prompt)
	}
	if !strings.Contains(prompt, body) {
		t.Fatalf("expected prompt to include loaded skill body, got %q", prompt)
	}
}

func TestBuildRuntimeSystemPromptMemoryModeCreatesDayFileWithoutInjectingPrompt(t *testing.T) {
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
	if strings.Contains(prompt, "Memory:") {
		t.Fatalf("expected prompt to skip memory section, got %q", prompt)
	}

	dayFile, err := memoryDayFilePath(time.Now())
	if err != nil {
		t.Fatalf("memoryDayFilePath: %v", err)
	}
	if _, err := os.Stat(dayFile); err != nil {
		t.Fatalf("expected memory day file to exist: %s err=%v", dayFile, err)
	}
}

func TestBuildRuntimeSystemPromptIgnoresLegacyMemoryPromptCards(t *testing.T) {
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
	if strings.Contains(prompt, "custom memory guidance") {
		t.Fatalf("expected prompt to ignore legacy memory override, got %q", prompt)
	}
	if strings.Contains(prompt, "Memory:") {
		t.Fatalf("expected prompt to skip memory section, got %q", prompt)
	}
}

func TestBuildRuntimeSystemPromptUsesActiveRuleLibraryOverride(t *testing.T) {
	promptsDir := filepath.Join(t.TempDir(), "prompts")
	library := []bridgeconfig.SystemPromptLibraryItem{
		{
			ID:          "rule-card",
			Name:        "Rule",
			InsertPoint: bridgeconfig.SystemPromptInsertPointRule,
			Content:     "You are the overridden bridge rule.",
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
		MaxTurns:   3,
		PromptsDir: promptsDir,
	}, registry)
	if err != nil {
		t.Fatalf("buildRuntimeSystemPrompt returned error: %v", err)
	}
	if !strings.Contains(prompt, "You are the overridden bridge rule.") {
		t.Fatalf("expected prompt to include rule override, got %q", prompt)
	}
	if strings.Contains(prompt, "Ghost-OS bridge agent (digital twin execution layer).") {
		t.Fatalf("expected default rule to be replaced, got %q", prompt)
	}
}

func ptr(value string) *string {
	return &value
}
