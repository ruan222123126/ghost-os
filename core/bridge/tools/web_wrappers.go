package tools

import (
	"context"

	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
	toolweb "ghost-os/bridge/tools/web"
)

const webRooterToolName = "web_rooter"

type WebSearchConfig = toolweb.WebSearchConfig

type WebRooterConfig = toolweb.WebRooterConfig

type RSSFetchOptions = toolweb.RSSFetchOptions

type RSSResult = toolweb.RSSResult

type RSSFeedInfo = toolweb.RSSFeedInfo

type RSSItem = toolweb.RSSItem

type WebSearchTool = toolweb.WebSearchTool

type WebRooterTool = toolweb.WebRooterTool

type RSSFetchTool = toolweb.RSSFetchTool

type FeedManageTool = toolweb.FeedManageTool

func NewWebSearchTool(cfg WebSearchConfig) Tool {
	return toolweb.NewWebSearchTool(cfg)
}

func NewWebRooterTool(cfg WebRooterConfig) Tool {
	return toolweb.NewWebRooterTool(cfg)
}

func NewRSSFetchTool() Tool {
	return toolweb.NewRSSFetchTool()
}

func NewFeedManageTool(store *rsssubscriptions.FeedStore) Tool {
	return toolweb.NewFeedManageTool(store)
}

func FetchRSS(ctx context.Context, rawURL string, opts RSSFetchOptions) (RSSResult, error) {
	return toolweb.FetchRSS(ctx, rawURL, opts)
}
