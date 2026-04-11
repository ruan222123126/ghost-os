package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type toolPromptCandidate struct {
	prompt        string
	modTime       time.Time
	index         int
	isBaseDefault bool
}

func syncToolPromptRoots(roots []string) error {
	if len(roots) < 2 {
		return nil
	}

	for _, name := range configuredToolNames() {
		prompt, err := newestToolPromptContent(roots, name)
		if err != nil {
			return err
		}
		for _, root := range roots {
			if writeErr := writeToolPromptFileIfChanged(root, name, prompt); writeErr != nil {
				return writeErr
			}
		}
	}
	return nil
}

func newestToolPromptContent(roots []string, name string) (string, error) {
	basePrompt, _ := toolBasePrompt(name)
	candidates, hasNonDefault, err := loadToolPromptCandidates(roots, name, strings.TrimSpace(basePrompt))
	if err != nil {
		return "", err
	}
	if len(candidates) == 0 {
		return "", nil
	}

	selected := candidates[0]
	for _, candidate := range candidates[1:] {
		if hasNonDefault {
			if selected.isBaseDefault && !candidate.isBaseDefault {
				selected = candidate
				continue
			}
			if !selected.isBaseDefault && candidate.isBaseDefault {
				continue
			}
		}
		switch {
		case candidate.modTime.After(selected.modTime):
			selected = candidate
		case candidate.modTime.Equal(selected.modTime) && candidate.index < selected.index:
			selected = candidate
		}
	}
	return selected.prompt, nil
}

func loadToolPromptCandidates(
	roots []string,
	name string,
	basePrompt string,
) ([]toolPromptCandidate, bool, error) {
	candidates := make([]toolPromptCandidate, 0, len(roots))
	hasNonDefault := false
	for index, root := range roots {
		path, err := toolPromptFilePath(root, name)
		if err != nil {
			return nil, false, err
		}
		info, err := os.Stat(path)
		if err != nil {
			return nil, false, fmt.Errorf("stat tool prompt file %s: %w", path, err)
		}
		prompt, err := readToolPromptOverrideFile(root, name)
		if err != nil {
			return nil, false, err
		}
		isBaseDefault := prompt == basePrompt
		if !isBaseDefault {
			hasNonDefault = true
		}
		candidates = append(candidates, toolPromptCandidate{
			prompt:        prompt,
			modTime:       info.ModTime(),
			index:         index,
			isBaseDefault: isBaseDefault,
		})
	}
	return candidates, hasNonDefault, nil
}
