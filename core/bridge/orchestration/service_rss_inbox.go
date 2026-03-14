package orchestration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

const (
	busActionRSSInboxPoll     = "RSS_INBOX_POLL"
	busActionRSSInboxList     = "RSS_INBOX_LIST"
	busActionRSSInboxGet      = "RSS_INBOX_GET"
	busActionRSSInboxGroups   = "RSS_INBOX_GROUPS"
	busActionRSSBriefingBuild = "RSS_BRIEFING_BUILD"
	busActionRSSBriefingGet   = "RSS_BRIEFING_GET"
)

type rssInboxPollParams struct {
	MaxItemsPerFeed int    `json:"max_items_per_feed,omitempty"`
	AIBatchSize     int    `json:"ai_batch_size,omitempty"`
	TraceID         string `json:"trace_id,omitempty"`
}

type rssInboxListParams struct {
	FeedID          string `json:"feed_id,omitempty"`
	Tag             string `json:"tag,omitempty"`
	Importance      string `json:"importance,omitempty"`
	SavedAfter      string `json:"saved_after,omitempty"`
	SavedBefore     string `json:"saved_before,omitempty"`
	PublishedAfter  string `json:"published_after,omitempty"`
	PublishedBefore string `json:"published_before,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

type rssInboxGetParams struct {
	ID string `json:"id"`
}

type rssInboxGroupsParams struct {
	FeedID        string `json:"feed_id,omitempty"`
	Tag           string `json:"tag,omitempty"`
	Importance    string `json:"importance,omitempty"`
	WindowHours   int    `json:"window_hours,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	ItemLimit     int    `json:"item_limit,omitempty"`
	ItemsPerGroup int    `json:"items_per_group,omitempty"`
}

type rssBriefingParams struct {
	FeedID          string `json:"feed_id,omitempty"`
	Tag             string `json:"tag,omitempty"`
	Importance      string `json:"importance,omitempty"`
	WindowHours     int    `json:"window_hours,omitempty"`
	GroupLimit      int    `json:"group_limit,omitempty"`
	ItemLimit       int    `json:"item_limit,omitempty"`
	ItemsPerGroup   int    `json:"items_per_group,omitempty"`
	HighlightsLimit int    `json:"highlights_limit,omitempty"`
	TraceID         string `json:"trace_id,omitempty"`
	TaskID          string `json:"task_id,omitempty"`
}

func (s *bridgeService) requireRSSInbox() (*RSSInboxService, int, error) {
	if s == nil || s.rssInbox == nil {
		if s != nil && s.rssInitErr != nil {
			return nil, http.StatusInternalServerError, s.rssInitErr
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("rss inbox service is not configured")
	}
	return s.rssInbox, http.StatusOK, nil
}

func mapRSSInboxUsecaseError(err error) int {
	switch {
	case errors.Is(err, errInvalidRSSInboxParam):
		return http.StatusBadRequest
	case errors.Is(err, ErrRSSInboxItemNotFound), errors.Is(err, ErrRSSBriefingNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func (s *bridgeService) executeRSSInboxPollAction(ctx context.Context, params rssInboxPollParams, traceID string) (any, int, error) {
	return s.executeRSSInboxPollUsecase(ctx, params, "", traceID)
}

func (s *bridgeService) executeRSSInboxPollUsecase(
	ctx context.Context,
	params rssInboxPollParams,
	taskID string,
	traceID string,
) (RSSInboxPollResult, int, error) {
	runner, code, err := s.requireRSSInboxRunner()
	if err != nil {
		return RSSInboxPollResult{}, code, err
	}
	result, err := runner.Poll(ctx, params, taskID, traceID)
	if err != nil {
		logAction(traceID, busActionRSSInboxPoll, "error", err)
		return RSSInboxPollResult{}, mapRSSInboxUsecaseError(err), err
	}
	logAction(traceID, busActionRSSInboxPoll, "success", nil)
	return result, http.StatusOK, nil
}

func (s *bridgeService) executeRSSInboxListAction(params rssInboxListParams, traceID string) (any, int, error) {
	runner, code, err := s.requireRSSInboxRunner()
	if err != nil {
		return nil, code, err
	}
	result, err := runner.List(params)
	if err != nil {
		logAction(traceID, busActionRSSInboxList, "error", err)
		return nil, mapRSSInboxUsecaseError(err), err
	}
	logAction(traceID, busActionRSSInboxList, "success", nil)
	return result, http.StatusOK, nil
}

func (s *bridgeService) executeRSSInboxGetAction(params rssInboxGetParams, traceID string) (any, int, error) {
	runner, code, err := s.requireRSSInboxRunner()
	if err != nil {
		return nil, code, err
	}
	result, err := runner.Get(params)
	if err != nil {
		logAction(traceID, busActionRSSInboxGet, "error", err)
		return nil, mapRSSInboxUsecaseError(err), err
	}
	logAction(traceID, busActionRSSInboxGet, "success", nil)
	return result, http.StatusOK, nil
}

func (s *bridgeService) executeRSSInboxGroupsAction(params rssInboxGroupsParams, traceID string) (any, int, error) {
	runner, code, err := s.requireRSSInboxRunner()
	if err != nil {
		return nil, code, err
	}
	result, err := runner.Groups(params)
	if err != nil {
		logAction(traceID, busActionRSSInboxGroups, "error", err)
		return nil, mapRSSInboxUsecaseError(err), err
	}
	logAction(traceID, busActionRSSInboxGroups, "success", nil)
	return result, http.StatusOK, nil
}

func (s *bridgeService) executeRSSBriefingBuildAction(ctx context.Context, params rssBriefingParams, traceID string) (any, int, error) {
	runner, code, err := s.requireRSSInboxRunner()
	if err != nil {
		return nil, code, err
	}
	result, err := runner.BuildBriefing(ctx, params, traceID)
	if err != nil {
		logAction(traceID, busActionRSSBriefingBuild, "error", err)
		return nil, mapRSSInboxUsecaseError(err), err
	}
	logAction(traceID, busActionRSSBriefingBuild, "success", nil)
	return result, http.StatusOK, nil
}

func (s *bridgeService) executeRSSBriefingGetAction(traceID string) (any, int, error) {
	runner, code, err := s.requireRSSInboxRunner()
	if err != nil {
		return nil, code, err
	}
	result, err := runner.LatestBriefing()
	if err != nil {
		logAction(traceID, busActionRSSBriefingGet, "error", err)
		return nil, mapRSSInboxUsecaseError(err), err
	}
	logAction(traceID, busActionRSSBriefingGet, "success", nil)
	return result, http.StatusOK, nil
}

func (s *bridgeService) ensureRSSPollTask() error {
	return newRSSSystemTaskCoordinator(s).syncPollTask()
}

func (s *bridgeService) ensureRSSBriefingTask() error {
	return newRSSSystemTaskCoordinator(s).syncBriefingTask()
}
