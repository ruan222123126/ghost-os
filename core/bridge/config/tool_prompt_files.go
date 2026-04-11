package config

import "strings"

const (
	toolPromptDirName  = "tools"
	toolPromptFileExt  = ".md"
	toolPromptInitFile = ".initialized"
	toolPromptDirPerm  = 0o755
	toolPromptFilePerm = 0o644
	ghostDirName       = ".ghost"
	ghostOSDirName     = ".ghost-os"
	promptsDirName     = "prompts"
)

func loadToolPromptOverridesFromFiles(promptsDir string) (map[string]string, error) {
	roots, err := ensureToolPromptRoots(promptsDir, toolBasePrompts())
	if err != nil {
		return nil, err
	}
	if err := syncToolPromptRoots(roots); err != nil {
		return nil, err
	}
	return readToolPromptOverridesFromRoot(roots[0])
}

func readToolPromptOverridesFromRoot(root string) (map[string]string, error) {
	names := configuredToolNames()
	out := make(map[string]string, len(names))
	for _, name := range names {
		prompt, readErr := readToolPromptOverrideFile(root, name)
		if readErr != nil {
			return nil, readErr
		}
		if prompt != "" {
			out[name] = prompt
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func writeToolPromptOverrideToFile(promptsDir string, name string, prompt string) error {
	roots, err := ensureToolPromptRoots(promptsDir, toolBasePrompts())
	if err != nil {
		return err
	}

	trimmedPrompt := strings.TrimSpace(prompt)
	for _, root := range roots {
		if err := writeToolPromptFile(root, name, trimmedPrompt); err != nil {
			return err
		}
	}
	return nil
}
