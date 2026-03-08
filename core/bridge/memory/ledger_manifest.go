package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"ghost-os/bridge/memory/internal/indexer"
)

type ledgerManifest struct {
	Namespace   string    `json:"namespace"`
	WorkspaceID string    `json:"workspace_id,omitempty"`
	Month       string    `json:"month"`
	SegmentFile string    `json:"segment_file"`
	MinOccurred time.Time `json:"min_occurred_at,omitempty"`
	MaxOccurred time.Time `json:"max_occurred_at,omitempty"`
	SessionIDs  []string  `json:"session_ids,omitempty"`
	Sessions    []indexer.SessionManifest `json:"sessions,omitempty"`
	TopTerms    []string  `json:"top_terms,omitempty"`
	Kinds       []string  `json:"kinds,omitempty"`
	TraceCount  int       `json:"trace_count,omitempty"`
	UserTurns   int       `json:"user_turns,omitempty"`
	AssistantTurns int    `json:"assistant_turns,omitempty"`
	Count       int       `json:"count"`
	FirstOffset int64     `json:"first_offset,omitempty"`
	LastOffset  int64     `json:"last_offset,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

func readLedgerManifest(path string) (ledgerManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ledgerManifest{}, nil
		}
		return ledgerManifest{}, fmt.Errorf("read ledger manifest: %w", err)
	}
	var manifest ledgerManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ledgerManifest{}, fmt.Errorf("decode ledger manifest: %w", err)
	}
	manifest.Namespace = normalizeLedgerNamespace(manifest.Namespace)
	manifest.WorkspaceID = strings.TrimSpace(manifest.WorkspaceID)
	manifest.Month = strings.TrimSpace(manifest.Month)
	manifest.SegmentFile = strings.TrimSpace(manifest.SegmentFile)
	manifest.SessionIDs = uniqueStrings(manifest.SessionIDs)
	manifest.Sessions = indexer.NormalizeSessionManifests(manifest.Sessions)
	manifest.TopTerms = uniqueStrings(manifest.TopTerms)
	manifest.Kinds = uniqueStrings(manifest.Kinds)
	manifest.MinOccurred = normalizeLedgerTime(manifest.MinOccurred)
	manifest.MaxOccurred = normalizeLedgerTime(manifest.MaxOccurred)
	manifest.UpdatedAt = normalizeLedgerTime(manifest.UpdatedAt)
	return manifest, nil
}

func writeLedgerManifest(path string, manifest ledgerManifest) error {
	normalized := manifest
	normalized.Namespace = normalizeLedgerNamespace(normalized.Namespace)
	normalized.WorkspaceID = strings.TrimSpace(normalized.WorkspaceID)
	normalized.Month = strings.TrimSpace(normalized.Month)
	if normalized.SegmentFile == "" {
		normalized.SegmentFile = defaultLedgerSegmentFileName
	}
	normalized.SessionIDs = uniqueStrings(normalized.SessionIDs)
	normalized.Sessions = indexer.NormalizeSessionManifests(normalized.Sessions)
	normalized.TopTerms = uniqueStrings(normalized.TopTerms)
	normalized.Kinds = uniqueStrings(normalized.Kinds)
	normalized.MinOccurred = normalizeLedgerTime(normalized.MinOccurred)
	normalized.MaxOccurred = normalizeLedgerTime(normalized.MaxOccurred)
	normalized.UpdatedAt = time.Now().UTC()
	return writeJSONAtomic(path, normalized)
}

func rebuildLedgerManifest(monthDir string, namespace string, workspaceID string) (ledgerManifest, error) {
	segmentPath := filepath.Join(monthDir, defaultLedgerSegmentFileName)
	events, repaired, err := readLedgerSegment(segmentPath, true)
	if err != nil {
		return ledgerManifest{}, err
	}
	manifest, err := buildLedgerManifest(monthDir, namespace, workspaceID, events)
	if err != nil {
		return ledgerManifest{}, err
	}
	if repaired || manifest.Count > 0 || manifest.UpdatedAt.IsZero() {
		if err := writeLedgerManifest(filepath.Join(monthDir, defaultLedgerManifestName), manifest); err != nil {
			return ledgerManifest{}, err
		}
	}
	return manifest, nil
}

func buildLedgerManifest(monthDir string, namespace string, workspaceID string, events []LedgerEvent) (ledgerManifest, error) {
	monthName := filepath.Base(monthDir)
	sessionIDs := make([]string, 0, len(events))
	sessionStats := make(map[string]*indexer.SessionManifest, len(events))
	kindCounts := make(map[string]int, 4)
	termCounts := make(map[string]int, len(events)*4)
	traceIDs := make(map[string]struct{}, len(events))
	var minOccurred time.Time
	var maxOccurred time.Time
	for _, event := range events {
		sessionIDs = append(sessionIDs, strings.TrimSpace(event.SessionID))
		kind := strings.TrimSpace(event.Kind)
		if kind != "" {
			kindCounts[kind]++
		}
		if traceID := strings.TrimSpace(event.TraceID); traceID != "" {
			traceIDs[traceID] = struct{}{}
		}
		stats := sessionStats[strings.TrimSpace(event.SessionID)]
		if stats == nil {
			stats = &indexer.SessionManifest{SessionID: strings.TrimSpace(event.SessionID), FirstOffset: eventOffsetGuess(event), LastOffset: eventOffsetGuess(event)}
			sessionStats[stats.SessionID] = stats
		}
		stats.Count++
		if stats.MinOccurredAt.IsZero() || event.OccurredAt.Before(stats.MinOccurredAt) {
			stats.MinOccurredAt = event.OccurredAt
		}
		if stats.MaxOccurredAt.IsZero() || event.OccurredAt.After(stats.MaxOccurredAt) {
			stats.MaxOccurredAt = event.OccurredAt
		}
		if stats.FirstOffset == 0 || eventOffsetGuess(event) < stats.FirstOffset {
			stats.FirstOffset = eventOffsetGuess(event)
		}
		if eventOffsetGuess(event) > stats.LastOffset {
			stats.LastOffset = eventOffsetGuess(event)
		}
		if traceID := strings.TrimSpace(event.TraceID); traceID != "" {
			stats.TraceCount++
		}
		stats.Kinds = append(stats.Kinds, kind)
		if event.Payload.Message != nil {
			for _, term := range ledgerTopTerms(messageToContent(*event.Payload.Message)) {
				termCounts[term]++
			}
			for _, term := range ledgerTopTerms(messageToContent(*event.Payload.Message)) {
				stats.TopTerms = append(stats.TopTerms, term)
			}
			switch event.Payload.Message.Role {
			case "user":
				stats.UserTurns++
			case "assistant":
				stats.AssistantTurns++
			}
		}
		if minOccurred.IsZero() || event.OccurredAt.Before(minOccurred) {
			minOccurred = event.OccurredAt
		}
		if maxOccurred.IsZero() || event.OccurredAt.After(maxOccurred) {
			maxOccurred = event.OccurredAt
		}
	}
	firstOffset, lastOffset, err := ledgerSegmentOffsetRange(filepath.Join(monthDir, defaultLedgerSegmentFileName))
	if err != nil {
		return ledgerManifest{}, err
	}
	manifest := ledgerManifest{
		Namespace:   normalizeLedgerNamespace(namespace),
		WorkspaceID: strings.TrimSpace(workspaceID),
		Month:       monthName,
		SegmentFile: defaultLedgerSegmentFileName,
		MinOccurred: normalizeLedgerTime(minOccurred),
		MaxOccurred: normalizeLedgerTime(maxOccurred),
		SessionIDs:  uniqueStrings(sessionIDs),
		Sessions:    buildLedgerSessionManifests(sessionStats),
		TopTerms:    ledgerTopCountKeys(termCounts, 12),
		Kinds:       ledgerTopCountKeys(kindCounts, 8),
		TraceCount:  len(traceIDs),
		UserTurns:   ledgerSessionRoleCount(sessionStats, true),
		AssistantTurns: ledgerSessionRoleCount(sessionStats, false),
		Count:       len(events),
		FirstOffset: firstOffset,
		LastOffset:  lastOffset,
		UpdatedAt:   time.Now().UTC(),
	}
	sort.Strings(manifest.SessionIDs)
	return manifest, nil
}

func buildLedgerSessionManifests(stats map[string]*indexer.SessionManifest) []indexer.SessionManifest {
	if len(stats) == 0 {
		return nil
	}
	out := make([]indexer.SessionManifest, 0, len(stats))
	for _, item := range stats {
		normalized := indexer.NormalizeSessionManifest(*item)
		normalized.TopTerms = ledgerTopCountKeys(ledgerCountTerms(normalized.TopTerms), 8)
		out = append(out, normalized)
	}
	return indexer.NormalizeSessionManifests(out)
}

func ledgerSessionRoleCount(stats map[string]*indexer.SessionManifest, user bool) int {
	total := 0
	for _, item := range stats {
		if user {
			total += item.UserTurns
			continue
		}
		total += item.AssistantTurns
	}
	return total
}

func ledgerTopTerms(text string) []string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(text)), func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune(",.!?:;()[]{}\"'`|/\\_-", r)
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if len(trimmed) < 3 {
			continue
		}
		switch trimmed {
		case "the", "and", "for", "with", "that", "this", "from", "have", "will", "your", "about":
			continue
		}
		out = append(out, trimmed)
	}
	return uniqueStrings(out)
}

