package tools

import "ghost-os/bridge/llm"

func (AskHumanTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (BrowserControlTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (CodexCLITool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (ComputerUseTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (FeedManageTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (MemoryLearnedListTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{ReadOnly: true}
}

func (MemoryManageTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (MemoryRecallDebugTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{ReadOnly: true}
}

func (ProCompleteTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (ProUpdateRecordTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (ReadAndSummarizeTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{ReadOnly: true}
}

func (RSSFetchTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{ReadOnly: true}
}

func (ScreenActionTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (ScriptExecTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (SendFileTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (SetProjectRootTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (TaskManageTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (TextInputTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (ToolSearchTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{SideEffect: true}
}

func (WebSearchTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{ReadOnly: true}
}

func (WebRooterTool) ToolSemantics() llm.ToolSemantics {
	return llm.ToolSemantics{ReadOnly: true}
}
