package rss

import (
	"errors"
	"fmt"
	"strings"
	"time"

	bridgeTasks "ghost-os/bridge/tasks"
)

// decodeInboxListFilter converts API params to a store filter.
func decodeInboxListFilter(params InboxListParams) (RSSInboxListFilter, error) {
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

// DecodeInboxPollParams decodes and validates poll params from a map.
func DecodeInboxPollParams(input map[string]any) (InboxPollParams, error) {
	params, err := bridgeTasks.DecodeParamsMap[InboxPollParams](input)
	if err != nil {
		return InboxPollParams{}, err
	}
	if params.MaxItemsPerFeed < 0 {
		return InboxPollParams{}, fmt.Errorf("max_items_per_feed must be >= 0")
	}
	if params.AIBatchSize < 0 {
		return InboxPollParams{}, fmt.Errorf("ai_batch_size must be >= 0")
	}
	return params, nil
}

// InboxPollParamsToMap converts poll params to a map for storage.
func InboxPollParamsToMap(params InboxPollParams) map[string]any {
	out := map[string]any{}
	if params.MaxItemsPerFeed > 0 {
		out["max_items_per_feed"] = params.MaxItemsPerFeed
	}
	if params.AIBatchSize > 0 {
		out["ai_batch_size"] = params.AIBatchSize
	}
	return out
}

// DecodeBriefingParams decodes and validates briefing params from a map.
func DecodeBriefingParams(input map[string]any) (BriefingParams, error) {
	params, err := bridgeTasks.DecodeParamsMap[BriefingParams](input)
	if err != nil {
		return BriefingParams{}, err
	}
	if params.WindowHours < 0 || params.GroupLimit < 0 || params.ItemLimit < 0 {
		return BriefingParams{}, fmt.Errorf("briefing limits must be >= 0")
	}
	if params.ItemsPerGroup < 0 || params.HighlightsLimit < 0 {
		return BriefingParams{}, fmt.Errorf("briefing limits must be >= 0")
	}
	return params, nil
}

// BriefingParamsToMap converts briefing params to a map for storage.
func BriefingParamsToMap(params BriefingParams) map[string]any {
	out := map[string]any{}
	if params.FeedID != "" {
		out["feed_id"] = strings.TrimSpace(params.FeedID)
	}
	if params.Tag != "" {
		out["tag"] = strings.TrimSpace(params.Tag)
	}
	if params.Importance != "" {
		out["importance"] = strings.TrimSpace(params.Importance)
	}
	if params.WindowHours > 0 {
		out["window_hours"] = params.WindowHours
	}
	if params.GroupLimit > 0 {
		out["group_limit"] = params.GroupLimit
	}
	if params.ItemLimit > 0 {
		out["item_limit"] = params.ItemLimit
	}
	if params.ItemsPerGroup > 0 {
		out["items_per_group"] = params.ItemsPerGroup
	}
	if params.HighlightsLimit > 0 {
		out["highlights_limit"] = params.HighlightsLimit
	}
	return out
}
