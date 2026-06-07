package sessions

import (
	"sort"
	"strings"

	"ghost-os/bridge/session"
)

func searchSessionSummaries(
	summaries []session.SessionMetadata,
	query string,
	limit int,
	partitionState session.SessionSidebarPartitionState,
) []session.SessionMetadata {
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	partitionSessionIDs := matchingPartitionSessionIDs(partitionState, normalizedQuery)
	matches := make([]session.SessionMetadata, 0, len(summaries))
	for _, summary := range summaries {
		if sessionSummaryMatches(summary, normalizedQuery, partitionSessionIDs) {
			matches = append(matches, summary)
		}
	}

	sort.SliceStable(matches, func(leftIndex, rightIndex int) bool {
		left := matches[leftIndex]
		right := matches[rightIndex]
		if !left.UpdatedAt.Equal(right.UpdatedAt) {
			return left.UpdatedAt.After(right.UpdatedAt)
		}
		if !left.CreatedAt.Equal(right.CreatedAt) {
			return left.CreatedAt.After(right.CreatedAt)
		}
		return left.ID > right.ID
	})

	if limit > 0 && len(matches) > limit {
		return matches[:limit]
	}
	return matches
}

func sessionSummaryMatches(
	summary session.SessionMetadata,
	normalizedQuery string,
	partitionSessionIDs map[string]struct{},
) bool {
	if normalizedQuery == "" {
		return true
	}
	if _, ok := partitionSessionIDs[summary.ID]; ok {
		return true
	}
	for _, value := range []string{summary.ID, summary.Title} {
		if strings.Contains(strings.ToLower(strings.TrimSpace(value)), normalizedQuery) {
			return true
		}
	}
	return false
}

func matchingPartitionSessionIDs(
	state session.SessionSidebarPartitionState,
	normalizedQuery string,
) map[string]struct{} {
	if normalizedQuery == "" {
		return nil
	}

	partitionIDs := make(map[string]struct{})
	for _, partition := range state.Partitions {
		if strings.Contains(strings.ToLower(strings.TrimSpace(partition.Name)), normalizedQuery) {
			id := strings.TrimSpace(partition.ID)
			if id != "" {
				partitionIDs[id] = struct{}{}
			}
		}
	}
	if len(partitionIDs) == 0 {
		return nil
	}

	sessionIDs := make(map[string]struct{})
	for sessionID, partitionID := range state.Assignments {
		if _, ok := partitionIDs[strings.TrimSpace(partitionID)]; !ok {
			continue
		}
		id := strings.TrimSpace(sessionID)
		if id != "" {
			sessionIDs[id] = struct{}{}
		}
	}
	return sessionIDs
}
