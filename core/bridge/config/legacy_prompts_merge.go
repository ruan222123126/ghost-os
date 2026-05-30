package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func mergePromptManagedFiles(canonicalDir string, legacyDir string) ([]string, error) {
	merged := make([]string, 0, len(configuredToolNames())+3)
	for _, name := range configuredToolNames() {
		target := filepath.Join(canonicalDir, toolPromptDirName, name+toolPromptFileExt)
		legacy := filepath.Join(legacyDir, toolPromptDirName, name+toolPromptFileExt)
		ok, err := mergeManagedFile(target, legacy)
		if err != nil {
			return nil, err
		}
		if ok {
			merged = append(merged, filepath.Join(toolPromptDirName, name+toolPromptFileExt))
		}
	}
	for _, fileName := range []string{
		systemPromptCorePromptKey + systemPromptFileExt,
		systemPromptPromptLibraryKey + systemPromptJSONExt,
		presetFileName,
	} {
		target := filepath.Join(canonicalDir, systemPromptDirName, fileName)
		legacy := filepath.Join(legacyDir, systemPromptDirName, fileName)
		ok, err := mergeManagedFile(target, legacy)
		if err != nil {
			return nil, err
		}
		if ok {
			merged = append(merged, filepath.Join(systemPromptDirName, fileName))
		}
	}
	return merged, nil
}

func mergeManagedFile(canonicalPath string, legacyPath string) (bool, error) {
	selected, ok, err := newestManagedFileContent(canonicalPath, legacyPath)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(canonicalPath), toolPromptDirPerm); err != nil {
		return false, fmt.Errorf("create prompt directory %s: %w", filepath.Dir(canonicalPath), err)
	}
	if err := os.WriteFile(canonicalPath, []byte(selected.content), toolPromptFilePerm); err != nil {
		return false, fmt.Errorf("write canonical prompt file %s: %w", canonicalPath, err)
	}
	return true, nil
}

type managedFileSelection struct {
	content string
	modTime time.Time
	source  string
}

func newestManagedFileContent(canonicalPath string, legacyPath string) (managedFileSelection, bool, error) {
	candidates := make([]managedFileSelection, 0, 2)
	if candidate, ok, err := managedFileCandidate(canonicalPath); err != nil {
		return managedFileSelection{}, false, err
	} else if ok {
		candidates = append(candidates, candidate)
	}
	if candidate, ok, err := managedFileCandidate(legacyPath); err != nil {
		return managedFileSelection{}, false, err
	} else if ok {
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		return managedFileSelection{}, false, nil
	}

	selected := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.modTime.After(selected.modTime) {
			selected = candidate
			continue
		}
		if candidate.modTime.Equal(selected.modTime) && selected.source != canonicalPath && candidate.source == canonicalPath {
			selected = candidate
		}
	}
	return selected, true, nil
}

func managedFileCandidate(path string) (managedFileSelection, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return managedFileSelection{}, false, nil
		}
		return managedFileSelection{}, false, fmt.Errorf("stat prompt file %s: %w", path, err)
	}
	if info.IsDir() {
		return managedFileSelection{}, false, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return managedFileSelection{}, false, fmt.Errorf("read prompt file %s: %w", path, err)
	}
	return managedFileSelection{
		content: strings.TrimSpace(string(raw)),
		modTime: info.ModTime(),
		source:  path,
	}, true, nil
}

func hasLegacyPromptsDirState(promptsDir string) bool {
	canonicalDir, legacyDir, ok := promptMirrorPair(promptsDir)
	if !ok {
		return false
	}
	return promptsDir == legacyDir || (pathExists(canonicalDir) && pathExists(legacyDir))
}

func promptMirrorPair(promptsDir string) (string, string, bool) {
	cleaned := filepath.Clean(strings.TrimSpace(promptsDir))
	mirror := mirrorPromptsDir(cleaned)
	if mirror == "" {
		return "", "", false
	}
	switch filepath.Base(filepath.Dir(cleaned)) {
	case ghostOSDirName:
		return cleaned, mirror, true
	case ghostDirName:
		return mirror, cleaned, true
	default:
		return "", "", false
	}
}

func resolvedStoredPromptsDir(fileCfg bridgeFileConfig) string {
	if fileCfg.PromptsDir == nil {
		return ""
	}
	resolved, err := resolveUserPath(stringValue(fileCfg.PromptsDir))
	if err != nil {
		return ""
	}
	return resolved
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
