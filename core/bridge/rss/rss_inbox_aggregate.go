package rss

import "fmt"

func (s *RSSInboxService) Aggregate(query RSSInboxGroupQuery) (RSSInboxGroupResult, error) {
	if s == nil || s.inboxStore == nil {
		return RSSInboxGroupResult{}, fmt.Errorf("rss inbox service is not configured")
	}

	spec := newRSSInboxAggregateSpec(query, s.currentTime())
	items, err := s.inboxStore.List(spec.listFilter())
	if err != nil {
		return RSSInboxGroupResult{}, err
	}
	return newRSSInboxAggregator(spec).Run(items), nil
}
