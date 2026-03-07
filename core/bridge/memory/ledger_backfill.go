package memory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

type ledgerBackfillCheckpoint struct {
	Namespace       string    `json:"namespace,omitempty"`
	WorkspaceID     string    `json:"workspace_id,omitempty"`
	CompletedMonths []string  `json:"completed_months,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

func (s *LedgerStore) BackfillLegacy(legacy *LegacyColdStore, opts LedgerBackfillOptions) (LedgerBackfillStats, error) {
	stats := LedgerBackfillStats{
		Namespace:   normalizeLedgerNamespace(firstNonEmpty(opts.Namespace, s.Namespace())),
		WorkspaceID: strings.TrimSpace(firstNonEmpty(opts.WorkspaceID, s.WorkspaceID())),
	}
	if s == nil || s.baseDir == "" || legacy == nil {
		return stats, nil
	}
	records, err := legacy.ListArchiveRecords(opts.TimeRange)
	if err != nil {
		return stats, err
	}
	if len(records) == 0 {
		return stats, nil
	}
	checkpoint, err := s.readBackfillCheckpoint(stats.Namespace, stats.WorkspaceID)
	if err != nil {
		return stats, err
	}
	completed := make(map[string]struct{}, len(checkpoint.CompletedMonths))
	for _, month := range checkpoint.CompletedMonths {
		completed[month] = struct{}{}
	}
	months := groupLegacyRecordsByMonth(records)
	monthKeys := make([]string, 0, len(months))
	for month := range months {
		monthKeys = append(monthKeys, month)
	}
	sort.Strings(monthKeys)
	stats.MonthsScanned = len(monthKeys)

	for _, month := range monthKeys {
		if _, ok := completed[month]; ok {
			continue
		}
		monthRecords := months[month]
		for _, record := range monthRecords {
			stats.ArchivesScanned++
			events := buildLedgerBackfillEvents(stats.Namespace, stats.WorkspaceID, month, record)
			stats.EventsRequested += len(events)
			if opts.DryRun {
				continue
			}
			appendResult, err := s.Append(events...)
			if err != nil {
				return stats, err
			}
			stats.EventsWritten += appendResult.Written
			stats.DuplicateEvents += appendResult.Duplicates
			if s.metrics != nil {
				s.metrics.ledgerBackfillProgress.Add(1)
			}
		}
		stats.MonthsCompleted++
		if opts.DryRun {
			continue
		}
		checkpoint.CompletedMonths = append(checkpoint.CompletedMonths, month)
		checkpoint.UpdatedAt = time.Now().UTC()
		if err := s.writeBackfillCheckpoint(stats.Namespace, stats.WorkspaceID, checkpoint); err != nil {
			return stats, err
		}
		stats.CheckpointsWritten++
	}
	return stats, nil
}

func buildLedgerBackfillEvents(namespace string, workspaceID string, month string, record LegacyArchiveRecord) []LedgerEvent {
	archive := record.Archive
	sessionID := strings.TrimSpace(archive.SessionID)
	turnID := "legacy-import:" + sessionID + ":" + strings.TrimSpace(month)
	legacyRef := &LedgerLegacyRef{
		MonthDir:   strings.TrimSpace(month),
		SessionID:  sessionID,
		ArchivedAt: archive.ArchivedAt.UTC(),
		SourcePath: strings.TrimSpace(record.Path),
	}
	events := make([]LedgerEvent, 0, len(archive.Messages)+1)
	events = append(events, normalizeLedgerEvent(LedgerEvent{
		Namespace:   normalizeLedgerNamespace(namespace),
		WorkspaceID: strings.TrimSpace(workspaceID),
		SessionID:   sessionID,
		TurnID:      turnID,
		Kind:        ledgerEventKindSessionSnapshotImported,
		OccurredAt:  archive.ArchivedAt.UTC(),
		WrittenAt:   time.Now().UTC(),
		DedupeKey:   ledgerImportDedupeKey(sessionID, month, archive.ArchivedAt),
		Payload: LedgerPayload{
			ArchivedAt:   archive.ArchivedAt.UTC(),
			ImportedAt:   time.Now().UTC(),
			MessageCount: len(archive.Messages),
		},
		LegacyRef: legacyRef,
	}))
	for index, message := range archive.Messages {
		cloned := llm.CloneMessages([]llm.Message{message})
		if len(cloned) == 0 {
			continue
		}
		messageCopy := cloned[0]
		messageRef := *legacyRef
		messageRef.MessageIndex = index
		events = append(events, normalizeLedgerEvent(LedgerEvent{
			Namespace:    normalizeLedgerNamespace(namespace),
			WorkspaceID:  strings.TrimSpace(workspaceID),
			SessionID:    sessionID,
			TurnID:       turnID,
			MessageIndex: index,
			Kind:         ledgerEventKindMessageAppended,
			OccurredAt:   archive.ArchivedAt.UTC().Add(time.Duration(index) * time.Millisecond),
			WrittenAt:    time.Now().UTC(),
			DedupeKey:    ledgerMessageDedupeKey(sessionID, index, messageCopy),
			Payload: LedgerPayload{
				Message: &messageCopy,
			},
			LegacyRef: &messageRef,
		}))
	}
	return events
}

func groupLegacyRecordsByMonth(records []LegacyArchiveRecord) map[string][]LegacyArchiveRecord {
	out := make(map[string][]LegacyArchiveRecord, len(records))
	for _, record := range records {
		month := strings.TrimSpace(record.MonthDir)
		out[month] = append(out[month], record)
	}
	return out
}

func (s *LedgerStore) readBackfillCheckpoint(namespace string, workspaceID string) (ledgerBackfillCheckpoint, error) {
	path := s.backfillCheckpointPath(namespace, workspaceID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ledgerBackfillCheckpoint{Namespace: namespace, WorkspaceID: workspaceID}, nil
		}
		return ledgerBackfillCheckpoint{}, err
	}
	var checkpoint ledgerBackfillCheckpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return ledgerBackfillCheckpoint{}, err
	}
	checkpoint.Namespace = normalizeLedgerNamespace(firstNonEmpty(checkpoint.Namespace, namespace))
	checkpoint.WorkspaceID = firstNonEmpty(strings.TrimSpace(checkpoint.WorkspaceID), strings.TrimSpace(workspaceID))
	checkpoint.CompletedMonths = uniqueStrings(checkpoint.CompletedMonths)
	checkpoint.UpdatedAt = normalizeLedgerTime(checkpoint.UpdatedAt)
	return checkpoint, nil
}

func (s *LedgerStore) writeBackfillCheckpoint(namespace string, workspaceID string, checkpoint ledgerBackfillCheckpoint) error {
	checkpoint.Namespace = normalizeLedgerNamespace(namespace)
	checkpoint.WorkspaceID = strings.TrimSpace(workspaceID)
	checkpoint.CompletedMonths = uniqueStrings(checkpoint.CompletedMonths)
	checkpoint.UpdatedAt = time.Now().UTC()
	return writeJSONAtomic(s.backfillCheckpointPath(namespace, workspaceID), checkpoint)
}

func (s *LedgerStore) backfillCheckpointPath(namespace string, workspaceID string) string {
	return filepath.Join(s.namespaceRoot(namespace, workspaceID), "_backfill", defaultLedgerCheckpointName)
}
