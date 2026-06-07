package taskconfig

import (
	"strings"
	"testing"

	"ghost-os/bridge/config/internal/storage"
)

func TestResolveRejectsBlockedWorkflowTools(t *testing.T) {
	fileCfg := storage.FileConfig{}
	env := storage.EnvSnapshot{"GHOST_WORKFLOW_TOOL_ALLOWLIST": "ask_human"}

	_, err := Resolve(fileCfg, env, Defaults{TasksPath: "~/.ghost-os/tasks", ExecutionTimeoutMS: 300000})
	if err == nil || !strings.Contains(err.Error(), `cannot appear in workflow_tool_allowlist`) {
		t.Fatalf("expected blocked workflow tool error, got %v", err)
	}
}

func TestResolveAcceptsSplitSandboxAtomicTools(t *testing.T) {
	fileCfg := storage.FileConfig{}
	env := storage.EnvSnapshot{"GHOST_WORKFLOW_TOOL_ALLOWLIST": "search_files,bash_exec,write_file"}

	cfg, err := Resolve(fileCfg, env, Defaults{TasksPath: "~/.ghost-os/tasks", ExecutionTimeoutMS: 300000})
	if err != nil {
		t.Fatalf("resolve task config: %v", err)
	}
	if got := strings.Join(cfg.WorkflowToolAllowlist, ","); got != "bash_exec,search_files,write_file" {
		t.Fatalf("unexpected workflow tool allowlist: %q", got)
	}
}

func TestResolveReadsExplicitFileValues(t *testing.T) {
	tasksPath := "/tmp/ghost-tasks"
	timeoutMS := 600000
	fileCfg := storage.FileConfig{
		TasksPath:              &tasksPath,
		TaskExecutionTimeoutMS: &timeoutMS,
		WorkflowToolAllowlist:  []string{"script_exec", "web_search"},
	}

	cfg, err := Resolve(fileCfg, storage.EnvSnapshot{}, Defaults{TasksPath: "~/.ghost-os/tasks", ExecutionTimeoutMS: 300000})
	if err != nil {
		t.Fatalf("resolve task config: %v", err)
	}
	if cfg.TasksPath != tasksPath {
		t.Fatalf("unexpected tasks path: %#v", cfg)
	}
	if cfg.ExecutionTimeoutMS != timeoutMS {
		t.Fatalf("unexpected task execution timeout: %#v", cfg)
	}
	if got := strings.Join(cfg.WorkflowToolAllowlist, ","); got != "script_exec,web_search" {
		t.Fatalf("unexpected workflow tool allowlist: %q", got)
	}
}
