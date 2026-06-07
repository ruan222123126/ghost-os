package tools

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writeToolPromptFileIfChanged(root string, name string, prompt string) error {
	current, err := readToolPromptOverrideFile(root, name)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(prompt)
	if current == trimmed {
		return nil
	}
	return writeToolPromptFile(root, name, trimmed)
}

func writeToolPromptFile(root string, name string, prompt string) error {
	path, err := toolPromptFilePath(root, name)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(prompt)), toolPromptFilePerm); err != nil {
		return fmt.Errorf("write tool prompt file %s: %w", path, err)
	}
	return nil
}

func readToolPromptOverrideFile(root string, name string) (string, error) {
	path, err := toolPromptFilePath(root, name)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read tool prompt file %s: %w", path, err)
	}
	return strings.TrimSpace(string(raw)), nil
}

func ensureToolPromptFiles(promptsDir string, defaults map[string]string) (string, error) {
	root, err := resolveToolPromptsDir(promptsDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, toolPromptDirPerm); err != nil {
		return "", fmt.Errorf("create tool prompts directory %s: %w", root, err)
	}
	initialized := toolPromptFilesInitialized(root)
	for _, name := range configuredToolNames() {
		if fileErr := ensureToolPromptFile(root, name, defaults[name], initialized); fileErr != nil {
			return "", fileErr
		}
	}
	if !initialized {
		if err := markToolPromptFilesInitialized(root); err != nil {
			return "", err
		}
	}
	return root, nil
}

func ensureToolPromptFile(root string, name string, defaultPrompt string, initialized bool) error {
	path, err := toolPromptFilePath(root, name)
	if err != nil {
		return err
	}
	content := []byte(strings.TrimSpace(defaultPrompt))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, toolPromptFilePerm)
	if err == nil {
		if _, writeErr := file.Write(content); writeErr != nil {
			file.Close()
			return fmt.Errorf("write tool prompt file %s: %w", path, writeErr)
		}
		return file.Close()
	}
	if errors.Is(err, os.ErrExist) {
		if initialized {
			return nil
		}
		if strings.TrimSpace(defaultPrompt) == "" {
			return nil
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read tool prompt file %s: %w", path, readErr)
		}
		if strings.TrimSpace(string(raw)) != "" {
			return nil
		}
		if writeErr := os.WriteFile(path, content, toolPromptFilePerm); writeErr != nil {
			return fmt.Errorf("write tool prompt file %s: %w", path, writeErr)
		}
		return nil
	}
	return fmt.Errorf("create tool prompt file %s: %w", path, err)
}

func resolveToolPromptsDir(promptsDir string) (string, error) {
	trimmed := strings.TrimSpace(promptsDir)
	if trimmed == "" {
		return "", errors.New("prompts_dir is empty")
	}
	return filepath.Join(filepath.Clean(trimmed), toolPromptDirName), nil
}

func toolPromptFilePath(root string, name string) (string, error) {
	trimmedRoot := strings.TrimSpace(root)
	if trimmedRoot == "" {
		return "", errors.New("tool prompts root is empty")
	}
	trimmedName := strings.TrimSpace(name)
	if !validConfiguredToolNames()[trimmedName] {
		return "", fmt.Errorf("unknown tool: %s", trimmedName)
	}
	return filepath.Join(trimmedRoot, trimmedName+toolPromptFileExt), nil
}

func configuredToolNames() []string {
	names := make([]string, 0, len(configuredToolCatalog))
	for _, raw := range configuredToolCatalog {
		if name := strings.TrimSpace(raw); name != "" {
			names = append(names, name)
		}
	}
	return normalizeConfiguredToolNames(names)
}

func ConfiguredToolNames() []string {
	return configuredToolNames()
}

func toolPromptFilesInitialized(root string) bool {
	path := filepath.Join(strings.TrimSpace(root), toolPromptInitFile)
	_, err := os.Stat(path)
	return err == nil
}

func markToolPromptFilesInitialized(root string) error {
	path := filepath.Join(strings.TrimSpace(root), toolPromptInitFile)
	if err := os.WriteFile(path, []byte("initialized"), toolPromptFilePerm); err != nil {
		return fmt.Errorf("write tool prompt init marker %s: %w", path, err)
	}
	return nil
}
