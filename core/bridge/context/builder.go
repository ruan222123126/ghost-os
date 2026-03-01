package context

import "ghost-os/bridge/llm"

// ToolRegistry 定义了上下文构建器依赖的最小工具目录接口。
type ToolRegistry interface {
	ToolDefs() []llm.ToolDef
}

// Builder 负责组装提示词与工具定义。
type Builder struct {
	promptManager *PromptManager
	toolRegistry  ToolRegistry
}

// NewBuilder 创建上下文构建器。
func NewBuilder(pm *PromptManager, registry ToolRegistry) *Builder {
	if pm == nil {
		pm = NewPromptManagerWithDefault()
	}

	return &Builder{
		promptManager: pm,
		toolRegistry:  registry,
	}
}

// BuildSystemPrompt 生成系统提示词文本。
func (b *Builder) BuildSystemPrompt(vars map[string]string) string {
	if b == nil || b.promptManager == nil {
		return NewPromptManagerWithDefault().Render(vars)
	}

	return b.promptManager.Render(vars)
}
