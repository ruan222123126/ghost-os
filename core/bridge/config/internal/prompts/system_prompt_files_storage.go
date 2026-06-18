package prompts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func readSystemPromptFilesFromRoot(root string) (SystemPromptFiles, error) {
	files := SystemPromptFiles{}
	for _, key := range systemPromptFileKeys() {
		content, err := readSystemPromptFile(root, key)
		if err != nil {
			return SystemPromptFiles{}, err
		}
		if err := setSystemPromptFileValue(&files, key, content); err != nil {
			return SystemPromptFiles{}, err
		}
	}
	return files, nil
}

func writeSystemPromptFilesToRoots(roots []string, files SystemPromptFiles) error {
	for _, root := range roots {
		if err := writeSystemPromptFilesToRoot(root, files); err != nil {
			return err
		}
	}
	return nil
}

func writeSystemPromptFilesToRoot(root string, files SystemPromptFiles) error {
	for _, key := range systemPromptFileKeys() {
		content, err := systemPromptFileValue(files, key)
		if err != nil {
			return err
		}
		if err := writeSystemPromptFileIfChanged(root, key, content); err != nil {
			return err
		}
	}
	return nil
}

func writeSystemPromptFileIfChanged(root string, key string, content string) error {
	current, err := readSystemPromptFile(root, key)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(content)
	if current == trimmed {
		return nil
	}
	return writeSystemPromptFile(root, key, trimmed)
}

func writeSystemPromptFile(root string, key string, content string) error {
	path, err := systemPromptFilePath(root, key)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), systemPromptFilePerm); err != nil {
		return fmt.Errorf("write system prompt file %s: %w", path, err)
	}
	return nil
}

func readSystemPromptFile(root string, key string) (string, error) {
	path, err := systemPromptFilePath(root, key)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read system prompt file %s: %w", path, err)
	}
	return strings.TrimSpace(string(raw)), nil
}

func systemPromptFilePath(root string, key string) (string, error) {
	trimmedRoot := strings.TrimSpace(root)
	if trimmedRoot == "" {
		return "", errors.New("system prompts root is empty")
	}

	fileName, ok := systemPromptFileName(key)
	if !ok {
		return "", fmt.Errorf("unknown system prompt key: %s", key)
	}
	return filepath.Join(trimmedRoot, fileName), nil
}

func systemPromptFileName(key string) (string, bool) {
	switch key {
	case systemPromptCorePromptKey:
		return key + systemPromptFileExt, true
	case systemPromptPromptLibraryKey:
		return key + systemPromptJSONExt, true
	default:
		return "", false
	}
}
