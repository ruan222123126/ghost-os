package context

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	template := "Hello {{name}}, missing={{missing}}!"
	got := RenderTemplate(template, map[string]string{
		"name": "Ghost",
	})

	if got != "Hello Ghost, missing={{missing}}!" {
		t.Fatalf("unexpected render result: got %q", got)
	}
}

func TestNewPromptManagerFromFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "prompts.yaml")
	content := `version: "1.0"
system:
  default: |
    OS={{os_type}}
    TOOLS={{tools_count}}
`

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write prompts file: %v", err)
	}

	pm, err := NewPromptManagerWithOptions(PromptLoadOptions{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("NewPromptManager returned error: %v", err)
	}

	rendered := pm.Render(map[string]string{
		"os_type":     "linux",
		"tools_count": "2",
	})

	if got, want := rendered, "OS=linux\nTOOLS=2"; got != want {
		t.Fatalf("unexpected rendered prompt: got %q want %q", got, want)
	}
}

func TestNewPromptManagerSupportsFoldedStyle(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "prompts.yaml")
	content := `version: "1.0"
system:
  default: >-
    OS={{os_type}}
    TOOLS={{tools_count}}
`

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write prompts file: %v", err)
	}

	pm, err := NewPromptManagerWithOptions(PromptLoadOptions{ConfigPath: configPath})
	if err != nil {
		t.Fatalf("NewPromptManager returned error: %v", err)
	}

	rendered := pm.Render(map[string]string{
		"os_type":     "linux",
		"tools_count": "2",
	})

	if got, want := rendered, "OS=linux TOOLS=2"; got != want {
		t.Fatalf("unexpected rendered prompt: got %q want %q", got, want)
	}
}

func TestNewPromptManagerWithoutSystemDefaultFails(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "prompts.yaml")
	content := `version: "1.0"
system:
  other: "not-used"
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write prompts file: %v", err)
	}

	_, err := NewPromptManagerWithOptions(PromptLoadOptions{ConfigPath: configPath})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "system.default") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewPromptManagerWithDefault(t *testing.T) {
	pm := NewPromptManagerWithDefault()
	rendered := pm.Render(map[string]string{
		"os_type":      "darwin",
		"max_turns":    "20",
		"project_root": "/tmp/ghost-os",
		"context":      "OS: darwin | Root: /tmp/ghost-os | Max turns: 20",
	})

	for _, snippet := range []string{
		"Role:",
		"Job:",
		"Context:\nOS: darwin | Root: /tmp/ghost-os | Max turns: 20",
	} {
		if !strings.Contains(rendered, snippet) {
			t.Fatalf("rendered prompt missing %q: %q", snippet, rendered)
		}
	}
}

func TestPromptTemplatesKeepCompactToolStrategy(t *testing.T) {
	vars := map[string]string{
		"os_type":      "linux",
		"max_turns":    "20",
		"project_root": "/tmp/project",
		"context":      "OS: linux | Root: /tmp/project | Max turns: 20",
	}
	fromFile, err := NewPromptManagerWithOptions(PromptLoadOptions{
		ConfigPath: filepath.Join("..", "prompts.yaml"),
	})
	if err != nil {
		t.Fatalf("NewPromptManager returned error: %v", err)
	}

	prompts := []string{
		fromFile.Render(vars),
		NewPromptManagerWithDefault().Render(vars),
	}
	for _, prompt := range prompts {
		for _, snippet := range []string{
			"Role:",
			"Job:",
			"Skills:",
			"Skill Context:",
			"Context:\nOS: linux | Root: /tmp/project | Max turns: 20",
			"- No visible skills available.",
			defaultSkillContext,
		} {
			if !strings.Contains(prompt, snippet) {
				t.Fatalf("prompt missing %q: %q", snippet, prompt)
			}
		}
		for _, snippet := range []string{
			"## Dynamic Tool State",
			"## Dynamic Skill Context",
			defaultDynamicState,
			"## Runtime Constraints",
			"## Response Rules",
			"END_SESSION",
			"Available tools:",
			"Tool list:",
			"screen_action.click_text",
			"tools.read_file reads at most 200 lines",
			"## Tool Guidance",
			"{{dynamic_tool_state}}",
			"{{dynamic_skill_context}}",
			"## Operating Context",
		} {
			if strings.Contains(prompt, snippet) {
				t.Fatalf("prompt should not include %q: %q", snippet, prompt)
			}
		}
	}
}

func TestNewPromptManagerWithCoreFilesOverridesCoreJob(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "prompts.yaml")
	content := `version: "1.0"
system:
  default: |
    Core: {{core_job}}
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write prompts file: %v", err)
	}
	corePath := filepath.Join(tempDir, "core.txt")
	if err := os.WriteFile(corePath, []byte("from file"), 0o644); err != nil {
		t.Fatalf("write core file: %v", err)
	}

	pm, err := NewPromptManagerWithOptions(PromptLoadOptions{
		ConfigPath: configPath,
		CoreDir:    tempDir,
		CoreFiles:  []string{"core.txt"},
	})
	if err != nil {
		t.Fatalf("NewPromptManagerWithOptions returned error: %v", err)
	}
	rendered := pm.Render(nil)
	if !strings.Contains(rendered, "Core: from file") {
		t.Fatalf("unexpected rendered prompt: %q", rendered)
	}
}

func TestNewPromptManagerWithSectionFilesOverridesPromptSections(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "prompts.yaml")
	content := `version: "1.0"
system:
  default: |
    Core: {{core_job}}
    Runtime: {{runtime_constraints}}
    Rules: {{response_rules}}
  core_job: |
    inline core
  runtime_constraints: |
    inline runtime
  response_rules: |
    inline rules
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write prompts file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "runtime.txt"), []byte("from runtime file"), 0o644); err != nil {
		t.Fatalf("write runtime file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "rules.txt"), []byte("from rules file"), 0o644); err != nil {
		t.Fatalf("write rules file: %v", err)
	}

	pm, err := NewPromptManagerWithOptions(PromptLoadOptions{
		ConfigPath:             configPath,
		CoreDir:                tempDir,
		RuntimeConstraintFiles: []string{"runtime.txt"},
		ResponseRuleFiles:      []string{"rules.txt"},
	})
	if err != nil {
		t.Fatalf("NewPromptManagerWithOptions returned error: %v", err)
	}

	rendered := pm.Render(nil)
	for _, snippet := range []string{"Core: inline core", "Runtime: from runtime file", "Rules: from rules file"} {
		if !strings.Contains(rendered, snippet) {
			t.Fatalf("unexpected rendered prompt, missing %q: %q", snippet, rendered)
		}
	}
}
