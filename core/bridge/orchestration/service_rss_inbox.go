package orchestration

import (
	"context"

	bridgerss "ghost-os/bridge/rss"
)

// RSS action dispatch methods on bridgeService delegate to rssHandler.
// All business logic lives in bridge/rss.ActionHandler.

func (s *bridgeService) executeRSSInboxPollAction(ctx context.Context, params bridgerss.InboxPollParams, traceID string) (any, int, error) {
	return s.rssHandler.ExecuteInboxPollAction(ctx, params, traceID)
}

func (s *bridgeService) executeRSSInboxPollUsecase(ctx context.Context, params bridgerss.InboxPollParams, taskID string, traceID string) (bridgerss.RSSInboxPollResult, int, error) {
	return s.rssHandler.ExecuteInboxPollUsecase(ctx, params, taskID, traceID)
}

func (s *bridgeService) executeRSSInboxListAction(params bridgerss.InboxListParams, traceID string) (any, int, error) {
	return s.rssHandler.ExecuteInboxListAction(params, traceID)
}

func (s *bridgeService) executeRSSInboxGetAction(params bridgerss.InboxGetParams, traceID string) (any, int, error) {
	return s.rssHandler.ExecuteInboxGetAction(params, traceID)
}

func (s *bridgeService) executeRSSInboxGroupsAction(params bridgerss.InboxGroupsParams, traceID string) (any, int, error) {
	return s.rssHandler.ExecuteInboxGroupsAction(params, traceID)
}

func (s *bridgeService) executeRSSBriefingBuildAction(ctx context.Context, params bridgerss.BriefingParams, traceID string) (any, int, error) {
	return s.rssHandler.ExecuteBriefingBuildAction(ctx, params, traceID)
}

func (s *bridgeService) executeRSSBriefingGetAction(traceID string) (any, int, error) {
	return s.rssHandler.ExecuteBriefingGetAction(traceID)
}
