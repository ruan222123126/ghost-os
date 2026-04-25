package web

import "ghost-os/bridge/llm"

func (WebSearchTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{ReadOnly: true}
}

func (WebRooterTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{ReadOnly: true}
}

func (ImageGenerateTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}
