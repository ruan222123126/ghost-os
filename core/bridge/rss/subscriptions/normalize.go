package subscriptions

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	feedPriorityRankLow    = 1
	feedPriorityRankNormal = 2
	feedPriorityRankHigh   = 3
	feedIDDigestBytes      = 6
)

func resolveFeedStorePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("feed store path is empty")
	}
	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		return resolveTildePath(trimmed)
	}
	return filepath.Clean(trimmed), nil
}

func resolveTildePath(path string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	if path == "~" {
		return homeDir, nil
	}
	return filepath.Join(homeDir, strings.TrimPrefix(path, "~/")), nil
}

func normalizeFeedURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("url is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	parsed.Fragment = ""
	return parsed.String(), nil
}

func normalizeFeedPriority(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "low":
		return "low", nil
	case "", defaultFeedPriority:
		return defaultFeedPriority, nil
	case "high":
		return "high", nil
	default:
		return "", fmt.Errorf("priority must be one of low, normal, or high")
	}
}

func normalizeFeedTags(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		tag := strings.TrimSpace(value)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, tag)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

func mergeFeedTags(existing []string, incoming []string) []string {
	if len(incoming) == 0 {
		return normalizeFeedTags(existing)
	}
	merged := append(cloneStringSlice(existing), incoming...)
	return normalizeFeedTags(merged)
}

func sortFeedSubscriptions(feeds []FeedSubscription) {
	sort.SliceStable(feeds, func(i, j int) bool {
		left := feeds[i]
		right := feeds[j]
		if left.Enabled != right.Enabled {
			return left.Enabled
		}
		leftPriority := feedPriorityRank(left.Priority)
		rightPriority := feedPriorityRank(right.Priority)
		if leftPriority != rightPriority {
			return leftPriority > rightPriority
		}
		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}
		leftLabel := strings.ToLower(feedFirstNonEmpty(left.Title, left.URL))
		rightLabel := strings.ToLower(feedFirstNonEmpty(right.Title, right.URL))
		if leftLabel != rightLabel {
			return leftLabel < rightLabel
		}
		return left.ID < right.ID
	})
}

func feedPriorityRank(priority string) int {
	switch priority {
	case "high":
		return feedPriorityRankHigh
	case defaultFeedPriority:
		return feedPriorityRankNormal
	case "low":
		return feedPriorityRankLow
	default:
		return 0
	}
}

func newFeedID(urlValue string) string {
	sum := sha256.Sum256([]byte(urlValue))
	return "feed-" + hex.EncodeToString(sum[:feedIDDigestBytes])
}

func feedHasTag(feed FeedSubscription, wanted string) bool {
	for _, tag := range feed.Tags {
		if strings.EqualFold(strings.TrimSpace(tag), wanted) {
			return true
		}
	}
	return false
}

func cloneFeeds(raw []FeedSubscription) []FeedSubscription {
	if len(raw) == 0 {
		return nil
	}
	out := make([]FeedSubscription, 0, len(raw))
	for _, item := range raw {
		out = append(out, cloneFeed(item))
	}
	return out
}

func cloneFeed(feed FeedSubscription) FeedSubscription {
	feed.Tags = cloneStringSlice(feed.Tags)
	return feed
}

func cloneStringSlice(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	out := make([]string, len(raw))
	copy(out, raw)
	return out
}

func equalStringSlices(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func feedFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
