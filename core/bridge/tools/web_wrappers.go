package tools

import toolweb "ghost-os/bridge/tools/web"

type WebSearchConfig = toolweb.WebSearchConfig

type WebSearchTool = toolweb.WebSearchTool

func NewWebSearchTool(cfg WebSearchConfig) Tool {
	return toolweb.NewWebSearchTool(cfg)
}
