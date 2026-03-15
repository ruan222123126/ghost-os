package context

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	defaultPromptVersion = "1.0"
	defaultPromptPath    = "prompts.yaml"
	defaultToolGuidance  = "- Use only the tools included in the structured tool schema for this turn."
	defaultDynamicState  = "- No dynamic tools loaded."
)

var (
	defaultPromptConfig     PromptConfig
	defaultPromptConfigErr  error
	defaultPromptConfigOnce sync.Once

	errSystemDefaultRequired = errors.New("system.default is required")
)

type PromptSystemConfig struct {
	Default            string `yaml:"default"`
	CoreJob            string `yaml:"core_job"`
	RuntimeConstraints string `yaml:"runtime_constraints"`
	ResponseRules      string `yaml:"response_rules"`
}

// PromptConfig 描述 prompts.yaml 的最小结构。
type PromptConfig struct {
	Version string             `yaml:"version"`
	System  PromptSystemConfig `yaml:"system"`
}

// PromptManager 负责加载与渲染系统提示词模板。
type PromptManager struct {
	config   PromptConfig
	template string
}

// PromptLoadOptions 描述提示词加载的可选参数。
type PromptLoadOptions struct {
	ConfigPath             string
	CoreDir                string
	CoreFiles              []string
	RuntimeConstraintFiles []string
	ResponseRuleFiles      []string
}

type promptSectionOptions struct {
	coreJob            bool
	runtimeConstraints bool
	responseRules      bool
}

type promptSectionRequirement struct {
	field       string
	placeholder string
	value       string
	optional    bool
}

// NewPromptManager 从配置文件加载提示词模板。
func NewPromptManager(configPath string) (*PromptManager, error) {
	return NewPromptManagerWithOptions(PromptLoadOptions{ConfigPath: configPath})
}

// NewPromptManagerWithOptions 允许在加载 prompts.yaml 的基础上覆盖提示词片段。
func NewPromptManagerWithOptions(options PromptLoadOptions) (*PromptManager, error) {
	cfgPath := strings.TrimSpace(options.ConfigPath)
	if cfgPath == "" {
		cfgPath = defaultPromptPath
	}

	raw, resolvedPath, err := readPromptConfigFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read prompts config: %w", err)
	}

	cfg, err := parsePromptYAML(raw, promptSectionOptions{
		coreJob:            len(options.CoreFiles) > 0,
		runtimeConstraints: len(options.RuntimeConstraintFiles) > 0,
		responseRules:      len(options.ResponseRuleFiles) > 0,
	})
	if err != nil {
		return nil, fmt.Errorf("parse prompts config %q: %w", resolvedPath, err)
	}
	if err := applyPromptOverrides(&cfg, options); err != nil {
		return nil, err
	}

	return newPromptManager(cfg), nil
}

// NewPromptManagerWithDefault 使用仓库内置 prompts.yaml，避免配置缺失时阻塞启动。
func NewPromptManagerWithDefault() *PromptManager {
	return newPromptManager(mustDefaultPromptConfig())
}

func newPromptManager(cfg PromptConfig) *PromptManager {
	return &PromptManager{
		config:   cfg,
		template: cfg.System.Default,
	}
}

// Render 渲染系统提示词并替换模板变量。
func (pm *PromptManager) Render(vars map[string]string) string {
	cfg := mustDefaultPromptConfig()
	if pm != nil && strings.TrimSpace(pm.template) != "" {
		cfg = pm.config
	}

	merged := map[string]string{
		"core_job":            strings.TrimSpace(cfg.System.CoreJob),
		"tool_guidance":       defaultToolGuidance,
		"dynamic_tool_state":  defaultDynamicState,
		"runtime_constraints": strings.TrimSpace(cfg.System.RuntimeConstraints),
		"response_rules":      strings.TrimSpace(cfg.System.ResponseRules),
	}
	for key, value := range vars {
		merged[key] = value
	}

	return strings.TrimSpace(RenderTemplate(cfg.System.Default, merged))
}

