package tasks

import (
	"fmt"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	bridgeTasks "ghost-os/bridge/tasks"
)

func LoadRuntimeConfig(store bridgeconfig.Store) (bridgeconfig.TaskConfig, error) {
	if store != nil {
		cfg, err := store.Config()
		if err != nil {
			return bridgeconfig.TaskConfig{}, err
		}
		return cfg.Task, nil
	}
	return bridgeconfig.LoadTaskConfig()
}

func ValidateRuntime(task *bridgeTasks.ScheduledTask, store bridgeconfig.Store) error {
	if task == nil {
		return InvalidConfig("task is nil")
	}
	kind := bridgeTasks.NormalizeKind(task.TaskKind)
	if kind == bridgeTasks.KindAgentMessage {
		return validateAgentRuntime(task, store)
	}
	if task.RuntimeOverrides != nil {
		return InvalidConfig(kind + " does not allow runtime_overrides")
	}
	if kind == bridgeTasks.KindWorkflow {
		cfg, err := LoadRuntimeConfig(store)
		if err != nil {
			return err
		}
		if err := ValidateWorkflowRuntime(task.Workflow, cfg); err != nil {
			return err
		}
		return ValidateWorkflowAgentRuntime(task.Workflow, store)
	}
	if kind == bridgeTasks.KindOrchestration {
		return ValidateOrchestrationAgentRuntime(task.Orchestration, store)
	}
	return nil
}

func validateAgentRuntime(task *bridgeTasks.ScheduledTask, store bridgeconfig.Store) error {
	overrides, err := NormalizeTaskRuntimeOverrides(task.RuntimeOverrides)
	if err != nil {
		return InvalidConfig(err.Error())
	}
	if err := ValidateRuntimeOverridesWithStore(overrides, store, false); err != nil {
		return InvalidConfig(err.Error())
	}
	task.RuntimeOverrides = overrides
	return normalizeAgentRelay(task, store)
}

func normalizeAgentRelay(task *bridgeTasks.ScheduledTask, store bridgeconfig.Store) error {
	if task == nil || task.AgentMode != bridgeTasks.AgentModeRelay {
		return nil
	}
	defaults, err := relayDefaults(store)
	if err != nil {
		return err
	}
	relay := bridgeTasks.CloneTaskRelayConfig(task.Relay)
	if relay == nil {
		relay = &bridgeTasks.TaskRelayConfig{}
	}
	if strings.TrimSpace(relay.StopPolicy) == "" {
		relay.StopPolicy = defaults.stopPolicy
	}
	if relay.MaxRounds < 0 {
		return InvalidConfig("relay max_rounds must be >= 0")
	}
	if relay.MaxRounds == 0 {
		relay.MaxRounds = defaults.maxRounds
	}
	if relay.ExecutionTimeoutMS == nil {
		timeout := defaults.executionTimeoutMS
		relay.ExecutionTimeoutMS = &timeout
	}
	task.Relay = relay
	return ValidateRelayConfig(task.Relay)
}

type relayDefaultValues struct {
	stopPolicy         string
	maxRounds          int
	executionTimeoutMS int
}

func relayDefaults(store bridgeconfig.Store) (relayDefaultValues, error) {
	if store == nil {
		return relayDefaultValues{
			stopPolicy:         bridgeconfig.DefaultRelayStopPolicy,
			maxRounds:          bridgeconfig.DefaultRelayMaxRounds,
			executionTimeoutMS: bridgeconfig.DefaultRelayExecutionTimeoutMS,
		}, nil
	}
	cfg, err := store.Config()
	if err != nil {
		return relayDefaultValues{}, err
	}
	return relayDefaultValues{
		stopPolicy:         strings.TrimSpace(cfg.RelayDefaultStopPolicy),
		maxRounds:          cfg.RelayDefaultMaxRounds,
		executionTimeoutMS: cfg.RelayDefaultExecutionTimeoutMS,
	}, nil
}

func ValidateWorkflowRuntime(definition *bridgeTasks.WorkflowDefinition, cfg bridgeconfig.TaskConfig) error {
	if definition == nil {
		return InvalidConfig("workflow is required")
	}
	allowed := workflowToolAllowlistSet(cfg.WorkflowToolAllowlist)
	if len(allowed) == 0 {
		return nil
	}
	for _, node := range definition.Nodes {
		if node.Type != workflowdomain.NodeTypeTool {
			continue
		}
		toolName := strings.TrimSpace(node.Tool.ToolName)
		if allowed[toolName] {
			continue
		}
		return InvalidConfig(fmt.Sprintf("workflow tool %q is not allowed", toolName))
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

func ValidateWorkflowAgentRuntime(definition *bridgeTasks.WorkflowDefinition, store bridgeconfig.Store) error {
	if definition == nil {
		return InvalidConfig("workflow is required")
	}
	for index := range definition.Nodes {
		node := &definition.Nodes[index]
		if node.Type != workflowdomain.NodeTypeAgent || node.Agent == nil {
			continue
		}
		overrides, err := NormalizeWorkflowAgentRuntimeOverrides(node.Agent.RuntimeOverrides)
		if err != nil {
			return InvalidConfig(fmt.Sprintf("workflow agent node %q %v", node.ID, err))
		}
		if err := ValidateRuntimeOverridesWithStore(overrides, store, true); err != nil {
			return InvalidConfig(fmt.Sprintf("workflow agent node %q %v", node.ID, err))
		}
		node.Agent.RuntimeOverrides = overrides
	}
	return nil
}

func ValidateOrchestrationAgentRuntime(definition *bridgeTasks.OrchestrationDefinition, store bridgeconfig.Store) error {
	if definition == nil {
		return InvalidConfig("orchestration is required")
	}
	for index := range definition.Nodes {
		node := &definition.Nodes[index]
		if node.Type != bridgeTasks.OrchestrationNodeTypeAgent || node.Agent == nil {
			continue
		}
		overrides, err := NormalizeOrchestrationAgentRuntimeOverrides(node.Agent.RuntimeOverrides)
		if err != nil {
			return InvalidConfig(fmt.Sprintf("orchestration agent node %q %v", node.ID, err))
		}
		if err := ValidateRuntimeOverridesWithStore(overrides, store, true); err != nil {
			return InvalidConfig(fmt.Sprintf("orchestration agent node %q %v", node.ID, err))
		}
		node.Agent.RuntimeOverrides = overrides
	}
	return nil
}

func ValidateRuntimeOverridesWithStore(
	overrides *bridgeTasks.TaskRuntimeOverrides,
	store bridgeconfig.Store,
	requireProviderModelPair bool,
) error {
	if overrides == nil || store == nil {
		return nil
	}
	if err := validateRuntimePresetWithStore(overrides, store); err != nil {
		return err
	}
	catalog, err := LoadRuntimeProviderCatalog(store)
	if err != nil {
		return err
	}
	return ValidateRuntimeOverridesAgainstCatalog(overrides, catalog, requireProviderModelPair)
}

func validateRuntimePresetWithStore(
	overrides *bridgeTasks.TaskRuntimeOverrides,
	store bridgeconfig.Store,
) error {
	if overrides == nil || store == nil || strings.TrimSpace(overrides.PresetID) == "" {
		return nil
	}
	presets, err := store.Presets()
	if err != nil {
		return err
	}
	if _, ok := bridgeconfig.FindPresetByID(presets, overrides.PresetID); ok {
		return nil
	}
	return fmt.Errorf("preset_id %q is not configured", strings.TrimSpace(overrides.PresetID))
}
