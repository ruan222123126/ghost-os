package prompts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ensureSystemPromptRoots(promptsDir string, defaults SystemPromptFiles) ([]string, error) {
	dirs, err := resolveSystemPromptDirs(promptsDir)
	if err != nil {
		return nil, err
	}

	roots := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		root, ensureErr := ensureSystemPromptFiles(dir, defaults)
		if ensureErr != nil {
			return nil, ensureErr
		}
		roots = append(roots, root)
	}
	return roots, nil
}

func resolveSystemPromptDirs(promptsDir string) ([]string, error) {
	trimmed := strings.TrimSpace(promptsDir)
	if trimmed == "" {
		return nil, errors.New("prompts_dir is empty")
	}
	return []string{filepath.Clean(trimmed)}, nil
}

func ResolveSystemPromptDirs(promptsDir string) ([]string, error) {
	return resolveSystemPromptDirs(promptsDir)
}

func ensureSystemPromptFiles(promptsDir string, defaults SystemPromptFiles) (string, error) {
	root, err := resolveSystemPromptsDir(promptsDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, systemPromptDirPerm); err != nil {
		return "", fmt.Errorf("create system prompts directory %s: %w", root, err)
	}

	initialized := systemPromptFilesInitialized(root)
	for _, key := range systemPromptFileKeys() {
		if fileErr := ensureSystemPromptFile(root, key, defaults, initialized); fileErr != nil {
			return "", fileErr
		}
	}
	if !initialized {
		if err := markSystemPromptFilesInitialized(root); err != nil {
			return "", err
		}
	}
	return root, nil
}

func resolveSystemPromptsDir(promptsDir string) (string, error) {
	trimmed := strings.TrimSpace(promptsDir)
	if trimmed == "" {
		return "", errors.New("prompts_dir is empty")
	}
	return filepath.Join(filepath.Clean(trimmed), systemPromptDirName), nil
}

func ResolveSystemPromptsDir(promptsDir string) (string, error) {
	return resolveSystemPromptsDir(promptsDir)
}

func systemPromptFilesInitialized(root string) bool {
	path := filepath.Join(strings.TrimSpace(root), systemPromptInitFile)
	_, err := os.Stat(path)
	return err == nil
}

func markSystemPromptFilesInitialized(root string) error {
	path := filepath.Join(strings.TrimSpace(root), systemPromptInitFile)
	if err := os.WriteFile(path, []byte("initialized"), systemPromptFilePerm); err != nil {
		return fmt.Errorf("write system prompt init marker %s: %w", path, err)
	}
	return nil
}

func ensureSystemPromptFile(root string, key string, defaults SystemPromptFiles, initialized bool) error {
	path, err := systemPromptFilePath(root, key)
	if err != nil {
		return err
	}
	content, err := systemPromptFileValue(defaults, key)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(content)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, systemPromptFilePerm)
	if err == nil {
		if _, writeErr := file.Write([]byte(trimmed)); writeErr != nil {
			file.Close()
			return fmt.Errorf("write system prompt file %s: %w", path, writeErr)
		}
		return file.Close()
	}
	if !errors.Is(err, os.ErrExist) {
		return fmt.Errorf("create system prompt file %s: %w", path, err)
	}
	if initialized || trimmed == "" {
		return nil
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		return fmt.Errorf("read system prompt file %s: %w", path, readErr)
	}
	if strings.TrimSpace(string(raw)) != "" {
		return nil
	}
	if writeErr := os.WriteFile(path, []byte(trimmed), systemPromptFilePerm); writeErr != nil {
		return fmt.Errorf("write system prompt file %s: %w", path, writeErr)
	}
	return nil
}