func mustDefaultPromptConfig() PromptConfig {
	defaultPromptConfigOnce.Do(func() {
		defaultPromptConfig, defaultPromptConfigErr = parsePromptYAML(
			[]byte(defaultPromptConfigYAML),
			promptSectionOptions{},
		)
	})
	if defaultPromptConfigErr != nil {
		panic(fmt.Sprintf("parse bundled prompts config: %v", defaultPromptConfigErr))
	}
	return defaultPromptConfig
}

func applyPromptOverrides(cfg *PromptConfig, options PromptLoadOptions) error {
	overrides := []struct {
		files []string
		name  string
		set   func(string)
	}{
		{
			files: options.CoreFiles,
			name:  "core job",
			set:   func(content string) { cfg.System.CoreJob = content },
		},
		{
			files: options.RuntimeConstraintFiles,
			name:  "runtime constraints",
			set:   func(content string) { cfg.System.RuntimeConstraints = content },
		},
		{
			files: options.ResponseRuleFiles,
			name:  "response rules",
			set:   func(content string) { cfg.System.ResponseRules = content },
		},
	}

	for _, override := range overrides {
		if len(override.files) == 0 {
			continue
		}
		content, err := loadPromptSectionFromFiles(options.CoreDir, override.files)
		if err != nil {
			return fmt.Errorf("load %s: %w", override.name, err)
		}
		override.set(content)
	}
	return nil
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

func parsePromptYAML(raw []byte, optional promptSectionOptions) (PromptConfig, error) {
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

	requirements := []promptSectionRequirement{
		{
			field:       "system.core_job",
			placeholder: "{{core_job}}",
			value:       cfg.System.CoreJob,
			optional:    optional.coreJob,
		},
		{
			field:       "system.runtime_constraints",
			placeholder: "{{runtime_constraints}}",
			value:       cfg.System.RuntimeConstraints,
			optional:    optional.runtimeConstraints,
		},
		{
			field:       "system.response_rules",
			placeholder: "{{response_rules}}",
			value:       cfg.System.ResponseRules,
			optional:    optional.responseRules,
		},
	}
	for _, requirement := range requirements {
		if err := validatePromptSection(cfg.System.Default, requirement); err != nil {
			return PromptConfig{}, err
		}
	}
	return cfg, nil
}

func validatePromptSection(template string, requirement promptSectionRequirement) error {
	if !strings.Contains(template, requirement.placeholder) {
		return nil
	}
	if requirement.optional || strings.TrimSpace(requirement.value) != "" {
		return nil
	}
	return fmt.Errorf(
		"%s is required when system.default references %s",
		requirement.field,
		requirement.placeholder,
	)
}

func loadPromptSectionFromFiles(baseDir string, files []string) (string, error) {
	parts := make([]string, 0, len(files))
	root := strings.TrimSpace(baseDir)
	if root != "" {
		root = filepath.Clean(root)
	}
	for _, file := range files {
		path := strings.TrimSpace(file)
		if path == "" {
			return "", errors.New("prompt section file name is empty")
		}
		if !filepath.IsAbs(path) {
			if root == "" {
				return "", errors.New("prompts dir is empty for relative prompt section file")
			}
			path = filepath.Join(root, path)
		}
		cleaned := filepath.Clean(path)
		raw, err := os.ReadFile(cleaned)
		if err != nil {
			return "", fmt.Errorf("read prompt section file %s: %w", cleaned, err)
		}
		content := strings.TrimSpace(string(raw))
		if content == "" {
			return "", fmt.Errorf("prompt section file %s is empty", cleaned)
		}
		parts = append(parts, content)
	}
	if len(parts) == 0 {
		return "", errors.New("no prompt section files provided")
	}
	return strings.Join(parts, "\n\n"), nil
}
