package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ledgerManifest struct {
	Namespace   string    `json:"namespace"`
	WorkspaceID string    `json:"workspace_id,omitempty"`
	Month       string    `json:"month"`
	SegmentFile string    `json:"segment_file"`
	MinOccurred time.Time `json:"min_occurred_at,omitempty"`
	MaxOccurred time.Time `json:"max_occurred_at,omitempty"`
	SessionIDs  []string  `json:"session_ids,omitempty"`
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
	var minOccurred time.Time
	var maxOccurred time.Time
	for _, event := range events {
		sessionIDs = append(sessionIDs, strings.TrimSpace(event.SessionID))
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
		Count:       len(events),
		FirstOffset: firstOffset,
		LastOffset:  lastOffset,
		UpdatedAt:   time.Now().UTC(),
	}
	sort.Strings(manifest.SessionIDs)
	return manifest, nil
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
