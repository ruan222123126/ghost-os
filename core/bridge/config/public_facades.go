package config

import (
	presetstore "ghost-os/bridge/config/internal/presets"
	promptstore "ghost-os/bridge/config/internal/prompts"
	configtools "ghost-os/bridge/config/internal/tools"
)

const (
	systemPromptDirName          = promptstore.SystemPromptDirName
	systemPromptFileExt          = promptstore.SystemPromptFileExt
	systemPromptJSONExt          = promptstore.SystemPromptJSONExt
	systemPromptInitFile         = promptstore.SystemPromptInitFile
	systemPromptDirPerm          = promptstore.SystemPromptDirPerm
	systemPromptFilePerm         = promptstore.SystemPromptFilePerm
	systemPromptCorePromptKey    = promptstore.SystemPromptCorePromptKey
	systemPromptPromptLibraryKey = promptstore.SystemPromptPromptLibraryKey
	presetFileName               = presetstore.PresetFileName
	configToolNameAskHuman       = configtools.ConfigToolNameAskHuman
	configToolNameSFind          = configtools.ConfigToolNameSFind
	toolPromptDirName            = configtools.ToolPromptDirName
	toolPromptFileExt            = configtools.ToolPromptFileExt
	toolPromptInitFile           = configtools.ToolPromptInitFile
	toolPromptDirPerm            = configtools.ToolPromptDirPerm
	toolPromptFilePerm           = configtools.ToolPromptFilePerm
)

const (
	SystemPromptInsertPointRule    SystemPromptInsertPoint = promptstore.SystemPromptInsertPointRule
	SystemPromptInsertPointCoreJob SystemPromptInsertPoint = promptstore.SystemPromptInsertPointCoreJob
	SystemPromptInsertPointMemory  SystemPromptInsertPoint = promptstore.SystemPromptInsertPointMemory
	SystemPromptInsertPointContext SystemPromptInsertPoint = promptstore.SystemPromptInsertPointContext
)

var (
	errSystemPromptUpdateEmpty    = promptstore.ErrSystemPromptUpdateEmpty
	errSystemPromptUpdateConflict = promptstore.ErrSystemPromptUpdateConflict
	errSystemPromptLibraryInvalid = promptstore.ErrSystemPromptLibraryInvalid
	errPresetInvalid              = presetstore.ErrPresetInvalid
	errPresetIDRequired           = presetstore.ErrPresetIDRequired
	errPresetNotFound             = presetstore.ErrPresetNotFound
	errPresetUpdateEmpty          = presetstore.ErrPresetUpdateEmpty
	configuredToolCatalog         = configtools.ConfiguredToolCatalog()
	toolPromptDefaults            = configtools.ToolPromptDefaults()
)

type SystemPromptInsertPoint = promptstore.SystemPromptInsertPoint
type SystemPromptLibraryItem = promptstore.SystemPromptLibraryItem
type SystemPromptFiles = promptstore.SystemPromptFiles
type SystemPromptUpdateRequest = promptstore.SystemPromptUpdateRequest

type PresetPromptRefs = presetstore.PresetPromptRefs
type Preset = presetstore.Preset
type PresetCreateRequest = presetstore.PresetCreateRequest
type PresetUpdateRequest = presetstore.PresetUpdateRequest

func LoadSystemPromptFiles(promptsDir string) (SystemPromptFiles, error) {
	return promptstore.LoadSystemPromptFiles(promptsDir)
}

func UpdateSystemPromptFiles(
	promptsDir string,
	req SystemPromptUpdateRequest,
) (SystemPromptFiles, error) {
	return promptstore.UpdateSystemPromptFiles(promptsDir, req)
}

func LoadPresets(promptsDir string) ([]Preset, error) {
	return presetstore.LoadPresets(promptsDir)
}

func CreatePreset(promptsDir string, req PresetCreateRequest) (Preset, error) {
	return presetstore.CreatePreset(promptsDir, req)
}

func UpdatePreset(promptsDir string, presetID string, req PresetUpdateRequest) (Preset, error) {
	return presetstore.UpdatePreset(promptsDir, presetID, req)
}

func DeletePreset(promptsDir string, presetID string) (Preset, error) {
	return presetstore.DeletePreset(promptsDir, presetID)
}

func FindPresetByID(presets []Preset, presetID string) (Preset, bool) {
	return presetstore.FindPresetByID(presets, presetID)
}

func LoadToolPromptOverrides(promptsDir string) (map[string]string, error) {
	return configtools.LoadToolPromptOverrides(promptsDir)
}

func ToolBasePrompt(name string) (string, bool) {
	return configtools.ToolBasePrompt(name)
}

func ToolBasePrompts() map[string]string {
	return configtools.ToolBasePrompts()
}

func defaultSystemPromptFileValues() map[string]string {
	return promptstore.DefaultSystemPromptFileValues()
}

func systemPromptFileKeys() []string {
	return promptstore.SystemPromptFileKeys()
}

func systemPromptFilePath(root string, key string) (string, error) {
	return promptstore.SystemPromptFilePath(root, key)
}

func normalizePromptLibrary(library []SystemPromptLibraryItem) ([]SystemPromptLibraryItem, error) {
	return promptstore.NormalizePromptLibrary(library)
}

func compileCorePromptFromLibrary(library []SystemPromptLibraryItem) string {
	return promptstore.CompileCorePromptFromLibrary(library)
}

func resetSystemPromptFilesCacheForTests() {
	promptstore.ResetSystemPromptFilesCacheForTests()
}

func toolNameListOrEnvWithEnv(raw []string, env envSnapshot, envName string) []string {
	return configtools.ToolNameListOrDefault(raw, env.DefaultValue(envName, ""))
}

func normalizeConfiguredToolLists(allowlist []string, blocklist []string) ([]string, []string, error) {
	return configtools.NormalizeConfiguredToolLists(allowlist, blocklist)
}

func normalizeConfiguredToolNames(names []string) []string {
	return configtools.NormalizeConfiguredToolNames(names)
}

func validConfiguredToolNames() map[string]bool {
	return configtools.ValidConfiguredToolNames()
}

func normalizeToolNames(names []string) []string {
	return configtools.NormalizeToolNames(names)
}

func configuredToolNames() []string {
	return configtools.ConfiguredToolNames()
}

func toolNameSetFromSlice(raw []string) map[string]bool {
	return configtools.NameSetFromSlice(raw)
}

func loadToolPromptOverridesFromFiles(promptsDir string) (map[string]string, error) {
	return configtools.LoadToolPromptOverrides(promptsDir)
}

func writeToolPromptOverrideToFile(promptsDir string, name string, prompt string) error {
	return configtools.WriteToolPromptOverride(promptsDir, name, prompt)
}

func toolBasePrompt(name string) (string, bool) {
	return configtools.ToolBasePrompt(name)
}

func toolBasePrompts() map[string]string {
	return configtools.ToolBasePrompts()
}
