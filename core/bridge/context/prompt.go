package context

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultPromptVersion = "1.0"
	defaultPromptPath    = "prompts.yaml"
)

var errSystemDefaultRequired = errors.New("system.default is required")

const defaultCoreJob = "You can coordinate local execution, web retrieval, desktop interaction, and human confirmation."

const defaultSystemPromptTemplate = `You are Ghost-OS bridge agent, an AI-driven digital twin execution layer.

## Core Job
{{core_job}}

## Tool Strategy
- Use script_exec as the primary workspace tool for local file discovery, reading, searching, and patching. Prefer tools.list_files/tools.read_file/tools.search_files/tools.apply_diff inside script_exec for deterministic edits.
- Use read_and_summarize for broad multi-file triage; verify exact code with script_exec + tools.read_file before editing.
- Use script_exec for loops, branching, or shell commands. Never call it with {}. For shell commands inside scripts, call tools.bash_exec.
- Use feed_manage to subscribe, list, update, or unsubscribe shared RSS sources; use rss_fetch to read a specific RSS/Atom feed; use web_search for broad internet lookup, use screen_action for desktop OCR or icon matching, and ask_human only when blocked on required user input.
- Prefer screen_action.click_text for visible UI labels; use screen_action.click_icon only for unlabeled icons or template-driven clicks.
- RSS inbox polling and AI filtering run as a backend system pipeline. Do not treat RSS inbox polling as a normal chat-tool chain unless an explicit admin/runtime endpoint is being used.
- When ask_human needs predefined choices, provide selection_mode and options, and ensure the final option allows custom input.

## Limits
- In script_exec helpers, tools.read_file reads at most 200 lines per call.
- In script_exec helpers, tools.search_files returns at most 100 matches per call.
- In script_exec helpers, tools.apply_diff patches one file per call.
- Execution and sandbox budgets are enforced in the native layer.
- File access may be restricted to allowlisted paths and may block sensitive files.

## Operating Context
- OS: {{os_type}}
- Available tools: {{tools_count}}
- Tool list:
{{tool_list}}
- Max turns: {{max_turns}}
- Project root: {{project_root}}

## Response Rules
- If no tool is needed, answer directly.
- For normal turns, reply with plain natural text.
- Keep actions concise, deterministic, and traceable.
- Only when you intentionally end the entire session, output JSON only: {"signal":"END_SESSION","message":"<final reply>"}.`

// PromptConfig 描述 prompts.yaml 的最小结构。
type PromptConfig struct {
	Version string `yaml:"version"`
	System  struct {
		Default string `yaml:"default"`
		CoreJob string `yaml:"core_job"`
	} `yaml:"system"`
}

// PromptManager 负责加载与渲染系统提示词模板。
type PromptManager struct {
	config   PromptConfig
	template string
}

// PromptLoadOptions 描述提示词加载的可选参数。
type PromptLoadOptions struct {
	ConfigPath string
	CoreDir    string
	CoreFiles  []string
}

// NewPromptManager 从配置文件加载提示词模板。
func NewPromptManager(configPath string) (*PromptManager, error) {
	return NewPromptManagerWithOptions(PromptLoadOptions{ConfigPath: configPath})
}

// NewPromptManagerWithOptions 允许在加载 prompts.yaml 的基础上覆盖核心提示词片段。
func NewPromptManagerWithOptions(options PromptLoadOptions) (*PromptManager, error) {
	cfgPath := strings.TrimSpace(options.ConfigPath)
	if cfgPath == "" {
		cfgPath = defaultPromptPath
	}

	raw, resolvedPath, err := readPromptConfigFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read prompts config: %w", err)
	}

	cfg, err := parsePromptYAML(raw, len(options.CoreFiles) > 0)
	if err != nil {
		return nil, fmt.Errorf("parse prompts config %q: %w", resolvedPath, err)
	}
	if len(options.CoreFiles) > 0 {
		coreJob, err := loadCoreJobFromFiles(options.CoreDir, options.CoreFiles)
		if err != nil {
			return nil, fmt.Errorf("load core job: %w", err)
		}
		cfg.System.CoreJob = coreJob
	}

	return &PromptManager{
		config:   cfg,
		template: cfg.System.Default,
	}, nil
}

