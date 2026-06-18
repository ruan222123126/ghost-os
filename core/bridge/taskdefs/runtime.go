package taskdefs

import (
	"strings"
)

type TaskRuntimeOverrides struct {
	ProviderName      string   `json:"provider_name,omitempty"`
	Model             string   `json:"model,omitempty"`
	SystemPrompt      string   `json:"system_prompt,omitempty"`
	PresetID          string   `json:"preset_id,omitempty"`
	ToolAllowlist     []string `json:"tool_allowlist,omitempty"`
	ToolAllowlistOnly *bool    `json:"tool_allowlist_only,omitempty"`
	MaxTurns          *int     `json:"max_turns,omitempty"`
}

func CloneTaskRuntimeOverrides(input *TaskRuntimeOverrides) *TaskRuntimeOverrides {
	if input == nil {
		return nil
	}
	return &TaskRuntimeOverrides{
		ProviderName:      strings.TrimSpace(input.ProviderName),
		Model:             strings.TrimSpace(input.Model),
		SystemPrompt:      strings.TrimSpace(input.SystemPrompt),
		PresetID:          strings.TrimSpace(input.PresetID),
		ToolAllowlist:     append([]string(nil), input.ToolAllowlist...),
		ToolAllowlistOnly: cloneOptionalBoolPointer(input.ToolAllowlistOnly),
		MaxTurns:          cloneOptionalIntPointer(input.MaxTurns),
	}
}

func cloneOptionalBoolPointer(input *bool) *bool {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func cloneOptionalIntPointer(input *int) *int {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}
