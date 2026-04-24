package tools

import "ghost-os/bridge/llm"

func (AskHumanTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (CodexCLITool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (ProCompleteTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (ProUpdateRecordTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (ScriptExecTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (TaskManageTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (ToolSearchTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}
