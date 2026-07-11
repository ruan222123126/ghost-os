package taskconfig

import (
	"fmt"

	"ghost-os/bridge/config/internal/storage"
	"ghost-os/bridge/config/internal/tools"
)

type Config struct {
	TasksPath             string
	ExecutionTimeoutMS    int
	WorkflowToolAllowlist []string
}

type Defaults struct {
	TasksPath          string
	ExecutionTimeoutMS int
}

func Resolve(fileCfg storage.FileConfig, env storage.EnvSnapshot, defaults Defaults) (Config, error) {
	allowlist, err := ResolveWorkflowToolAllowlist(fileCfg, env)
	if err != nil {
		return Config{}, err
	}
	executionTimeoutMS, err := ResolveExecutionTimeoutMS(fileCfg, env, defaults.ExecutionTimeoutMS)
	if err != nil {
		return Config{}, err
	}
	return Config{
		TasksPath:             storage.ResolveTasksPath(fileCfg, env, defaults.TasksPath),
		ExecutionTimeoutMS:    executionTimeoutMS,
		WorkflowToolAllowlist: allowlist,
	}, nil
}

func ResolveExecutionTimeoutMS(fileCfg storage.FileConfig, env storage.EnvSnapshot, fallback int) (int, error) {
	return storage.IntOrEnvWithEnv(
		fileCfg.TaskExecutionTimeoutMS,
		"task_execution_timeout_ms",
		env,
		"GHOST_TASK_EXECUTION_TIMEOUT_MS",
		fallback,
	)
}

func ResolveWorkflowToolAllowlist(fileCfg storage.FileConfig, env storage.EnvSnapshot) ([]string, error) {
	return NormalizeWorkflowToolAllowlist(
		tools.ToolNameListOrDefault(fileCfg.WorkflowToolAllowlist, env.DefaultValue("GHOST_WORKFLOW_TOOL_ALLOWLIST", "")),
	)
}

func NormalizeWorkflowToolAllowlist(raw []string) ([]string, error) {
	names := tools.NormalizeConfiguredToolNames(raw)
	valid := tools.ValidConfiguredToolNames()
	denylist := tools.WorkflowToolDenylist()
	for _, name := range names {
		if !valid[name] {
			return nil, fmt.Errorf("unknown tool in workflow_tool_allowlist: %s", name)
		}
		if reason, blocked := denylist[name]; blocked {
			return nil, fmt.Errorf("tool %q cannot appear in workflow_tool_allowlist: %s", name, reason)
		}
	}
	return names, nil
}
