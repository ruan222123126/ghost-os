package config

import (
	"errors"
	"fmt"
	"strings"
)

const (
	systemPromptDirName  = "system"
	systemPromptFileExt  = ".md"
	systemPromptJSONExt  = ".json"
	systemPromptInitFile = ".initialized"
	systemPromptDirPerm  = 0o755
	systemPromptFilePerm = 0o644

	systemPromptCorePromptKey    = "core_prompt"
	systemPromptPromptLibraryKey = "prompt_library"
)

var (
	errSystemPromptUpdateEmpty    = errors.New("core_prompt or prompt_library is required")
	errSystemPromptUpdateConflict = errors.New("core_prompt and prompt_library cannot be updated together")
	errSystemPromptLibraryInvalid = errors.New("prompt_library is invalid")
)

type SystemPromptInsertPoint string

const (
	SystemPromptInsertPointCoreJob SystemPromptInsertPoint = "core_job"
	SystemPromptInsertPointMemory  SystemPromptInsertPoint = "memory"
)

// SystemPromptLibraryItem stores one prompt card in the prompt library.
type SystemPromptLibraryItem struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	InsertPoint SystemPromptInsertPoint `json:"insert_point"`
	Content     string                  `json:"content"`
	Active      bool                    `json:"active"`
}

// SystemPromptFiles stores the persisted system prompt file values.
type SystemPromptFiles struct {
	CorePrompt    string                    `json:"core_prompt"`
	PromptLibrary []SystemPromptLibraryItem `json:"prompt_library"`
}

// SystemPromptUpdateRequest updates one or more system prompt files.
type SystemPromptUpdateRequest struct {
	CorePrompt    *string                    `json:"core_prompt,omitempty"`
	PromptLibrary *[]SystemPromptLibraryItem `json:"prompt_library,omitempty"`
	TraceID       string                     `json:"trace_id,omitempty"`
}

// LoadSystemPromptFiles initializes, synchronizes, and reads the system prompt files.
func LoadSystemPromptFiles(promptsDir string) (SystemPromptFiles, error) {
	roots, err := ensureSystemPromptRoots(promptsDir, defaultSystemPromptFiles())
	if err != nil {
		return SystemPromptFiles{}, err
	}
	if err := syncSystemPromptRoots(roots); err != nil {
		return SystemPromptFiles{}, err
	}
	files, err := readSystemPromptFilesFromRoot(roots[0])
	if err != nil {
		return SystemPromptFiles{}, err
	}
	return migrateSystemPromptFiles(roots, files)
}

// UpdateSystemPromptFiles persists a partial update and returns the reloaded files.
func UpdateSystemPromptFiles(promptsDir string, req SystemPromptUpdateRequest) (SystemPromptFiles, error) {
	if !req.hasUpdates() {
		return SystemPromptFiles{}, errSystemPromptUpdateEmpty
	}

	roots, err := ensureSystemPromptRoots(promptsDir, defaultSystemPromptFiles())
	if err != nil {
		return SystemPromptFiles{}, err
	}
	if err := syncSystemPromptRoots(roots); err != nil {
		return SystemPromptFiles{}, err
	}

	files, err := readSystemPromptFilesFromRoot(roots[0])
	if err != nil {
		return SystemPromptFiles{}, err
	}
	files, err = migrateSystemPromptFiles(roots, files)
	if err != nil {
		return SystemPromptFiles{}, err
	}
	files, err = applySystemPromptUpdate(files, req)
	if err != nil {
		return SystemPromptFiles{}, err
	}
	if err := writeSystemPromptFilesToRoots(roots, files); err != nil {
		return SystemPromptFiles{}, err
	}
	return files, nil
}

func defaultSystemPromptFiles() SystemPromptFiles {
	return SystemPromptFiles{
		CorePrompt:    "",
		PromptLibrary: []SystemPromptLibraryItem{},
	}
}

func applySystemPromptUpdate(files SystemPromptFiles, req SystemPromptUpdateRequest) (SystemPromptFiles, error) {
	if req.hasConflict() {
		return SystemPromptFiles{}, errSystemPromptUpdateConflict
	}
	if req.CorePrompt != nil {
		files.PromptLibrary = buildCoreJobPromptLibrary(req.CorePrompt)
		files.CorePrompt = compileCorePromptFromLibrary(files.PromptLibrary)
		return files, nil
	}
	if req.PromptLibrary != nil {
		library, err := normalizePromptLibrary(*req.PromptLibrary)
		if err != nil {
			return SystemPromptFiles{}, err
		}
		files.PromptLibrary = library
		files.CorePrompt = compileCorePromptFromLibrary(files.PromptLibrary)
	}
	return files, nil
}

func trimSystemPromptValue(raw *string) string {
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(*raw)
}

func migrateSystemPromptFiles(roots []string, files SystemPromptFiles) (SystemPromptFiles, error) {
	if err := removeLegacySystemPromptFiles(roots); err != nil {
		return SystemPromptFiles{}, err
	}
	migrated, changed, err := migrateAndNormalizeSystemPromptFiles(files)
	if err != nil {
		return SystemPromptFiles{}, err
	}
	if changed {
		if err := writeSystemPromptFilesToRoots(roots, migrated); err != nil {
			return SystemPromptFiles{}, err
		}
	}
	return migrated, nil
}

func (req SystemPromptUpdateRequest) hasUpdates() bool {
	return req.CorePrompt != nil || req.PromptLibrary != nil
}

func (req SystemPromptUpdateRequest) hasConflict() bool {
	return req.CorePrompt != nil && req.PromptLibrary != nil
}

func defaultSystemPromptFileValues() map[string]string {
	values := map[string]string{}
	for _, key := range systemPromptFileKeys() {
		content, err := systemPromptFileValue(defaultSystemPromptFiles(), key)
		if err != nil {
			panic(err)
		}
		values[key] = content
	}
	return values
}

func systemPromptFileKeys() []string {
	return []string{
		systemPromptCorePromptKey,
		systemPromptPromptLibraryKey,
	}
}

func systemPromptFileValue(files SystemPromptFiles, key string) (string, error) {
	switch key {
	case systemPromptCorePromptKey:
		return trimSystemPromptValue(&files.CorePrompt), nil
	case systemPromptPromptLibraryKey:
		return marshalPromptLibrary(files.PromptLibrary)
	default:
		return "", fmt.Errorf("unknown system prompt key: %s", key)
	}
}

func setSystemPromptFileValue(files *SystemPromptFiles, key string, value string) error {
	if files == nil {
		return errors.New("system prompt files are nil")
	}
	switch key {
	case systemPromptCorePromptKey:
		files.CorePrompt = value
	case systemPromptPromptLibraryKey:
		library, err := parsePromptLibrary(value)
		if err != nil {
			return err
		}
		files.PromptLibrary = library
	default:
		return fmt.Errorf("unknown system prompt key: %s", key)
	}
	return nil
}