// NewPromptManagerWithDefault 使用内置模板，避免配置缺失时阻塞启动。
func NewPromptManagerWithDefault() *PromptManager {
	cfg := PromptConfig{
		Version: defaultPromptVersion,
	}
	cfg.System.Default = defaultSystemPromptTemplate
	cfg.System.CoreJob = defaultCoreJob

	return &PromptManager{
		config:   cfg,
		template: cfg.System.Default,
	}
}

// Render 渲染系统提示词并替换模板变量。
func (pm *PromptManager) Render(vars map[string]string) string {
	template := ""
	coreJob := ""
	if pm == nil || strings.TrimSpace(pm.template) == "" {
		template = defaultSystemPromptTemplate
		coreJob = defaultCoreJob
	} else {
		template = pm.template
		coreJob = strings.TrimSpace(pm.config.System.CoreJob)
	}

	merged := make(map[string]string, len(vars)+1)
	if coreJob != "" {
		merged["core_job"] = coreJob
	}
	for key, value := range vars {
		merged[key] = value
	}

	return strings.TrimSpace(RenderTemplate(template, merged))
}

func readPromptConfigFile(path string) ([]byte, string, error) {
	candidates := promptPathCandidates(path)
	var lastErr error

	for _, candidate := range candidates {
		raw, err := os.ReadFile(candidate)
		if err == nil {
			return raw, candidate, nil
		}
		lastErr = err
	}

	if lastErr == nil {
		lastErr = errors.New("prompt config path is empty")
	}
	return nil, "", fmt.Errorf("tried %v: %w", candidates, lastErr)
}

func promptPathCandidates(path string) []string {
	normalized := strings.TrimSpace(path)
	if normalized == "" {
		normalized = defaultPromptPath
	}
	if filepath.IsAbs(normalized) {
		return []string{normalized}
	}

	primary := filepath.Clean(normalized)
	fallback := filepath.Join("core", "bridge", primary)
	if fallback == primary {
		return []string{primary}
	}
	return []string{primary, fallback}
}

func parsePromptYAML(raw []byte, allowMissingCoreJob bool) (PromptConfig, error) {
	cfg := PromptConfig{}
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return PromptConfig{}, err
	}

	if strings.TrimSpace(cfg.Version) == "" {
		cfg.Version = defaultPromptVersion
	}
	if strings.TrimSpace(cfg.System.Default) == "" {
		return PromptConfig{}, errSystemDefaultRequired
	}
	if strings.Contains(cfg.System.Default, "{{core_job}}") && strings.TrimSpace(cfg.System.CoreJob) == "" && !allowMissingCoreJob {
		return PromptConfig{}, errors.New("system.core_job is required when system.default references {{core_job}}")
	}

	return cfg, nil
}

func loadCoreJobFromFiles(coreDir string, files []string) (string, error) {
	trimmedDir := strings.TrimSpace(coreDir)
	if trimmedDir == "" {
		return "", errors.New("prompts core dir is empty")
	}
	baseDir := filepath.Clean(trimmedDir)
	parts := make([]string, 0, len(files))
	for _, file := range files {
		path := strings.TrimSpace(file)
		if path == "" {
			return "", errors.New("prompts core file name is empty")
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		cleaned := filepath.Clean(path)
		raw, err := os.ReadFile(cleaned)
		if err != nil {
			return "", fmt.Errorf("read core job file %s: %w", cleaned, err)
		}
		content := strings.TrimSpace(string(raw))
		if content == "" {
			return "", fmt.Errorf("core job file %s is empty", cleaned)
		}
		parts = append(parts, content)
	}
	if len(parts) == 0 {
		return "", errors.New("no core job files provided")
	}
	return strings.Join(parts, "\n\n"), nil
}
