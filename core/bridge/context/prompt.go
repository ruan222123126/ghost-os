package context

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

const (
	defaultPromptVersion = "1.0"
	defaultPromptPath    = "prompts.yaml"
	defaultToolGuidance  = "- Use only the tools included in the structured tool schema for this turn."
	defaultDynamicState  = "- No dynamic tools loaded."
	defaultSkillContext  = "- No dynamic skills loaded."
)

var (
	defaultPromptConfig     PromptConfig
	defaultPromptConfigErr  error
	defaultPromptConfigOnce sync.Once

	errPromptManagerNil      = errors.New("prompt manager is nil")
	errPromptTemplateNil     = errors.New("prompt template is empty")
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

// NewPromptManagerWithOptions 允许在加载 prompts.yaml 的基础上覆盖提示词片段。
func NewPromptManagerWithOptions(options PromptLoadOptions) (*PromptManager, error) {
	cfg, err := loadPromptConfigFromOptions(options)
	if err != nil {
		return nil, err
	}
	return newPromptManager(cfg), nil
}

// NewPromptManagerWithDefault 使用仓库内置 prompts.yaml。
func NewPromptManagerWithDefault() *PromptManager {
	return newPromptManager(mustDefaultPromptConfig())
}

func newPromptManager(cfg PromptConfig) *PromptManager {
	return &PromptManager{config: cfg, template: cfg.System.Default}
}

// Render 渲染系统提示词并替换模板变量。
func (pm *PromptManager) Render(vars map[string]string) string {
	if pm == nil {
		panic(errPromptManagerNil)
	}
	if strings.TrimSpace(pm.template) == "" {
		panic(errPromptTemplateNil)
	}

	merged := map[string]string{
		"core_job":              strings.TrimSpace(pm.config.System.CoreJob),
		"tool_guidance":         defaultToolGuidance,
		"dynamic_tool_state":    defaultDynamicState,
		"dynamic_skill_context": defaultSkillContext,
		"runtime_constraints":   strings.TrimSpace(pm.config.System.RuntimeConstraints),
		"response_rules":        strings.TrimSpace(pm.config.System.ResponseRules),
	}
	for key, value := range vars {
		merged[key] = value
	}

	return strings.TrimSpace(RenderTemplate(pm.template, merged))
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
