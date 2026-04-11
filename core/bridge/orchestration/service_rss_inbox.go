package orchestration

import "context"

// RSS action dispatch methods on bridgeService delegate to rssHandler.
// All business logic lives in bridge/rss.ActionHandler.

func (s *bridgeService) executeRSSInboxPollAction(ctx context.Context, params RSSInboxPollParams, traceID string) (any, int, error) {
	return s.rssHandler.ExecuteInboxPollAction(ctx, params, traceID)
}

func (s *bridgeService) executeRSSInboxPollUsecase(ctx context.Context, params RSSInboxPollParams, taskID string, traceID string) (RSSInboxPollResult, int, error) {
	return s.rssHandler.ExecuteInboxPollUsecase(ctx, params, taskID, traceID)
}

func (s *bridgeService) executeRSSInboxListAction(params RSSInboxListParams, traceID string) (any, int, error) {
	return s.rssHandler.ExecuteInboxListAction(params, traceID)
}

func (s *bridgeService) executeRSSInboxGetAction(params RSSInboxGetParams, traceID string) (any, int, error) {
	return s.rssHandler.ExecuteInboxGetAction(params, traceID)
}

func (s *bridgeService) executeRSSInboxGroupsAction(params RSSInboxGroupsParams, traceID string) (any, int, error) {
	return s.rssHandler.ExecuteInboxGroupsAction(params, traceID)
}

func (s *bridgeService) executeRSSBriefingBuildAction(ctx context.Context, params RSSBriefingParams, traceID string) (any, int, error) {
	return s.rssHandler.ExecuteBriefingBuildAction(ctx, params, traceID)
}

func (s *bridgeService) executeRSSBriefingGetAction(traceID string) (any, int, error) {
	return s.rssHandler.ExecuteBriefingGetAction(traceID)
}
