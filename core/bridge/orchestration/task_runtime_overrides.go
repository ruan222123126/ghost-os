package orchestration

import (
	"fmt"
	"strings"
)

type taskRuntimeOverrideNormalizationOptions struct {
	requireProviderModelPair bool
}

func normalizeTaskRuntimeOverrides(input *TaskRuntimeOverrides) (*TaskRuntimeOverrides, error) {
	return normalizeTaskRuntimeOverridesWithOptions(input, taskRuntimeOverrideNormalizationOptions{})
}

func normalizeWorkflowAgentRuntimeOverrides(input *TaskRuntimeOverrides) (*TaskRuntimeOverrides, error) {
	return normalizeTaskRuntimeOverridesWithOptions(input, taskRuntimeOverrideNormalizationOptions{
		requireProviderModelPair: true,
	})
}

func normalizeTaskRuntimeOverridesWithOptions(
	input *TaskRuntimeOverrides,
	options taskRuntimeOverrideNormalizationOptions,
) (*TaskRuntimeOverrides, error) {
	if input == nil {
		return nil, nil
	}
	allowlist, _, err := normalizeConfiguredToolLists(input.ToolAllowlist, nil)
	if err != nil {
		return nil, err
	}
	providerName := strings.TrimSpace(input.ProviderName)
	model := strings.TrimSpace(input.Model)
	systemPrompt := strings.TrimSpace(input.SystemPrompt)
	toolAllowlistOnly := normalizeTaskRuntimeBoolPointer(input.ToolAllowlistOnly)
	maxTurns, err := normalizeTaskRuntimeMaxTurns(input.MaxTurns)
	if err != nil {
		return nil, err
	}
	if err := validateTaskRuntimeProviderModelPair(providerName, model, options); err != nil {
		return nil, err
	}
	if taskRuntimeOverridesEmpty(providerName, model, systemPrompt, allowlist, toolAllowlistOnly, maxTurns) {
		return nil, nil
	}
	return &TaskRuntimeOverrides{
		ProviderName:      providerName,
		Model:             model,
		SystemPrompt:      systemPrompt,
		ToolAllowlist:     allowlist,
		ToolAllowlistOnly: toolAllowlistOnly,
		MaxTurns:          maxTurns,
	}, nil
}

func validateTaskRuntimeProviderModelPair(
	providerName string,
	model string,
	options taskRuntimeOverrideNormalizationOptions,
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

func normalizeTaskRuntimeMaxTurns(input *int) (*int, error) {
	if input == nil {
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
	allowlist []string,
	toolAllowlistOnly *bool,
	maxTurns *int,
) bool {
	return providerName == "" &&
		model == "" &&
		systemPrompt == "" &&
		len(allowlist) == 0 &&
		toolAllowlistOnly == nil &&
		maxTurns == nil
}
