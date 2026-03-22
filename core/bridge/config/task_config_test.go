package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTaskConfigReadsFileBackedTaskSettings(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte(`
tasks_path = "/tmp/ghost-tasks"
workflow_tool_allowlist = ["script_exec", "web_search"]
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	taskCfg, err := LoadTaskConfig()
	if err != nil {
		t.Fatalf("load task config: %v", err)
	}
	if taskCfg.TasksPath != "/tmp/ghost-tasks" {
		t.Fatalf("unexpected tasks path: %#v", taskCfg)
	}
	if len(taskCfg.WorkflowToolAllowlist) != 2 || taskCfg.WorkflowToolAllowlist[0] != "script_exec" || taskCfg.WorkflowToolAllowlist[1] != "web_search" {
		t.Fatalf("unexpected workflow tool allowlist: %#v", taskCfg)
	}
}

func TestLoadTaskConfigRejectsBlockedWorkflowTools(t *testing.T) {
	t.Setenv("GHOST_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.toml"))
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "ask_human")

	_, err := LoadTaskConfig()
	if err == nil || !strings.Contains(err.Error(), `cannot appear in workflow_tool_allowlist`) {
		t.Fatalf("expected blocked workflow tool error, got %v", err)
	}
}
