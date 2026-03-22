package orchestration

import (
	"fmt"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
)

func loadTaskRuntimeConfig(store *ConfigStore) (TaskConfig, error) {
	if store != nil {
		cfg, err := store.Config()
		if err != nil {
			return TaskConfig{}, err
		}
		return cfg.Task, nil
	}
	return bridgeconfig.LoadTaskConfig()
}

func (r taskMutationRunner) validateTaskRuntime(task ScheduledTask) error {
	if normalizeTaskKind(task.TaskKind) != taskKindWorkflow {
		return nil
	}
	cfg, err := loadTaskRuntimeConfig(r.configStore)
	if err != nil {
		return err
	}
	return validateWorkflowTaskRuntime(task.Workflow, cfg)
}

func validateWorkflowTaskRuntime(definition *WorkflowDefinition, cfg TaskConfig) error {
	if definition == nil {
		return invalidTaskConfig("workflow is required")
	}
	allowed := workflowToolAllowlistSet(cfg.WorkflowToolAllowlist)
	for _, node := range definition.Nodes {
		if node.Type != workflowNodeTypeTool {
			continue
		}
		toolName := strings.TrimSpace(node.Tool.ToolName)
		if allowed[toolName] {
			continue
		}
		return invalidTaskConfig(fmt.Sprintf("workflow tool %q is not allowed", toolName))
	}
	return nil
}

func workflowToolAllowlistSet(names []string) map[string]bool {
	if len(names) == 0 {
		return nil
	}
	out := make(map[string]bool, len(names))
	for _, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			out[trimmed] = true
		}
	}
	return out
}
