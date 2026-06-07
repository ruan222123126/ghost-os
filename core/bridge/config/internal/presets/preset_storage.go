package presets

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ghost-os/bridge/config/internal/prompts"
)

type presetCandidate struct {
	content string
	modTime time.Time
	index   int
}

func ensurePresetRoots(promptsDir string) ([]string, error) {
	dirs, err := prompts.ResolveSystemPromptDirs(promptsDir)
	if err != nil {
		return nil, err
	}

	roots := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		root, ensureErr := ensurePresetRoot(dir)
		if ensureErr != nil {
			return nil, ensureErr
		}
		roots = append(roots, root)
	}
	return roots, nil
}

func ensurePresetRoot(promptsDir string) (string, error) {
	root, err := prompts.ResolveSystemPromptsDir(promptsDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, prompts.SystemPromptDirPerm); err != nil {
		return "", fmt.Errorf("create preset directory %s: %w", root, err)
	}

	path := filepath.Join(root, presetFileName)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, prompts.SystemPromptFilePerm)
	if err == nil {
		if _, writeErr := file.Write([]byte("[]")); writeErr != nil {
			file.Close()
			return "", fmt.Errorf("write preset file %s: %w", path, writeErr)
		}
		if closeErr := file.Close(); closeErr != nil {
			return "", fmt.Errorf("close preset file %s: %w", path, closeErr)
		}
		return root, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return "", fmt.Errorf("create preset file %s: %w", path, err)
	}
	return root, nil
}

func readPresetsFromRoot(root string) ([]Preset, error) {
	content, err := readPresetFile(root)
	if err != nil {
		return nil, err
	}
	return parsePresetList(content)
}

func writePresetsToRoots(roots []string, presets []Preset) error {
	content, err := marshalPresetList(presets)
	if err != nil {
		return err
	}
	for _, root := range roots {
		if err := writePresetFileIfChanged(root, content); err != nil {
			return err
		}
	}
	return nil
}

func syncPresetRoots(roots []string) error {
	if len(roots) < 2 {
		return nil
	}

	content, err := newestPresetContent(roots)
	if err != nil {
		return err
	}
	for _, root := range roots {
		if err := writePresetFileIfChanged(root, content); err != nil {
			return err
		}
	}
	return nil
}

func newestPresetContent(roots []string) (string, error) {
	candidates, err := loadPresetCandidates(roots)
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "[]", nil
	}

	selected := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.modTime.After(selected.modTime) {
			selected = candidate
			continue
		}
		if candidate.modTime.Equal(selected.modTime) && candidate.index < selected.index {
			selected = candidate
		}
	}
	return selected.content, nil
}

func loadPresetCandidates(roots []string) ([]presetCandidate, error) {
	candidates := make([]presetCandidate, 0, len(roots))
	for index, root := range roots {
		path, err := presetFilePath(root)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("stat preset file %s: %w", path, err)
		}
		content, err := readPresetFile(root)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, presetCandidate{
			content: content,
			modTime: info.ModTime(),
			index:   index,
		})
	}
	return candidates, nil
}

func readPresetFile(root string) (string, error) {
	path, err := presetFilePath(root)
	if err != nil {
		return "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read preset file %s: %w", path, err)
	}
	return strings.TrimSpace(string(raw)), nil
}

func writePresetFileIfChanged(root string, content string) error {
	current, err := readPresetFile(root)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(content)
	if current == trimmed {
		return nil
	}
	return writePresetFile(root, trimmed)
}

func writePresetFile(root string, content string) error {
	path, err := presetFilePath(root)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)), prompts.SystemPromptFilePerm); err != nil {
		return fmt.Errorf("write preset file %s: %w", path, err)
	}
	return nil
}

func presetFilePath(root string) (string, error) {
	trimmedRoot := strings.TrimSpace(root)
	if trimmedRoot == "" {
		return "", errors.New("preset root is empty")
	}
	return filepath.Join(trimmedRoot, presetFileName), nil
}
