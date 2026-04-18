package config

import (
	"fmt"
	"strings"
)

func migrateLegacyToolPromptOverrides(
	fileCfg bridgeFileConfig,
	configPath string,
	env envSnapshot,
) (bridgeFileConfig, error) {
	legacy := cloneStringMap(fileCfg.ToolPromptOverrides)
	if len(legacy) == 0 {
		return fileCfg, nil
	}

	promptsDir, err := resolvePromptsDir(fileCfg, env)
	if err != nil {
		return bridgeFileConfig{}, err
	}
	if err := mergeLegacyToolPromptOverrides(promptsDir, legacy); err != nil {
		return bridgeFileConfig{}, err
	}

	fileCfg.ToolPromptOverrides = nil
	normalized, err := normalizeBridgeFileConfigForWrite(fileCfg)
	if err != nil {
		return bridgeFileConfig{}, err
	}
	if err := writeBridgeFileConfig(configPath, normalized); err != nil {
		return bridgeFileConfig{}, err
	}
	return normalized, nil
}

func mergeLegacyToolPromptOverrides(promptsDir string, legacy map[string]string) error {
	current, err := loadToolPromptOverridesFromFiles(promptsDir)
	if err != nil {
		return err
	}
	defaults := toolBasePrompts()

	for rawName, rawPrompt := range legacy {
		name := strings.TrimSpace(rawName)
		prompt := strings.TrimSpace(rawPrompt)
		if name == "" || prompt == "" {
			continue
		}
		currentPrompt := strings.TrimSpace(current[name])
		if currentPrompt != "" && currentPrompt != strings.TrimSpace(defaults[name]) {
			continue
		}
		if err := writeToolPromptOverrideToFile(promptsDir, name, prompt); err != nil {
			return fmt.Errorf("migrate tool prompt override %s: %w", name, err)
		}
	}
	return nil
}
