package tools

import (
	"ghost-os/bridge/artifacts"
	"ghost-os/bridge/llm"
	toolscreen "ghost-os/bridge/tools/screen"
)

const (
	screenControlToolName = "screen_control"
)

type ScreenActionTool = toolscreen.ScreenActionTool

type ScreenControlTool = toolscreen.ScreenControlTool

func NewScreenActionTool(client ExecutionClient) Tool {
	return toolscreen.NewScreenActionTool(client)
}

func NewScreenControlTool(
	client ExecutionClient,
	model llm.Completer,
	artifactStore *artifacts.SessionArtifactStore,
) Tool {
	return toolscreen.NewScreenControlTool(client, model, artifactStore)
}
