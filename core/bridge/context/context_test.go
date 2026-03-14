package context

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
)

type fakeRegistry struct {
	defs []llm.ToolDef
}

func (f *fakeRegistry) ToolDefs() []llm.ToolDef {
	return f.defs
}

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

	pm, err := NewPromptManager(configPath)
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

	pm, err := NewPromptManager(configPath)
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

	_, err := NewPromptManager(configPath)
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
		"os_type":     "darwin",
		"tools_count": "3",
		"max_turns":   "20",
	})

	if !strings.Contains(rendered, "OS: darwin") {
		t.Fatalf("rendered prompt missing os variable: %q", rendered)
	}
	if !strings.Contains(rendered, "Available tools: 3") {
		t.Fatalf("rendered prompt missing tools variable: %q", rendered)
	}
	if !strings.Contains(rendered, "Max turns: 20") {
		t.Fatalf("rendered prompt missing max_turns variable: %q", rendered)
	}
}

func TestPromptTemplatesKeepCompactToolStrategy(t *testing.T) {
	vars := map[string]string{
		"os_type":     "linux",
		"tools_count": "9",
		"max_turns":   "20",
	}
	fromFile, err := NewPromptManager(filepath.Join("..", "prompts.yaml"))
	if err != nil {
		t.Fatalf("NewPromptManager returned error: %v", err)
	}

	prompts := []string{
		fromFile.Render(vars),
		NewPromptManagerWithDefault().Render(vars),
	}
	requiredSnippets := []string{
		"primary workspace tool",
		"read_and_summarize",
		"tools.read_file",
		"feed_manage",
		"rss_fetch",
		"screen_action",
		"## Limits",
		"END_SESSION",
	}

	for _, prompt := range prompts {
		for _, snippet := range requiredSnippets {
			if !strings.Contains(prompt, snippet) {
				t.Fatalf("prompt missing %q: %q", snippet, prompt)
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

func TestNewPromptManagerWithCoreFilesMissingFileFails(t *testing.T) {
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

	_, err := NewPromptManagerWithOptions(PromptLoadOptions{
		ConfigPath: configPath,
		CoreDir:    tempDir,
		CoreFiles:  []string{"missing.txt"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNewPromptManagerWithCoreFilesEmptyFileFails(t *testing.T) {
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
	if err := os.WriteFile(corePath, []byte("   "), 0o644); err != nil {
		t.Fatalf("write core file: %v", err)
	}

	_, err := NewPromptManagerWithOptions(PromptLoadOptions{
		ConfigPath: configPath,
		CoreDir:    tempDir,
		CoreFiles:  []string{"core.txt"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBuilderBuildRequestClonesMessagesAndTools(t *testing.T) {
	params := json.RawMessage(`{"type":"object"}`)
	registry := &fakeRegistry{
		defs: []llm.ToolDef{
			{
				Name:        "list_files",
				Description: "list files",
				Parameters:  params,
			},
		},
	}

	builder := NewBuilder(NewPromptManagerWithDefault(), registry)

	messages := []llm.Message{
		{
			Role: llm.RoleUser,
			Text: "hello",
		},
	}
	req := builder.BuildRequest(messages)

	if len(req.Messages) != 1 {
		t.Fatalf("unexpected message count: got %d want %d", len(req.Messages), 1)
	}
	if len(req.Tools) != 1 {
		t.Fatalf("unexpected tool count: got %d want %d", len(req.Tools), 1)
	}

	messages[0].Text = "changed"
	registry.defs[0].Parameters[0] = '{'

	if req.Messages[0].Text != "hello" {
		t.Fatalf("messages should be cloned: got %q want %q", req.Messages[0].Text, "hello")
	}
	if got := string(req.Tools[0].Parameters); got != `{"type":"object"}` {
		t.Fatalf("tool params should be cloned: got %q want %q", got, `{"type":"object"}`)
	}
}

func TestBuilderBuildRequestNilBuilderDoesNotPanic(t *testing.T) {
	var builder *Builder
	req := builder.BuildRequest([]llm.Message{{Role: llm.RoleUser, Text: "hello"}})
	if len(req.Messages) != 1 {
		t.Fatalf("unexpected message count: got %d want %d", len(req.Messages), 1)
	}
	if len(req.Tools) != 0 {
		t.Fatalf("unexpected tool count: got %d want %d", len(req.Tools), 0)
	}
}
