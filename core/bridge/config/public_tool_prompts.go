package config

func LoadToolPromptOverrides(promptsDir string) (map[string]string, error) {
	return loadToolPromptOverridesFromFiles(promptsDir)
}

func ToolBasePrompt(name string) (string, bool) {
	return toolBasePrompt(name)
}

func ToolBasePrompts() map[string]string {
	return toolBasePrompts()
}
