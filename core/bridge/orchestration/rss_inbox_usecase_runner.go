package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var errInvalidRSSInboxParam = errors.New("invalid rss inbox params")

type rssInboxUsecaseRunner struct {
	inbox *RSSInboxService
}

func (s *bridgeService) requireRSSInboxRunner() (rssInboxUsecaseRunner, int, error) {
	inbox, code, err := s.requireRSSInbox()
	if err != nil {
		return rssInboxUsecaseRunner{}, code, err
	}
	return rssInboxUsecaseRunner{inbox: inbox}, code, nil
}

func (r rssInboxUsecaseRunner) Poll(
	ctx context.Context,
	params rssInboxPollParams,
	taskID string,
	traceID string,
) (RSSInboxPollResult, error) {
	return r.inbox.Poll(ctx, RSSInboxPollOptions{
		MaxItemsPerFeed: params.MaxItemsPerFeed,
		AIBatchSize:     params.AIBatchSize,
		TraceID:         strings.TrimSpace(traceID),
		TaskID:          strings.TrimSpace(taskID),
	})
}

func (r rssInboxUsecaseRunner) List(params rssInboxListParams) ([]RSSInboxItem, error) {
	filter, err := decodeRSSInboxListFilter(params)
	if err != nil {
		return nil, err
	}
	return r.inbox.List(filter)
}

func (r rssInboxUsecaseRunner) Get(params rssInboxGetParams) (RSSInboxItem, error) {
	id := strings.TrimSpace(params.ID)
	if id == "" {
		return RSSInboxItem{}, wrapRSSInboxParamError(errors.New("rss inbox item id is required"))
	}
	return r.inbox.Get(id)
}

func (r rssInboxUsecaseRunner) Groups(params rssInboxGroupsParams) (RSSInboxGroupResult, error) {
	return r.inbox.Aggregate(RSSInboxGroupQuery{
		FeedID:        strings.TrimSpace(params.FeedID),
		Tag:           strings.TrimSpace(params.Tag),
		Importance:    strings.TrimSpace(params.Importance),
		WindowHours:   params.WindowHours,
		Limit:         params.Limit,
		ItemLimit:     params.ItemLimit,
		ItemsPerGroup: params.ItemsPerGroup,
	})
}

func (r rssInboxUsecaseRunner) BuildBriefing(
	ctx context.Context,
	params rssBriefingParams,
	traceID string,
) (RSSBriefingResult, error) {
	return r.inbox.BuildAndStoreBriefing(ctx, RSSBriefingQuery{
		FeedID:          strings.TrimSpace(params.FeedID),
		Tag:             strings.TrimSpace(params.Tag),
		Importance:      strings.TrimSpace(params.Importance),
		WindowHours:     params.WindowHours,
		GroupLimit:      params.GroupLimit,
		ItemLimit:       params.ItemLimit,
		ItemsPerGroup:   params.ItemsPerGroup,
		HighlightsLimit: params.HighlightsLimit,
		TraceID:         firstNonEmptyString(strings.TrimSpace(traceID), strings.TrimSpace(params.TraceID)),
		TaskID:          strings.TrimSpace(params.TaskID),
	})
}

func (r rssInboxUsecaseRunner) LatestBriefing() (RSSBriefingResult, error) {
	return r.inbox.LatestBriefing()
}

func decodeRSSInboxListFilter(params rssInboxListParams) (RSSInboxListFilter, error) {
	filter := RSSInboxListFilter{
		FeedID:     strings.TrimSpace(params.FeedID),
		Tag:        strings.TrimSpace(params.Tag),
		Importance: strings.TrimSpace(params.Importance),
		Limit:      params.Limit,
	}

	var err error
	if filter.SavedAfter, err = parseOptionalTimeParam(params.SavedAfter, "saved_after"); err != nil {
		return RSSInboxListFilter{}, err
	}
	if filter.SavedBefore, err = parseOptionalTimeParam(params.SavedBefore, "saved_before"); err != nil {
		return RSSInboxListFilter{}, err
	}
	if filter.PublishedAfter, err = parseOptionalTimeParam(params.PublishedAfter, "published_after"); err != nil {
		return RSSInboxListFilter{}, err
	}
	if filter.PublishedBefore, err = parseOptionalTimeParam(params.PublishedBefore, "published_before"); err != nil {
		return RSSInboxListFilter{}, err
	}
	return filter, nil
}

func parseOptionalTimeParam(raw string, field string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, wrapRSSInboxParamError(fmt.Errorf("%s must use RFC3339", field))
	}
	return parsed.UTC(), nil
}

func wrapRSSInboxParamError(err error) error {
	if err == nil || errors.Is(err, errInvalidRSSInboxParam) {
		return err
	}
	return fmt.Errorf("%w: %v", errInvalidRSSInboxParam, err)
}
