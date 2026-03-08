package memory

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"ghost-os/bridge/memory/internal/indexer"
	"ghost-os/bridge/llm"
)

func (s *LedgerStore) ListBucketManifests(checkpointDir string) ([]BucketManifest, error) {
	if s == nil || s.baseDir == "" {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	monthDirs, err := listLedgerMonthDirs(s.namespaceRoot(s.Namespace(), s.WorkspaceID()))
	if err != nil {
		return nil, err
	}
	viewStore := indexer.NewFileCheckpointStore(checkpointDir)
	manifests := make([]BucketManifest, 0, len(monthDirs))
	for _, monthDir := range monthDirs {
		manifest, err := s.ensureManifestLocked(monthDir)
		if err != nil {
			return nil, err
		}
		bucketManifest := bucketManifestFromLedger(monthDir, manifest)
		if viewStore != nil {
			bucket := indexer.Bucket{Namespace: bucketManifest.Key.Namespace, Workspace: bucketManifest.Key.WorkspaceID, Month: bucketManifest.Key.Month, RootDir: monthDir, SegmentPath: filepath.Join(monthDir, defaultLedgerSegmentFileName)}
			for _, projector := range []string{"graph", "decision", "markdown", "vector"} {
				view, err := viewStore.LoadViewManifest(projector, bucket)
				if err != nil || view.Projector == "" {
					continue
				}
				bucketManifest.Views[projector] = BucketViewManifest{
					Projector:       view.Projector,
					UpdatedAt:       view.UpdatedAt,
					SessionCoverage: append([]string(nil), view.SessionCoverage...),
					EvidenceCount:   view.EvidenceCount,
					Graph:           view.Graph,
					Decision:        view.Decision,
					Markdown:        view.Markdown,
					Vector:          view.Vector,
				}
			}
		}
		manifests = append(manifests, bucketManifest)
	}
	sort.SliceStable(manifests, func(i, j int) bool {
		if manifests[i].Key.Month != manifests[j].Key.Month {
			return manifests[i].Key.Month > manifests[j].Key.Month
		}
		return manifests[i].Key.String() < manifests[j].Key.String()
	})
	return manifests, nil
}

func (s *LedgerStore) ReplaySession(sessionID string) (LedgerReplaySession, error) {
	if s == nil || s.baseDir == "" {
		return LedgerReplaySession{SessionID: strings.TrimSpace(sessionID)}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.replaySessionLocked(sessionID)
}

func (s *LedgerStore) replaySessionLocked(sessionID string) (LedgerReplaySession, error) {
	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		return LedgerReplaySession{}, nil
	}
	monthDirs, err := listLedgerMonthDirs(s.namespaceRoot(s.Namespace(), s.WorkspaceID()))
	if err != nil {
		return LedgerReplaySession{}, err
	}
	events := make([]LedgerEvent, 0, 32)
	for _, monthDir := range monthDirs {
		manifest, err := s.ensureManifestLocked(monthDir)
		if err != nil {
			return LedgerReplaySession{}, err
		}
		if manifest.Count == 0 || !ledgerManifestHasSession(manifest, resolvedSessionID) {
			continue
		}
		segmentEvents, _, err := readLedgerSegment(filepath.Join(monthDir, defaultLedgerSegmentFileName), true)
		if err != nil {
			return LedgerReplaySession{}, err
		}
		for _, event := range segmentEvents {
			if strings.TrimSpace(event.SessionID) == resolvedSessionID {
				events = append(events, event)
			}
		}
	}
	return projectLedgerSession(events), nil
}

func (s *LedgerStore) ReplayRange(timeRange *TimeRange) ([]LedgerEvent, error) {
	if s == nil || s.baseDir == "" {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	monthDirs, err := listLedgerMonthDirs(s.namespaceRoot(s.Namespace(), s.WorkspaceID()))
	if err != nil {
		return nil, err
	}
	events := make([]LedgerEvent, 0, 64)
	for _, monthDir := range monthDirs {
		manifest, err := s.ensureManifestLocked(monthDir)
		if err != nil {
			return nil, err
		}
		if manifest.Count == 0 || !ledgerManifestMatchesRange(manifest, timeRange) {
			continue
		}
		segmentEvents, _, err := readLedgerSegment(filepath.Join(monthDir, defaultLedgerSegmentFileName), true)
		if err != nil {
			return nil, err
		}
		for _, event := range segmentEvents {
			if timeRange != nil && !timeRange.Contains(event.OccurredAt) {
				continue
			}
			events = append(events, event)
		}
	}
	sortLedgerEvents(events)
	return events, nil
}

func (s *LedgerStore) ReplayArchivesCompat(timeRange *TimeRange) ([]ColdArchive, error) {
	sessions, err := s.replaySessionsCompat(timeRange)
	if err != nil {
		return nil, err
	}
	out := make([]ColdArchive, 0, len(sessions))
	for _, session := range sessions {
		out = append(out, ColdArchive{
			SessionID:  session.SessionID,
			ArchivedAt: session.ArchivedAt,
			Messages:   llm.CloneMessages(session.Messages),
		})
	}
	return out, nil
}

func (s *LedgerStore) ReplayArchivesCompatWithPlan(timeRange *TimeRange, plan *BucketPlan) ([]ColdArchive, error) {
	if plan == nil || len(plan.SelectedBuckets) == 0 {
		return s.ReplayArchivesCompat(timeRange)
	}
	sessions, err := s.replaySessionsCompatWithPlan(timeRange, plan)
	if err != nil {
		return nil, err
	}
	out := make([]ColdArchive, 0, len(sessions))
	for _, session := range sessions {
		out = append(out, ColdArchive{
			SessionID:  session.SessionID,
			ArchivedAt: session.ArchivedAt,
			Messages:   llm.CloneMessages(session.Messages),
		})
	}
	return out, nil
}

func (s *LedgerStore) RetrieveCompat(query MemoryQuery) ([]MemoryEntry, error) {
	archives, err := s.ReplayArchivesCompat(query.TimeRange)
	if err != nil {
		return nil, err
	}
	results := make([]MemoryEntry, 0, len(archives)*4)
	for _, archive := range archives {
		for index, msg := range archive.Messages {
			entry := normalizeEntry(MemoryEntry{
				ID:         fmt.Sprintf("%s:%06d", archive.SessionID, index),
				Content:    messageToContent(msg),
				Type:       MemoryTypeMessage,
				Timestamp:  archive.ArchivedAt,
				Source:     "archive",
				Summary:    summarizeLine(messageToContent(msg), 220),
				Confidence: 1,
				Metadata: map[string]any{
					"layer":        "cold",
					"source":       "archive",
					"session_id":   archive.SessionID,
					"role":         string(msg.Role),
					"tool_call_id": strings.TrimSpace(msg.ToolCallID),
				},
			})
			if !entryMatchesQuery(entry, query) {
				continue
			}
			results = append(results, entry)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}
	return cloneEntries(results), nil
}

func (s *LedgerStore) RetrieveCompatWithPlan(query MemoryQuery, plan *BucketPlan) ([]MemoryEntry, error) {
	if plan == nil || len(plan.SelectedBuckets) == 0 {
		return s.RetrieveCompat(query)
	}
	archives, err := s.ReplayArchivesCompatWithPlan(query.TimeRange, plan)
	if err != nil {
		return nil, err
	}
	results := make([]MemoryEntry, 0, len(archives)*4)
	for _, archive := range archives {
		for index, msg := range archive.Messages {
			entry := normalizeEntry(MemoryEntry{
				ID:         fmt.Sprintf("%s:%06d", archive.SessionID, index),
				Content:    messageToContent(msg),
				Type:       MemoryTypeMessage,
				Timestamp:  archive.ArchivedAt,
				Source:     "archive",
				Summary:    summarizeLine(messageToContent(msg), 220),
				Confidence: 1,
				Metadata: map[string]any{
					"layer":        "cold",
					"source":       "archive",
					"session_id":   archive.SessionID,
					"role":         string(msg.Role),
					"tool_call_id": strings.TrimSpace(msg.ToolCallID),
					"bucket_month": bucketMonthFromTime(archive.ArchivedAt),
				},
			})
			if !entryMatchesQuery(entry, query) {
				continue
			}
			results = append(results, entry)
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}
	return cloneEntries(results), nil
}

func (s *LedgerStore) ListSessionsCompat(timeRange TimeRange) ([]string, error) {
	archives, err := s.ReplayArchivesCompat(&timeRange)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(archives))
	out := make([]string, 0, len(archives))
	for _, archive := range archives {
		if _, ok := seen[archive.SessionID]; ok {
			continue
		}
		seen[archive.SessionID] = struct{}{}
		out = append(out, archive.SessionID)
	}
	sort.Strings(out)
	return out, nil
}

func (s *LedgerStore) replaySessionsCompat(timeRange *TimeRange) ([]LedgerReplaySession, error) {
	if s == nil || s.baseDir == "" {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	monthDirs, err := listLedgerMonthDirs(s.namespaceRoot(s.Namespace(), s.WorkspaceID()))
	if err != nil {
		return nil, err
	}
	seenSessions := make(map[string]struct{}, 32)
	sessionIDs := make([]string, 0, 32)
	for _, monthDir := range monthDirs {
		manifest, err := s.ensureManifestLocked(monthDir)
		if err != nil {
			return nil, err
		}
		if manifest.Count == 0 || !ledgerManifestMatchesRange(manifest, timeRange) {
			continue
		}
		for _, sessionID := range manifest.SessionIDs {
			if _, ok := seenSessions[sessionID]; ok {
				continue
			}
			seenSessions[sessionID] = struct{}{}
			sessionIDs = append(sessionIDs, sessionID)
		}
	}
	sort.Strings(sessionIDs)

	sessions := make([]LedgerReplaySession, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		replay, err := s.replaySessionLocked(sessionID)
		if err != nil {
			return nil, err
		}
		if replay.SessionID == "" {
			continue
		}
		if timeRange != nil && !timeRange.Contains(replay.ArchivedAt) {
			continue
		}
		sessions = append(sessions, replay)
	}
	sort.SliceStable(sessions, func(i, j int) bool {
		if sessions[i].ArchivedAt.Equal(sessions[j].ArchivedAt) {
			return sessions[i].SessionID < sessions[j].SessionID
		}
		return sessions[i].ArchivedAt.After(sessions[j].ArchivedAt)
	})
	return sessions, nil
}

func (s *LedgerStore) replaySessionsCompatWithPlan(timeRange *TimeRange, plan *BucketPlan) ([]LedgerReplaySession, error) {
	if s == nil || s.baseDir == "" {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	selected := selectedBucketsByKey(plan)
	if len(selected) == 0 {
		return nil, nil
	}
	seenSessions := make(map[string]struct{}, 32)
	sessionIDs := make([]string, 0, 32)
	for _, manifest := range selected {
		if manifest.Count == 0 || !bucketManifestMatchesRange(manifest, timeRange) {
			continue
		}
		for _, sessionID := range manifest.SessionIDs {
			if _, ok := seenSessions[sessionID]; ok {
				continue
			}
			seenSessions[sessionID] = struct{}{}
			sessionIDs = append(sessionIDs, sessionID)
		}
	}
	sort.Strings(sessionIDs)
	sessions := make([]LedgerReplaySession, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		replay, err := s.replaySessionAcrossBucketsLocked(sessionID, selected)
		if err != nil {
			return nil, err
		}
		if replay.SessionID == "" {
			continue
		}
		if timeRange != nil && !timeRange.Contains(replay.ArchivedAt) {
			continue
		}
		sessions = append(sessions, replay)
	}
	sort.SliceStable(sessions, func(i, j int) bool {
		if sessions[i].ArchivedAt.Equal(sessions[j].ArchivedAt) {
			return sessions[i].SessionID < sessions[j].SessionID
		}
		return sessions[i].ArchivedAt.After(sessions[j].ArchivedAt)
	})
	return sessions, nil
}

func (s *LedgerStore) replaySessionAcrossBucketsLocked(sessionID string, selected map[string]BucketManifest) (LedgerReplaySession, error) {
	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		return LedgerReplaySession{}, nil
	}
	events := make([]LedgerEvent, 0, 32)
	for _, bucket := range selected {
		segmentEvents, _, err := readLedgerSegment(filepath.Join(bucket.RootDir, defaultLedgerSegmentFileName), true)
		if err != nil {
			return LedgerReplaySession{}, err
		}
		for _, event := range segmentEvents {
			if strings.TrimSpace(event.SessionID) == resolvedSessionID {
				events = append(events, event)
			}
		}
	}
	return projectLedgerSession(events), nil
}

func selectedBucketsByKey(plan *BucketPlan) map[string]BucketManifest {
	if plan == nil || len(plan.SelectedBuckets) == 0 {
		return nil
	}
	out := make(map[string]BucketManifest, len(plan.SelectedBuckets))
	for _, selected := range plan.SelectedBuckets {
		out[selected.Bucket.Key.String()] = normalizeBucketManifest(selected.Bucket)
	}
	return out
}

func (s *LedgerStore) ensureManifestLocked(monthDir string) (ledgerManifest, error) {
	manifestPath := filepath.Join(monthDir, defaultLedgerManifestName)
	manifest, err := readLedgerManifest(manifestPath)
	if err != nil {
		return ledgerManifest{}, err
	}
	if manifest.Count == 0 && manifest.Month == "" {
		return rebuildLedgerManifest(monthDir, s.Namespace(), s.WorkspaceID())
	}
	return manifest, nil
}

func projectLedgerSession(events []LedgerEvent) LedgerReplaySession {
	sortLedgerEvents(events)
	messageByIndex := make(map[int]llm.Message, len(events))
	orderedIndexes := make([]int, 0, len(events))
	seenIndexes := make(map[int]struct{}, len(events))
	replay := LedgerReplaySession{}
	for _, event := range events {
		event = normalizeLedgerEvent(event)
		if replay.SessionID == "" {
			replay.Namespace = event.Namespace
			replay.WorkspaceID = event.WorkspaceID
			replay.SessionID = event.SessionID
		}
		switch event.Kind {
		case ledgerEventKindMessageAppended:
			if event.Payload.Message == nil {
				continue
			}
			if _, ok := messageByIndex[event.MessageIndex]; ok {
				continue
			}
			cloned := llm.CloneMessages([]llm.Message{*event.Payload.Message})
			if len(cloned) == 0 {
				continue
			}
			messageByIndex[event.MessageIndex] = cloned[0]
			if _, ok := seenIndexes[event.MessageIndex]; !ok {
				seenIndexes[event.MessageIndex] = struct{}{}
				orderedIndexes = append(orderedIndexes, event.MessageIndex)
			}
		case ledgerEventKindTurnCommitted:
			if replay.ArchivedAt.IsZero() || event.OccurredAt.After(replay.ArchivedAt) {
				replay.ArchivedAt = event.OccurredAt
			}
		case ledgerEventKindSessionSnapshotImported:
			archivedAt := event.Payload.ArchivedAt
			if archivedAt.IsZero() && event.LegacyRef != nil {
				archivedAt = event.LegacyRef.ArchivedAt
			}
			if replay.ArchivedAt.IsZero() || (!archivedAt.IsZero() && archivedAt.After(replay.ArchivedAt)) {
				replay.ArchivedAt = archivedAt
			}
		}
		replay.Events = append(replay.Events, event)
	}
	if replay.ArchivedAt.IsZero() && len(events) > 0 {
		replay.ArchivedAt = events[len(events)-1].OccurredAt
	}
	sort.Ints(orderedIndexes)
	replay.Messages = make([]llm.Message, 0, len(orderedIndexes))
	for _, index := range orderedIndexes {
		replay.Messages = append(replay.Messages, messageByIndex[index])
	}
	return replay
}

func sortLedgerEvents(events []LedgerEvent) {
	sort.SliceStable(events, func(i, j int) bool {
		left := normalizeLedgerEvent(events[i])
		right := normalizeLedgerEvent(events[j])
		if !left.OccurredAt.Equal(right.OccurredAt) {
			return left.OccurredAt.Before(right.OccurredAt)
		}
		if left.MessageIndex != right.MessageIndex {
			return left.MessageIndex < right.MessageIndex
		}
		if !left.WrittenAt.Equal(right.WrittenAt) {
			return left.WrittenAt.Before(right.WrittenAt)
		}
		return left.EventID < right.EventID
	})
}
