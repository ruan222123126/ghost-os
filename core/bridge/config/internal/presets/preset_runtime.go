package presets

import "strings"

func FindPresetByID(presets []Preset, presetID string) (Preset, bool) {
	target := strings.TrimSpace(presetID)
	for _, preset := range presets {
		if strings.TrimSpace(preset.ID) == target {
			return preset, true
		}
	}
	return Preset{}, false
}
