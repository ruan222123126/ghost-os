package runtimeopts

import (
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/taskdefs"
)

type ProviderCatalog struct {
	ActiveProvider string
	Providers      []ProviderRecord
}

type ProviderRecord struct {
	Name   string
	Models []string
}

type OverrideNormalizationOptions struct {
	RequireProviderModelPair bool
	DropMaxTurns             bool
	ValidToolNames           []string
}

func NormalizeTaskOverrides(
	input *taskdefs.TaskRuntimeOverrides,
	options OverrideNormalizationOptions,
) (*taskdefs.TaskRuntimeOverrides, error) {
	if input == nil {
		return nil, nil
	}
	allowlist, _, err := NormalizeToolLists(input.ToolAllowlist, nil, options.ValidToolNames)
	if err != nil {
		return nil, err
	}
	providerName := strings.TrimSpace(input.ProviderName)
	model := strings.TrimSpace(input.Model)
	systemPrompt := strings.TrimSpace(input.SystemPrompt)
	presetID := strings.TrimSpace(input.PresetID)
	toolAllowlistOnly := normalizeBoolPointer(input.ToolAllowlistOnly)
	maxTurns, err := normalizeMaxTurns(input.MaxTurns, options.DropMaxTurns)
	if err != nil {
		return nil, err
	}
	if err := validateProviderModelPair(providerName, model, options.RequireProviderModelPair); err != nil {
		return nil, err
	}
	if overridesEmpty(providerName, model, systemPrompt, presetID, allowlist, toolAllowlistOnly, maxTurns) {
		return nil, nil
	}
	return &taskdefs.TaskRuntimeOverrides{
		ProviderName:      providerName,
		Model:             model,
		SystemPrompt:      systemPrompt,
		PresetID:          presetID,
		ToolAllowlist:     allowlist,
		ToolAllowlistOnly: toolAllowlistOnly,
		MaxTurns:          maxTurns,
	}, nil
}

func ValidateOverridesAgainstCatalog(
	overrides *taskdefs.TaskRuntimeOverrides,
	catalog ProviderCatalog,
	requireProviderModelPair bool,
) error {
	normalized, err := NormalizeTaskOverrides(overrides, OverrideNormalizationOptions{
		RequireProviderModelPair: requireProviderModelPair,
	})
	if err != nil || normalized == nil {
		return err
	}
	if strings.TrimSpace(normalized.ProviderName) == "" && strings.TrimSpace(normalized.Model) == "" {
		return nil
	}
	targetName := strings.TrimSpace(normalized.ProviderName)
	if targetName == "" {
		targetName = strings.TrimSpace(catalog.ActiveProvider)
	}
	record, ok := findProviderRecord(catalog.Providers, targetName)
	if !ok {
		return fmt.Errorf("provider_name %q is not configured", targetName)
	}
	if !modelExists(record.Models, normalized.Model) {
		return fmt.Errorf("model %q is not configured for provider %q", normalized.Model, record.Name)
	}
	return nil
}

func UsesToolScope(overrides *taskdefs.TaskRuntimeOverrides) bool {
	if overrides == nil {
		return false
	}
	if overrides.ToolAllowlistOnly != nil && *overrides.ToolAllowlistOnly {
		return true
	}
	return overrides.ToolAllowlistOnly == nil && len(overrides.ToolAllowlist) > 0
}

func NormalizeRetryPolicy(count int, intervalMS int) (RetryPolicy, error) {
	if count < 0 {
		return RetryPolicy{}, fmt.Errorf("llm_completion_retry_count must be >= 0")
	}
	if intervalMS < 0 {
		return RetryPolicy{}, fmt.Errorf("llm_completion_retry_interval_ms must be >= 0")
	}
	return RetryPolicy{Count: count, IntervalMS: intervalMS}, nil
}

type RetryPolicy struct {
	Count      int
	IntervalMS int
}

func NormalizeToolLists(
	allowlist []string,
	blocklist []string,
	validToolNames []string,
) ([]string, []string, error) {
	normalizedAllowlist := normalizeToolNames(allowlist)
	normalizedBlocklist := normalizeToolNames(blocklist)
	valid := validToolNameSet(validToolNames)
	for _, name := range normalizedAllowlist {
		if len(valid) > 0 && !valid[name] {
			return nil, nil, fmt.Errorf("unknown tool in tool_allowlist: %s", name)
		}
	}

	filteredBlocklist := make([]string, 0, len(normalizedBlocklist))
	for _, name := range normalizedBlocklist {
		if len(valid) > 0 && !valid[name] {
			return nil, nil, fmt.Errorf("unknown tool in tool_blocklist: %s", name)
		}
		filteredBlocklist = append(filteredBlocklist, name)
	}

	allowSet := make(map[string]bool, len(normalizedAllowlist))
	for _, name := range normalizedAllowlist {
		allowSet[name] = true
	}
	for _, name := range filteredBlocklist {
		if allowSet[name] {
			return nil, nil, fmt.Errorf("tool %q cannot appear in both tool_allowlist and tool_blocklist", name)
		}
	}

	return normalizedAllowlist, filteredBlocklist, nil
}

func validateProviderModelPair(providerName string, model string, requirePair bool) error {
	if providerName != "" && model == "" {
		return fmt.Errorf("provider_name requires model")
	}
	if !requirePair {
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

func normalizeMaxTurns(input *int, drop bool) (*int, error) {
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

func normalizeBoolPointer(input *bool) *bool {
	if input == nil || !*input {
		return nil
	}
	value := true
	return &value
}

func overridesEmpty(
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

func findProviderRecord(providers []ProviderRecord, name string) (ProviderRecord, bool) {
	target := strings.TrimSpace(name)
	for _, provider := range providers {
		if strings.EqualFold(strings.TrimSpace(provider.Name), target) {
			return provider, true
		}
	}
	return ProviderRecord{}, false
}

func modelExists(models []string, model string) bool {
	target := strings.TrimSpace(model)
	if target == "" {
		return false
	}
	for _, raw := range models {
		if strings.TrimSpace(raw) == target {
			return true
		}
	}
	return false
}

func normalizeToolNames(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	result := make([]string, 0, len(names))
	seen := make(map[string]bool, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	if len(result) == 0 {
		return nil
	}
	sort.Strings(result)
	return result
}

func validToolNameSet(names []string) map[string]bool {
	if len(names) == 0 {
		return nil
	}
	valid := make(map[string]bool, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name != "" {
			valid[name] = true
		}
	}
	return valid
}
