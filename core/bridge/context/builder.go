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

// BuildRequest 组装一次 LLM 请求，包含消息与工具定义。
func (b *Builder) BuildRequest(messages []llm.Message) llm.CompletionRequest {
	req := llm.CompletionRequest{
		Messages: llm.CloneMessages(messages),
	}

	if b == nil || b.toolRegistry == nil {
		return req
	}

	req.Tools = cloneToolDefs(b.toolRegistry.ToolDefs())
	return req
}

func cloneToolDefs(defs []llm.ToolDef) []llm.ToolDef {
	if len(defs) == 0 {
		return nil
	}

	out := make([]llm.ToolDef, len(defs))
	for i, tool := range defs {
		out[i] = llm.ToolDef{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  cloneRawJSON(tool.Parameters),
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
