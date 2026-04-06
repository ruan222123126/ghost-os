package config

import (
	"fmt"
)

func buildTaskConfig(fileCfg bridgeFileConfig, env envSnapshot) (TaskConfig, error) {
	allowlist, err := resolveWorkflowToolAllowlist(fileCfg, env)
	if err != nil {
		return TaskConfig{}, err
	}
	return TaskConfig{
		TasksPath:             resolveTasksPath(fileCfg, env),
		WorkflowToolAllowlist: allowlist,
	}, nil
}

func resolveWorkflowToolAllowlist(fileCfg bridgeFileConfig, env envSnapshot) ([]string, error) {
	return normalizeWorkflowToolAllowlist(
		toolNameListOrEnvWithEnv(fileCfg.WorkflowToolAllowlist, env, "GHOST_WORKFLOW_TOOL_ALLOWLIST"),
	)
}

func normalizeWorkflowToolAllowlist(raw []string) ([]string, error) {
	names := normalizeConfiguredToolNames(raw)
	valid := validConfiguredToolNames()
	for _, name := range names {
		if !valid[name] {
			return nil, fmt.Errorf("unknown tool in workflow_tool_allowlist: %s", name)
		}
		if reason, blocked := workflowToolDenylist[name]; blocked {
			return nil, fmt.Errorf("tool %q cannot appear in workflow_tool_allowlist: %s", name, reason)
		}
	}
	return names, nil
}
