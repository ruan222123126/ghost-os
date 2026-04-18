package config

import (
	"errors"
	"fmt"
	"strings"

	ctxmgr "ghost-os/bridge/context"
)

const (
	systemPromptDirName  = "system"
	systemPromptFileExt  = ".md"
	systemPromptInitFile = ".initialized"
	systemPromptDirPerm  = 0o755
	systemPromptFilePerm = 0o644

	systemPromptGlobalTemplateKey = "global_template"
	systemPromptCorePromptKey     = "core_prompt"
	systemPromptToolPromptKey     = "tool_prompt"
	systemPromptToolKeySpecKey    = "tool_key_spec"

	defaultSystemPromptGlobalTemplate = "{{base_prompt}}\n\n{{core_prompt}}\n\n{{tool_prompt}}\n\n{{tool_key_spec}}"
)

var errSystemPromptUpdateEmpty = errors.New("at least one of global_template, core_prompt, tool_prompt, or tool_key_spec is required")

// SystemPromptFiles stores the four persisted system prompt file values.
type SystemPromptFiles struct {
	GlobalTemplate string `json:"global_template"`
	CorePrompt     string `json:"core_prompt"`
	ToolPrompt     string `json:"tool_prompt"`
	ToolKeySpec    string `json:"tool_key_spec"`
}

// SystemPromptUpdateRequest updates one or more system prompt files.
type SystemPromptUpdateRequest struct {
	GlobalTemplate *string `json:"global_template,omitempty"`
	CorePrompt     *string `json:"core_prompt,omitempty"`
	ToolPrompt     *string `json:"tool_prompt,omitempty"`
	ToolKeySpec    *string `json:"tool_key_spec,omitempty"`
	TraceID        string  `json:"trace_id,omitempty"`
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
	return readSystemPromptFilesFromRoot(roots[0])
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
	files = applySystemPromptUpdate(files, req)
	if err := writeSystemPromptFilesToRoots(roots, files); err != nil {
		return SystemPromptFiles{}, err
	}
	return files, nil
}

// RenderSystemPrompt injects the base prompt and local sections into the global template.
func RenderSystemPrompt(files SystemPromptFiles, basePrompt string) string {
	vars := map[string]string{
		"base_prompt":    strings.TrimSpace(basePrompt),
		"core_prompt":    strings.TrimSpace(files.CorePrompt),
		"tool_prompt":    strings.TrimSpace(files.ToolPrompt),
		"tool_key_spec":  strings.TrimSpace(files.ToolKeySpec),
	}
	template := strings.TrimSpace(files.GlobalTemplate)
	return strings.TrimSpace(ctxmgr.RenderTemplate(template, vars))
}

func defaultSystemPromptFiles() SystemPromptFiles {
	return SystemPromptFiles{
		GlobalTemplate: defaultSystemPromptGlobalTemplate,
		CorePrompt:     "",
		ToolPrompt:     "",
		ToolKeySpec:    "",
	}
}

func applySystemPromptUpdate(files SystemPromptFiles, req SystemPromptUpdateRequest) SystemPromptFiles {
	if req.GlobalTemplate != nil {
		files.GlobalTemplate = trimSystemPromptValue(req.GlobalTemplate)
	}
	if req.CorePrompt != nil {
		files.CorePrompt = trimSystemPromptValue(req.CorePrompt)
	}
	if req.ToolPrompt != nil {
		files.ToolPrompt = trimSystemPromptValue(req.ToolPrompt)
	}
	if req.ToolKeySpec != nil {
		files.ToolKeySpec = trimSystemPromptValue(req.ToolKeySpec)
	}
	return files
}

func trimSystemPromptValue(raw *string) string {
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(*raw)
}

func (req SystemPromptUpdateRequest) hasUpdates() bool {
	return req.GlobalTemplate != nil || req.CorePrompt != nil || req.ToolPrompt != nil || req.ToolKeySpec != nil
}

func defaultSystemPromptFileValues() map[string]string {
	files := defaultSystemPromptFiles()
	return map[string]string{
		systemPromptGlobalTemplateKey: files.GlobalTemplate,
		systemPromptCorePromptKey:     files.CorePrompt,
		systemPromptToolPromptKey:     files.ToolPrompt,
		systemPromptToolKeySpecKey:    files.ToolKeySpec,
	}
}

func systemPromptFileKeys() []string {
	return []string{
		systemPromptGlobalTemplateKey,
		systemPromptCorePromptKey,
		systemPromptToolPromptKey,
		systemPromptToolKeySpecKey,
	}
}

func systemPromptFilesFromMap(values map[string]string) SystemPromptFiles {
	return SystemPromptFiles{
		GlobalTemplate: strings.TrimSpace(values[systemPromptGlobalTemplateKey]),
		CorePrompt:     strings.TrimSpace(values[systemPromptCorePromptKey]),
		ToolPrompt:     strings.TrimSpace(values[systemPromptToolPromptKey]),
		ToolKeySpec:    strings.TrimSpace(values[systemPromptToolKeySpecKey]),
	}
}

func systemPromptFileValue(files SystemPromptFiles, key string) (string, bool) {
	switch key {
	case systemPromptGlobalTemplateKey:
		return files.GlobalTemplate, true
	case systemPromptCorePromptKey:
		return files.CorePrompt, true
	case systemPromptToolPromptKey:
		return files.ToolPrompt, true
	case systemPromptToolKeySpecKey:
		return files.ToolKeySpec, true
	default:
		return "", false
	}
}

func setSystemPromptFileValue(files *SystemPromptFiles, key string, value string) error {
	if files == nil {
		return errors.New("system prompt files are nil")
	}
	switch key {
	case systemPromptGlobalTemplateKey:
		files.GlobalTemplate = value
	case systemPromptCorePromptKey:
		files.CorePrompt = value
	case systemPromptToolPromptKey:
		files.ToolPrompt = value
	case systemPromptToolKeySpecKey:
		files.ToolKeySpec = value
	default:
		return fmt.Errorf("unknown system prompt key: %s", key)
	}
	return nil
}
