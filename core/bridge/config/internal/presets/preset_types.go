package presets

import "errors"

const presetFileName = "presets.json"

const PresetFileName = presetFileName

var (
	errPresetInvalid     = errors.New("preset is invalid")
	errPresetIDRequired  = errors.New("preset id is required")
	errPresetNotFound    = errors.New("preset not found")
	errPresetUpdateEmpty = errors.New(
		"at least one of name, tool_allowlist, prompt_refs is required",
	)
)

var (
	ErrPresetInvalid     = errPresetInvalid
	ErrPresetIDRequired  = errPresetIDRequired
	ErrPresetNotFound    = errPresetNotFound
	ErrPresetUpdateEmpty = errPresetUpdateEmpty
)

type PresetPromptRefs struct {
	Rule    string   `json:"rule,omitempty"`
	CoreJob string   `json:"core_job,omitempty"`
	Memory  string   `json:"memory,omitempty"`
	Context []string `json:"context,omitempty"`
}

type Preset struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	ToolAllowlist []string         `json:"tool_allowlist"`
	PromptRefs    PresetPromptRefs `json:"prompt_refs"`
}

type PresetCreateRequest struct {
	Name          string           `json:"name"`
	ToolAllowlist []string         `json:"tool_allowlist"`
	PromptRefs    PresetPromptRefs `json:"prompt_refs"`
	TraceID       string           `json:"trace_id,omitempty"`
}

type PresetUpdateRequest struct {
	Name          *string           `json:"name,omitempty"`
	ToolAllowlist *[]string         `json:"tool_allowlist,omitempty"`
	PromptRefs    *PresetPromptRefs `json:"prompt_refs,omitempty"`
	TraceID       string            `json:"trace_id,omitempty"`
}

func (req PresetUpdateRequest) hasUpdates() bool {
	return req.Name != nil || req.ToolAllowlist != nil || req.PromptRefs != nil
}
