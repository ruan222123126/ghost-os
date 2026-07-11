package rss

import (
	"sort"
	"strings"
)

func shouldSkipItem(item Item, seenIDs map[string]struct{}, seenLinks map[string]struct{}) bool {
	if id := strings.TrimSpace(item.ID); id != "" {
		if _, exists := seenIDs[id]; exists {
			return true
		}
		seenIDs[id] = struct{}{}
	}
	if link := strings.TrimSpace(item.Link); link != "" {
		if _, exists := seenLinks[link]; exists {
			return true
		}
		seenLinks[link] = struct{}{}
	}
	return false
}

func finalizeItems(items []sortableItem) []Item {
	sort.SliceStable(items, func(i, j int) bool {
		left := items[i]
		right := items[j]
		if left.hasPublished && right.hasPublished {
			if left.publishedTime.Equal(right.publishedTime) {
				return left.index < right.index
			}
			return left.publishedTime.After(right.publishedTime)
		}
		if left.hasPublished != right.hasPublished {
			return left.hasPublished
		}
		return left.index < right.index
	})

	out := make([]Item, 0, len(items))
	for _, entry := range items {
		out = append(out, entry.item)
	}
	return out
}
