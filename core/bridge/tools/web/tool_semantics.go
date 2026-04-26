package web

import "ghost-os/bridge/llm"

func (WebSearchTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{ReadOnly: true}
}
