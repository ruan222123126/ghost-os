package screen

import "ghost-os/bridge/llm"

func (ScreenActionTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (ScreenControlTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}
