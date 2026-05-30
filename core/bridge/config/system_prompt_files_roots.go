package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type systemPromptCandidate struct {
	content string
	modTime time.Time
	index   int
}

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

func syncSystemPromptRoots(roots []string) error {
	if len(roots) < 2 {
		return nil
	}

	for _, key := range systemPromptFileKeys() {
		content, err := newestSystemPromptContent(roots, key)
		if err != nil {
			return err
		}
		for _, root := range roots {
			if writeErr := writeSystemPromptFileIfChanged(root, key, content); writeErr != nil {
				return writeErr
			}
		}
	}
	return nil
}

func newestSystemPromptContent(roots []string, key string) (string, error) {
	candidates, err := loadSystemPromptCandidates(roots, key)
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", nil
	}

	selected := candidates[0]
	for _, candidate := range candidates[1:] {
		switch {
		case candidate.modTime.After(selected.modTime):
			selected = candidate
		case candidate.modTime.Equal(selected.modTime) && candidate.index < selected.index:
			selected = candidate
		}
	}
	return selected.content, nil
}

func loadSystemPromptCandidates(roots []string, key string) ([]systemPromptCandidate, error) {
	candidates := make([]systemPromptCandidate, 0, len(roots))
	for index, root := range roots {
		path, err := systemPromptFilePath(root, key)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat system prompt file %s: %w", path, err)
		}
		content, err := readSystemPromptFile(root, key)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, systemPromptCandidate{
			content: content,
			modTime: info.ModTime(),
			index:   index,
		})
	}
	return candidates, nil
}
