package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTaskConfigReadsFileBackedTaskSettings(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte(`
tasks_path = "/tmp/ghost-tasks"
task_execution_timeout_ms = 600000
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
	if taskCfg.ExecutionTimeoutMS != 600000 {
		t.Fatalf("unexpected task execution timeout: %#v", taskCfg)
	}
	if len(taskCfg.WorkflowToolAllowlist) != 2 || taskCfg.WorkflowToolAllowlist[0] != "script_exec" || taskCfg.WorkflowToolAllowlist[1] != "web_search" {
		t.Fatalf("unexpected workflow tool allowlist: %#v", taskCfg)
	}
}

func TestLoadTaskConfigReadsExecutionTimeoutFromEnv(t *testing.T) {
	t.Setenv("GHOST_CONFIG_PATH", filepath.Join(t.TempDir(), "missing.toml"))
	t.Setenv("GHOST_TASK_EXECUTION_TIMEOUT_MS", "450000")

	taskCfg, err := LoadTaskConfig()
	if err != nil {
		t.Fatalf("load task config: %v", err)
	}
	if taskCfg.ExecutionTimeoutMS != 450000 {
		t.Fatalf("unexpected execution timeout: got %d want %d", taskCfg.ExecutionTimeoutMS, 450000)
	}
}
