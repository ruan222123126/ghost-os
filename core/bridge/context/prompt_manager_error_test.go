package context

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

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

func TestNewPromptManagerWithResponseRuleFilesEmptyFileFails(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "prompts.yaml")
	content := `version: "1.0"
system:
  default: |
    Rules: {{response_rules}}
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write prompts file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "rules.txt"), []byte("  "), 0o644); err != nil {
		t.Fatalf("write rules file: %v", err)
	}

	_, err := NewPromptManagerWithOptions(PromptLoadOptions{
		ConfigPath:        configPath,
		CoreDir:           tempDir,
		ResponseRuleFiles: []string{"rules.txt"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPromptManagerRenderNilPanics(t *testing.T) {
	assertPanicsWithError(t, errPromptManagerNil, func() {
		var pm *PromptManager
		_ = pm.Render(nil)
	})
}

func TestPromptManagerRenderEmptyTemplatePanics(t *testing.T) {
	assertPanicsWithError(t, errPromptTemplateNil, func() {
		pm := &PromptManager{template: "   "}
		_ = pm.Render(nil)
	})
}

func assertPanicsWithError(t *testing.T, want error, fn func()) {
	t.Helper()

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected panic %v, got nil", want)
		}
		err, ok := recovered.(error)
		if !ok {
			t.Fatalf("unexpected panic type %T", recovered)
		}
		if !errors.Is(err, want) {
			t.Fatalf("unexpected panic error: got %v want %v", err, want)
		}
	}()

	fn()
}
