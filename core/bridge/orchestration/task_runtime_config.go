package orchestration

import (
	"fmt"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
)

func loadTaskRuntimeConfig(store bridgeconfig.Store) (bridgeconfig.TaskConfig, error) {
	if store != nil {
		cfg, err := store.Config()
		if err != nil {
			return bridgeconfig.TaskConfig{}, err
		}
		return cfg.Task, nil
	}
	return bridgeconfig.LoadTaskConfig()
}

func (r taskMutationRunner) validateTaskRuntime(task *ScheduledTask) error {
	if task == nil {
		return invalidTaskConfig("task is nil")
	}
	kind := normalizeTaskKind(task.TaskKind)
	if kind == taskKindAgentMessage {
		return validateAgentTaskRuntime(task, r.configStore)
	}
	if task.RuntimeOverrides != nil {
		return invalidTaskConfig(kind + " does not allow runtime_overrides")
	}
	if kind != taskKindWorkflow {
		return nil
	}
	cfg, err := loadTaskRuntimeConfig(r.configStore)
	if err != nil {
		return err
	}
	if err := validateWorkflowTaskRuntime(task.Workflow, cfg); err != nil {
		return err
	}
	return validateWorkflowAgentRuntime(task.Workflow, r.configStore)
}

func validateAgentTaskRuntime(task *ScheduledTask, store bridgeconfig.Store) error {
	overrides, err := normalizeTaskRuntimeOverrides(task.RuntimeOverrides)
	if err != nil {
		return invalidTaskConfig(err.Error())
	}
	if err := validateTaskRuntimeOverridesWithStore(overrides, store, false); err != nil {
		return invalidTaskConfig(err.Error())
	}
	task.RuntimeOverrides = overrides
	return nil
}

func validateWorkflowTaskRuntime(definition *WorkflowDefinition, cfg bridgeconfig.TaskConfig) error {
	if definition == nil {
		return invalidTaskConfig("workflow is required")
	}
	allowed := workflowToolAllowlistSet(cfg.WorkflowToolAllowlist)
	if len(allowed) == 0 {
		return nil
	}
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

func validateWorkflowAgentRuntime(definition *WorkflowDefinition, store bridgeconfig.Store) error {
	if definition == nil {
		return invalidTaskConfig("workflow is required")
	}
	for index := range definition.Nodes {
		node := &definition.Nodes[index]
		if node.Type != workflowNodeTypeAgent || node.Agent == nil {
			continue
		}
		overrides, err := normalizeWorkflowAgentRuntimeOverrides(node.Agent.RuntimeOverrides)
		if err != nil {
			return invalidTaskConfig(fmt.Sprintf("workflow agent node %q %v", node.ID, err))
		}
		if err := validateTaskRuntimeOverridesWithStore(overrides, store, true); err != nil {
			return invalidTaskConfig(fmt.Sprintf("workflow agent node %q %v", node.ID, err))
		}
		node.Agent.RuntimeOverrides = overrides
	}
	return nil
}

func validateTaskRuntimeOverridesWithStore(
	overrides *TaskRuntimeOverrides,
	store bridgeconfig.Store,
	requireProviderModelPair bool,
) error {
	if overrides == nil || store == nil {
		return nil
	}
	catalog, err := loadTaskRuntimeProviderCatalog(store)
	if err != nil {
		return err
	}
	return validateTaskRuntimeOverridesAgainstCatalog(overrides, catalog, requireProviderModelPair)
}
