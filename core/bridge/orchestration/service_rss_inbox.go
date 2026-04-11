package orchestration

import (
	"context"
	"errors"
	"net/http"

	bridgerss "ghost-os/bridge/rss"
)

// RSS action dispatch methods on bridgeService delegate to rssHandler.
// All business logic lives in bridge/rss.ActionHandler.

func (s *bridgeService) executeRSSInboxPollAction(ctx context.Context, params bridgerss.InboxPollParams, traceID string) (any, int, error) {
	handler, code, err := s.requireRSSHandler()
	if err != nil {
		return nil, code, err
	}
	return handler.ExecuteInboxPollAction(ctx, params, traceID)
}

func (s *bridgeService) executeRSSInboxPollUsecase(ctx context.Context, params bridgerss.InboxPollParams, taskID string, traceID string) (bridgerss.RSSInboxPollResult, int, error) {
	handler, code, err := s.requireRSSHandler()
	if err != nil {
		return bridgerss.RSSInboxPollResult{}, code, err
	}
	return handler.ExecuteInboxPollUsecase(ctx, params, taskID, traceID)
}

func (s *bridgeService) executeRSSInboxListAction(params bridgerss.InboxListParams, traceID string) (any, int, error) {
	handler, code, err := s.requireRSSHandler()
	if err != nil {
		return nil, code, err
	}
	return handler.ExecuteInboxListAction(params, traceID)
}

func (s *bridgeService) executeRSSInboxGetAction(params bridgerss.InboxGetParams, traceID string) (any, int, error) {
	handler, code, err := s.requireRSSHandler()
	if err != nil {
		return nil, code, err
	}
	return handler.ExecuteInboxGetAction(params, traceID)
}

func (s *bridgeService) executeRSSInboxGroupsAction(params bridgerss.InboxGroupsParams, traceID string) (any, int, error) {
	handler, code, err := s.requireRSSHandler()
	if err != nil {
		return nil, code, err
	}
	return handler.ExecuteInboxGroupsAction(params, traceID)
}

func (s *bridgeService) executeRSSBriefingBuildAction(ctx context.Context, params bridgerss.BriefingParams, traceID string) (any, int, error) {
	handler, code, err := s.requireRSSHandler()
	if err != nil {
		return nil, code, err
	}
	return handler.ExecuteBriefingBuildAction(ctx, params, traceID)
}

func (s *bridgeService) executeRSSBriefingGetAction(traceID string) (any, int, error) {
	handler, code, err := s.requireRSSHandler()
	if err != nil {
		return nil, code, err
	}
	return handler.ExecuteBriefingGetAction(traceID)
}

func (s *bridgeService) requireRSSHandler() (*bridgerss.ActionHandler, int, error) {
	handler := s.rssActionHandler()
	if handler == nil {
		return nil, http.StatusInternalServerError, errors.New("rss inbox service is not configured")
	}
	return handler, http.StatusOK, nil
}
