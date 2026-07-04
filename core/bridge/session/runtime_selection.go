package session

import "strings"

const (
	RuntimeSelectionGhost = "ghost"
	RuntimeSelectionCodex = "codex"

	RuntimeSelectionModeDefault = "default"
	RuntimeSelectionModePlan    = "plan"
)

type RuntimeSelection struct {
	Runtime      string `json:"runtime"`
	Provider     string `json:"provider,omitempty"`
	ProviderType string `json:"provider_type,omitempty"`
	Model        string `json:"model,omitempty"`
	Mode         string `json:"mode,omitempty"`
}

func NormalizeRuntimeSelection(input RuntimeSelection) (RuntimeSelection, bool) {
	selection := RuntimeSelection{
		Runtime:      strings.ToLower(strings.TrimSpace(input.Runtime)),
		Provider:     strings.TrimSpace(input.Provider),
		ProviderType: strings.ToLower(strings.TrimSpace(input.ProviderType)),
		Model:        strings.TrimSpace(input.Model),
		Mode:         strings.ToLower(strings.TrimSpace(input.Mode)),
	}
	if selection.Runtime == "" {
		return RuntimeSelection{}, false
	}
	if selection.Mode == "" {
		selection.Mode = RuntimeSelectionModeDefault
	}
	if selection.Provider == "" {
		selection.Provider = selection.ProviderType
	}
	if selection.ProviderType == "" && selection.Runtime == RuntimeSelectionCodex {
		selection.ProviderType = RuntimeSelectionCodex
	}
	return selection, true
}

func CloneRuntimeSelection(input *RuntimeSelection) *RuntimeSelection {
	if input == nil {
		return nil
	}
	selection, ok := NormalizeRuntimeSelection(*input)
	if !ok {
		return nil
	}
	return &selection
}

func (s *Session) SetLastRuntimeSelection(input RuntimeSelection) {
	if s == nil {
		debugNilReceiver("SetLastRuntimeSelection")
		return
	}
	selection, ok := NormalizeRuntimeSelection(input)
	if !ok {
		return
	}
	s.LastRuntimeSelection = &selection
}
