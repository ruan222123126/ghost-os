package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SystemPromptsMigrationReport struct {
	PromptsDir          string
	ArchivedFiles       []string
	GeneratedPromptLib  bool
	MigratedMemoryCards []string
	ClearedPresetMemory []string
}

// MigrateLegacySystemPrompts 清理 legacy system prompt 文件、旧 core_prompt 结构与 memory 卡片引用。
func MigrateLegacySystemPrompts() (SystemPromptsMigrationReport, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return SystemPromptsMigrationReport{}, err
	}
	promptsDir, err := resolvePromptsDir(fileCfg, currentEnv())
	if err != nil {
		return SystemPromptsMigrationReport{}, err
	}
	roots, err := ensureSystemPromptRoots(promptsDir, defaultSystemPromptFiles())
	if err != nil {
		return SystemPromptsMigrationReport{}, err
	}
	root := roots[0]
	if _, err := ensurePresetRoot(promptsDir); err != nil {
		return SystemPromptsMigrationReport{}, err
	}
	report := SystemPromptsMigrationReport{PromptsDir: promptsDir}

	archived, err := archiveLegacySystemPromptFiles(root)
	if err != nil {
		return SystemPromptsMigrationReport{}, err
	}
	report.ArchivedFiles = archived

	files, err := readSystemPromptFilesFromRoot(root)
	if err != nil {
		return SystemPromptsMigrationReport{}, err
	}
	nextFiles, generated, migratedMemory, err := migrateLegacySystemPromptState(files)
	if err != nil {
		return SystemPromptsMigrationReport{}, err
	}
	report.GeneratedPromptLib = generated
	report.MigratedMemoryCards = migratedMemory

	presets, err := readPresetsFromRoot(root)
	if err != nil {
		return SystemPromptsMigrationReport{}, err
	}
	nextPresets, clearedPresetIDs, err := migrateLegacyPresetMemoryRefs(presets, buildPromptLibraryByID(nextFiles.PromptLibrary))
	if err != nil {
		return SystemPromptsMigrationReport{}, err
	}
	report.ClearedPresetMemory = clearedPresetIDs

	if err := writeSystemPromptFilesToRoot(root, nextFiles); err != nil {
		return SystemPromptsMigrationReport{}, err
	}
	if err := writePresetFile(root, marshalPresetListOrPanic(nextPresets)); err != nil {
		return SystemPromptsMigrationReport{}, err
	}
	if err := storeSystemPromptFilesCache(root, nextFiles); err != nil {
		return SystemPromptsMigrationReport{}, err
	}
	return report, nil
}

func hasLegacySystemPromptState(promptsDir string) bool {
	root, err := resolveSystemPromptsDir(promptsDir)
	if err != nil {
		return false
	}
	for _, key := range legacySystemPromptFileKeys() {
		if pathExists(filepath.Join(root, key+systemPromptFileExt)) {
			return true
		}
	}

	files, ok := readSystemPromptFilesIfPresent(root)
	if ok {
		if len(files.PromptLibrary) == 0 && strings.TrimSpace(files.CorePrompt) != "" {
			return true
		}
		for _, item := range files.PromptLibrary {
			if item.InsertPoint == SystemPromptInsertPointMemory {
				return true
			}
		}
	}

	presets, ok := readPresetsIfPresent(root)
	if !ok {
		return false
	}
	for _, preset := range presets {
		if strings.TrimSpace(preset.PromptRefs.Memory) != "" {
			return true
		}
	}
	return false
}

func archiveLegacySystemPromptFiles(root string) ([]string, error) {
	archived := make([]string, 0, len(legacySystemPromptFileKeys()))
	suffix := fmt.Sprintf(".legacy.bak.%d", time.Now().UTC().Unix())
	for _, key := range legacySystemPromptFileKeys() {
		path := filepath.Join(root, key+systemPromptFileExt)
		if !pathExists(path) {
			continue
		}
		backup := path + suffix
		if err := os.Rename(path, backup); err != nil {
			return nil, fmt.Errorf("archive legacy system prompt file %s: %w", path, err)
		}
		archived = append(archived, backup)
	}
	return archived, nil
}

func migrateLegacySystemPromptState(files SystemPromptFiles) (SystemPromptFiles, bool, []string, error) {
	next := cloneSystemPromptFiles(files)
	generated := false
	if len(next.PromptLibrary) == 0 && strings.TrimSpace(next.CorePrompt) != "" {
		next.PromptLibrary = buildCoreJobPromptLibrary(&next.CorePrompt)
		generated = true
	}

	migratedMemory := make([]string, 0)
	for index := range next.PromptLibrary {
		if next.PromptLibrary[index].InsertPoint != SystemPromptInsertPointMemory {
			continue
		}
		next.PromptLibrary[index].InsertPoint = SystemPromptInsertPointContext
		next.PromptLibrary[index].Active = false
		migratedMemory = append(migratedMemory, next.PromptLibrary[index].ID)
	}

	library, err := normalizePromptLibrary(next.PromptLibrary)
	if err != nil {
		return SystemPromptFiles{}, false, nil, err
	}
	next.PromptLibrary = library
	next.CorePrompt = compileCorePromptFromLibrary(next.PromptLibrary)
	return next, generated, migratedMemory, nil
}

func migrateLegacyPresetMemoryRefs(
	presets []Preset,
	library promptLibraryByID,
) ([]Preset, []string, error) {
	next := make([]Preset, 0, len(presets))
	cleared := make([]string, 0)
	for _, preset := range presets {
		updated := preset
		if strings.TrimSpace(updated.PromptRefs.Memory) != "" {
			updated.PromptRefs.Memory = ""
			cleared = append(cleared, updated.ID)
		}
		next = append(next, updated)
	}
	normalized, err := normalizePresetList(next, library)
	if err != nil {
		return nil, nil, err
	}
	return normalized, cleared, nil
}

func readSystemPromptFilesIfPresent(root string) (SystemPromptFiles, bool) {
	corePath := filepath.Join(root, systemPromptCorePromptKey+systemPromptFileExt)
	libPath := filepath.Join(root, systemPromptPromptLibraryKey+systemPromptJSONExt)
	if !pathExists(corePath) && !pathExists(libPath) {
		return SystemPromptFiles{}, false
	}

	files := SystemPromptFiles{}
	if raw, err := os.ReadFile(corePath); err == nil {
		files.CorePrompt = strings.TrimSpace(string(raw))
	}
	if raw, err := os.ReadFile(libPath); err == nil {
		if library, parseErr := parsePromptLibrary(string(raw)); parseErr == nil {
			files.PromptLibrary = library
		}
	}
	return files, true
}

func readPresetsIfPresent(root string) ([]Preset, bool) {
	path := filepath.Join(root, presetFileName)
	if !pathExists(path) {
		return nil, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, true
	}
	presets, err := parsePresetList(string(raw))
	if err != nil {
		return nil, true
	}
	return presets, true
}

func marshalPresetListOrPanic(presets []Preset) string {
	content, err := marshalPresetList(presets)
	if err != nil {
		panic(err)
	}
	return content
}