func ledgerCountTerms(values []string) map[string]int {
	out := make(map[string]int, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(strings.ToLower(value))
		if trimmed == "" {
			continue
		}
		out[trimmed]++
	}
	return out
}

func ledgerTopCountKeys(counts map[string]int, limit int) []string {
	if len(counts) == 0 {
		return nil
	}
	type pair struct {
		Key   string
		Count int
	}
	pairs := make([]pair, 0, len(counts))
	for key, count := range counts {
		pairs = append(pairs, pair{Key: strings.TrimSpace(key), Count: count})
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].Count != pairs[j].Count {
			return pairs[i].Count > pairs[j].Count
		}
		return pairs[i].Key < pairs[j].Key
	})
	if limit > 0 && len(pairs) > limit {
		pairs = pairs[:limit]
	}
	out := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		if pair.Key == "" {
			continue
		}
		out = append(out, pair.Key)
	}
	return out
}

func eventOffsetGuess(event LedgerEvent) int64 {
	return int64(event.MessageIndex)
}

func ledgerManifestMatchesRange(manifest ledgerManifest, timeRange *TimeRange) bool {
	if timeRange == nil {
		return true
	}
	if !timeRange.Start.IsZero() && !manifest.MaxOccurred.IsZero() && manifest.MaxOccurred.Before(timeRange.Start.UTC()) {
		return false
	}
	if !timeRange.End.IsZero() && !manifest.MinOccurred.IsZero() && manifest.MinOccurred.After(timeRange.End.UTC()) {
		return false
	}
	return true
}

func ledgerManifestHasSession(manifest ledgerManifest, sessionID string) bool {
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return false
	}
	for _, current := range manifest.SessionIDs {
		if current == target {
			return true
		}
	}
	return false
}
