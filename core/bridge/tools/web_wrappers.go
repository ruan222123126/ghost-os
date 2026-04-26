package tools

import (
	"context"

	toolweb "ghost-os/bridge/tools/web"
)

type WebSearchConfig = toolweb.WebSearchConfig

type RSSFetchOptions = toolweb.RSSFetchOptions

type RSSResult = toolweb.RSSResult

type RSSFeedInfo = toolweb.RSSFeedInfo

type RSSItem = toolweb.RSSItem

type WebSearchTool = toolweb.WebSearchTool

func NewWebSearchTool(cfg WebSearchConfig) Tool {
	return toolweb.NewWebSearchTool(cfg)
}

func FetchRSS(ctx context.Context, rawURL string, opts RSSFetchOptions) (RSSResult, error) {
	return toolweb.FetchRSS(ctx, rawURL, opts)
}
