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

// BuildRequest 复制消息与工具定义，避免请求构建阶段共享可变切片。
func (b *Builder) BuildRequest(messages []llm.Message) llm.CompletionRequest {
	if b == nil {
		return llm.CompletionRequest{
			Messages: llm.CloneMessages(messages),
		}
	}

	return llm.CompletionRequest{
		Messages: llm.CloneMessages(messages),
		Tools:    cloneToolDefs(b.toolRegistry),
	}
}

func cloneToolDefs(registry ToolRegistry) []llm.ToolDef {
	if registry == nil {
		return nil
	}

	defs := registry.ToolDefs()
	if len(defs) == 0 {
		return nil
	}

	out := make([]llm.ToolDef, len(defs))
	for i, def := range defs {
		out[i] = llm.ToolDef{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  cloneRawJSON(def.Parameters),
		}
	}

	return out
}

func cloneRawJSON(raw []byte) []byte {
	if len(raw) == 0 {
		return nil
	}

	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}
