package rss

import (
	"errors"
	"fmt"
	"strings"
	"time"
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
