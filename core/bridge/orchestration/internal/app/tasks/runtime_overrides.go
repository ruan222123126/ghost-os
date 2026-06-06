package tasks

import (
	"fmt"
	"strings"

	apptools "ghost-os/bridge/orchestration/internal/app/tools"
	bridgeTasks "ghost-os/bridge/tasks"
)

type runtimeOverrideNormalizationOptions struct {
	requireProviderModelPair bool
	dropMaxTurns             bool
}

func NormalizeTaskRuntimeOverrides(
	input *bridgeTasks.TaskRuntimeOverrides,
) (*bridgeTasks.TaskRuntimeOverrides, error) {
	return normalizeTaskRuntimeOverridesWithOptions(input, runtimeOverrideNormalizationOptions{})
}

func NormalizeWorkflowAgentRuntimeOverrides(
	input *bridgeTasks.TaskRuntimeOverrides,
) (*bridgeTasks.TaskRuntimeOverrides, error) {
	return normalizeTaskRuntimeOverridesWithOptions(input, runtimeOverrideNormalizationOptions{
		requireProviderModelPair: true,
	})
}

func NormalizeOrchestrationAgentRuntimeOverrides(
	input *bridgeTasks.TaskRuntimeOverrides,
) (*bridgeTasks.TaskRuntimeOverrides, error) {
	return normalizeTaskRuntimeOverridesWithOptions(input, runtimeOverrideNormalizationOptions{
		requireProviderModelPair: true,
		dropMaxTurns:             true,
	})
}

func NormalizeTaskRuntimeOverridesForCatalogValidation(
	input *bridgeTasks.TaskRuntimeOverrides,
	requireProviderModelPair bool,
) (*bridgeTasks.TaskRuntimeOverrides, error) {
	return normalizeTaskRuntimeOverridesWithOptions(input, runtimeOverrideNormalizationOptions{
		requireProviderModelPair: requireProviderModelPair,
	})
}

func normalizeTaskRuntimeOverridesWithOptions(
	input *bridgeTasks.TaskRuntimeOverrides,
	options runtimeOverrideNormalizationOptions,
) (*bridgeTasks.TaskRuntimeOverrides, error) {
	if input == nil {
		return nil, nil
	}
	allowlist, _, err := apptools.NormalizeConfiguredToolLists(input.ToolAllowlist, nil)
	if err != nil {
		return nil, err
	}
	providerName := strings.TrimSpace(input.ProviderName)
	model := strings.TrimSpace(input.Model)
	systemPrompt := strings.TrimSpace(input.SystemPrompt)
	toolAllowlistOnly := normalizeTaskRuntimeBoolPointer(input.ToolAllowlistOnly)
	maxTurns, err := normalizeTaskRuntimeMaxTurns(input.MaxTurns, options.dropMaxTurns)
	if err != nil {
		return nil, err
	}
	presetID := strings.TrimSpace(input.PresetID)
	if err := validateTaskRuntimeProviderModelPair(providerName, model, options); err != nil {
		return nil, err
	}
	if taskRuntimeOverridesEmpty(
		providerName,
		model,
		systemPrompt,
		presetID,
		allowlist,
		toolAllowlistOnly,
		maxTurns,
	) {
		return nil, nil
	}
	return &bridgeTasks.TaskRuntimeOverrides{
		ProviderName:      providerName,
		Model:             model,
		SystemPrompt:      systemPrompt,
		PresetID:          presetID,
		ToolAllowlist:     allowlist,
		ToolAllowlistOnly: toolAllowlistOnly,
		MaxTurns:          maxTurns,
	}, nil
}

func validateTaskRuntimeProviderModelPair(
	providerName string,
	model string,
	options runtimeOverrideNormalizationOptions,
) error {
	if providerName != "" && model == "" {
		return fmt.Errorf("provider_name requires model")
	}
	if !options.requireProviderModelPair {
		return nil
	}
	if providerName == "" && model == "" {
		return nil
	}
	if providerName == "" || model == "" {
		return fmt.Errorf("provider_name and model must be set together")
	}
	return nil
}

func normalizeTaskRuntimeBoolPointer(input *bool) *bool {
	if input == nil || !*input {
		return nil
	}
	value := true
	return &value
}

func normalizeTaskRuntimeMaxTurns(input *int, drop bool) (*int, error) {
	if input == nil {
		return nil, nil
	}
	if drop {
		return nil, nil
	}
	if *input <= 0 {
		return nil, fmt.Errorf("max_turns must be > 0")
	}
	value := *input
	return &value, nil
}

func taskRuntimeOverridesEmpty(
	providerName string,
	model string,
	systemPrompt string,
	presetID string,
	allowlist []string,
	toolAllowlistOnly *bool,
	maxTurns *int,
) bool {
	return providerName == "" &&
		model == "" &&
		systemPrompt == "" &&
		presetID == "" &&
		len(allowlist) == 0 &&
		toolAllowlistOnly == nil &&
		maxTurns == nil
}
