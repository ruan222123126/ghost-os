package tools

import (
	"context"

	"ghost-os/bridge/artifacts"
	toolweb "ghost-os/bridge/tools/web"
)

const webRooterToolName = "web_rooter"

type WebSearchConfig = toolweb.WebSearchConfig

type WebRooterConfig = toolweb.WebRooterConfig

type ImageGenerateConfig = toolweb.ImageGenerateConfig

type RSSFetchOptions = toolweb.RSSFetchOptions

type RSSResult = toolweb.RSSResult

type RSSFeedInfo = toolweb.RSSFeedInfo

type RSSItem = toolweb.RSSItem

type WebSearchTool = toolweb.WebSearchTool

type WebRooterTool = toolweb.WebRooterTool

type ImageGenerateTool = toolweb.ImageGenerateTool

func NewWebSearchTool(cfg WebSearchConfig) Tool {
	return toolweb.NewWebSearchTool(cfg)
}

func NewWebRooterTool(cfg WebRooterConfig) Tool {
	return toolweb.NewWebRooterTool(cfg)
}

func NewImageGenerateTool(cfg ImageGenerateConfig, store *artifacts.SessionArtifactStore) Tool {
	return toolweb.NewImageGenerateTool(cfg, store)
}

func FetchRSS(ctx context.Context, rawURL string, opts RSSFetchOptions) (RSSResult, error) {
	return toolweb.FetchRSS(ctx, rawURL, opts)
}
