package tools

import "ghost-os/bridge/llm"

// ToolSemanticProvider 允许工具声明只读/副作用语义，供运行时协议复用。
type ToolSemanticProvider interface {
	ToolSemantics() llm.ToolSemantics
}

// ToolDefFromTool 把运行时工具导出为对模型可见的稳定定义。
func ToolDefFromTool(tool Tool) llm.ToolDef {
	if tool == nil {
		return llm.ToolDef{}
	}

	def := llm.ToolDef{
		Name:        tool.Name(),
		Description: tool.Description(),
		Parameters:  tool.Parameters(),
	}
	if provider, ok := tool.(ToolSemanticProvider); ok {
		def.Semantics = provider.ToolSemantics()
	}
	return def
}
