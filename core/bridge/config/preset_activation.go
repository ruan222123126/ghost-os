package config

import "slices"

func applyPresetToFileConfig(fileCfg bridgeFileConfig, preset Preset) bridgeFileConfig {
	fileCfg.ToolAllowlistOnly = boolPointer(true)
	fileCfg.ToolAllowlist = append([]string(nil), preset.ToolAllowlist...)
	fileCfg.ToolBlocklist = presetToolBlocklist(preset.ToolAllowlist)
	return fileCfg
}

func presetToolBlocklist(allowlist []string) []string {
	allowset := toolNameSetFromSlice(allowlist)
	blocklist := make([]string, 0, len(configuredToolCatalog))

	for _, raw := range configuredToolCatalog {
		if allowset[raw] {
			continue
		}
		blocklist = append(blocklist, raw)
	}
	return normalizeConfiguredToolNames(blocklist)
}

func applyPresetToPromptLibrary(
	promptLibrary []SystemPromptLibraryItem,
	preset Preset,
) ([]SystemPromptLibraryItem, error) {
	nextLibrary := setPresetPromptRefsActive(promptLibrary, preset.PromptRefs)
	nextLibrary = reorderPresetContextCards(nextLibrary, preset.PromptRefs.Context)
	return normalizePromptLibrary(nextLibrary)
}

func setPresetPromptRefsActive(
	promptLibrary []SystemPromptLibraryItem,
	refs PresetPromptRefs,
) []SystemPromptLibraryItem {
	contextSet := toolNameSetFromSlice(refs.Context)
	nextLibrary := make([]SystemPromptLibraryItem, 0, len(promptLibrary))

	for _, item := range promptLibrary {
		nextItem := item
		nextItem.Active = presetPromptRefActive(item, refs, contextSet)
		nextLibrary = append(nextLibrary, nextItem)
	}
	return nextLibrary
}

func presetPromptRefActive(
	item SystemPromptLibraryItem,
	refs PresetPromptRefs,
	contextSet map[string]bool,
) bool {
	switch item.InsertPoint {
	case SystemPromptInsertPointRule:
		return item.ID == refs.Rule
	case SystemPromptInsertPointCoreJob:
		return item.ID == refs.CoreJob
	case SystemPromptInsertPointMemory:
		return item.ID == refs.Memory
	case SystemPromptInsertPointContext:
		return contextSet[item.ID]
	default:
		return false
	}
}

func reorderPresetContextCards(
	promptLibrary []SystemPromptLibraryItem,
	contextRefs []string,
) []SystemPromptLibraryItem {
	contextCards := presetContextCards(promptLibrary, contextRefs)
	if len(contextCards) == 0 {
		return promptLibrary
	}

	nextLibrary := make([]SystemPromptLibraryItem, 0, len(promptLibrary))
	contextIndex := 0
	for _, item := range promptLibrary {
		if item.InsertPoint != SystemPromptInsertPointContext {
			nextLibrary = append(nextLibrary, item)
			continue
		}
		nextLibrary = append(nextLibrary, contextCards[contextIndex])
		contextIndex += 1
	}
	return nextLibrary
}

func presetContextCards(
	promptLibrary []SystemPromptLibraryItem,
	contextRefs []string,
) []SystemPromptLibraryItem {
	contextCards := make([]SystemPromptLibraryItem, 0, len(promptLibrary))
	for _, item := range promptLibrary {
		if item.InsertPoint == SystemPromptInsertPointContext {
			contextCards = append(contextCards, item)
		}
	}
	if len(contextCards) == 0 {
		return nil
	}

	sorted := make([]SystemPromptLibraryItem, 0, len(contextCards))
	seen := make(map[string]bool, len(contextCards))
	for _, refID := range contextRefs {
		index := slices.IndexFunc(contextCards, func(item SystemPromptLibraryItem) bool {
			return item.ID == refID
		})
		if index < 0 {
			continue
		}
		sorted = append(sorted, contextCards[index])
		seen[refID] = true
	}
	for _, item := range contextCards {
		if seen[item.ID] {
			continue
		}
		sorted = append(sorted, item)
	}
	return sorted
}

func boolPointer(value bool) *bool {
	return &value
}
