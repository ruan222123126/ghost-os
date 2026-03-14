package subscriptions

import (
	"strings"
	"time"
)

type upsertValues struct {
	url         string
	title       string
	probeTitle  string
	priority    string
	tags        []string
	enabled     *bool
	prioritySet bool
}

func (s *FeedStore) upsertExistingLocked(
	feeds []FeedSubscription,
	values upsertValues,
	now time.Time,
) (FeedSubscription, string, bool, error) {
	for index := range feeds {
		if feeds[index].URL != values.url {
			continue
		}
		if !mergeUpsertIntoFeed(&feeds[index], values) {
			return cloneFeed(feeds[index]), "exists", true, nil
		}
		feeds[index].UpdatedAt = now
		if err := s.saveLocked(feeds); err != nil {
			return FeedSubscription{}, "", true, err
		}
		return cloneFeed(feeds[index]), "updated_existing", true, nil
	}
	return FeedSubscription{}, "", false, nil
}

func normalizeFeedUpsertInput(input FeedUpsertInput) (upsertValues, error) {
	urlValue, err := normalizeFeedURL(input.URL)
	if err != nil {
		return upsertValues{}, err
	}
	values := upsertValues{
		url:        urlValue,
		title:      strings.TrimSpace(input.Title),
		probeTitle: strings.TrimSpace(input.ProbeTitle),
		tags:       normalizeFeedTags(input.Tags),
		priority:   defaultFeedPriority,
		enabled:    input.Enabled,
	}
	if strings.TrimSpace(input.Priority) == "" {
		return values, nil
	}
	values.priority, err = normalizeFeedPriority(input.Priority)
	if err != nil {
		return upsertValues{}, err
	}
	values.prioritySet = true
	return values, nil
}

func mergeUpsertIntoFeed(feed *FeedSubscription, values upsertValues) bool {
	changed := updateFeedTitle(feed, values.title, values.probeTitle)
	mergedTags := mergeFeedTags(feed.Tags, values.tags)
	if !equalStringSlices(feed.Tags, mergedTags) {
		feed.Tags = mergedTags
		changed = true
	}
	if values.prioritySet && feed.Priority != values.priority {
		feed.Priority = values.priority
		changed = true
	}
	if values.enabled != nil && feed.Enabled != *values.enabled {
		feed.Enabled = *values.enabled
		changed = true
	}
	return changed
}

func updateFeedTitle(feed *FeedSubscription, title string, probeTitle string) bool {
	switch {
	case title != "" && feed.Title != title:
		feed.Title = title
		return true
	case feed.Title == "" && probeTitle != "":
		feed.Title = probeTitle
		return true
	default:
		return false
	}
}

func newFeedSubscription(values upsertValues, now time.Time) FeedSubscription {
	created := FeedSubscription{
		ID:        newFeedID(values.url),
		URL:       values.url,
		Title:     feedFirstNonEmpty(values.title, values.probeTitle),
		Tags:      values.tags,
		Priority:  values.priority,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if values.enabled != nil {
		created.Enabled = *values.enabled
	}
	return created
}

func applyFeedUpdate(feed FeedSubscription, patch FeedUpdatePatch) (FeedSubscription, bool, error) {
	changed := false
	if patch.Title != nil {
		title := strings.TrimSpace(*patch.Title)
		if feed.Title != title {
			feed.Title = title
			changed = true
		}
	}
	if patch.Tags != nil {
		tags := normalizeFeedTags(*patch.Tags)
		if !equalStringSlices(feed.Tags, tags) {
			feed.Tags = tags
			changed = true
		}
	}
	if patch.Priority != nil {
		priority, err := normalizeFeedPriority(*patch.Priority)
		if err != nil {
			return FeedSubscription{}, false, err
		}
		if feed.Priority != priority {
			feed.Priority = priority
			changed = true
		}
	}
	if patch.Enabled != nil && feed.Enabled != *patch.Enabled {
		feed.Enabled = *patch.Enabled
		changed = true
	}
	return feed, changed, nil
}

func filterFeedSubscriptions(feeds []FeedSubscription, filter FeedListFilter) ([]FeedSubscription, error) {
	priority, err := normalizedFilterPriority(filter.Priority)
	if err != nil {
		return nil, err
	}
	tag := strings.ToLower(strings.TrimSpace(filter.Tag))
	result := make([]FeedSubscription, 0, len(feeds))
	for _, feed := range feeds {
		if filter.Enabled != nil && feed.Enabled != *filter.Enabled {
			continue
		}
		if priority != "" && feed.Priority != priority {
			continue
		}
		if tag != "" && !feedHasTag(feed, tag) {
			continue
		}
		result = append(result, cloneFeed(feed))
	}
	sortFeedSubscriptions(result)
	return result, nil
}

func normalizedFilterPriority(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	return normalizeFeedPriority(raw)
}

func normalizeLoadedFeeds(raw []FeedSubscription) []FeedSubscription {
	feeds := make([]FeedSubscription, 0, len(raw))
	for _, feed := range raw {
		if normalized, ok := normalizeLoadedFeed(feed); ok {
			feeds = append(feeds, normalized)
		}
	}
	return feeds
}

func normalizeLoadedFeed(feed FeedSubscription) (FeedSubscription, bool) {
	if strings.TrimSpace(feed.ID) == "" || strings.TrimSpace(feed.URL) == "" {
		return FeedSubscription{}, false
	}
	urlValue, err := normalizeFeedURL(feed.URL)
	if err != nil {
		return FeedSubscription{}, false
	}
	priority, err := normalizeFeedPriority(feed.Priority)
	if err != nil {
		priority = defaultFeedPriority
	}
	feed.URL = urlValue
	feed.Title = strings.TrimSpace(feed.Title)
	feed.Tags = normalizeFeedTags(feed.Tags)
	feed.Priority = priority
	return feed, true
}
