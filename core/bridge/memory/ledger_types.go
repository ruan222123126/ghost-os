package memory

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

const (
	ledgerEventKindMessageAppended         = "message_appended"
	ledgerEventKindTurnCommitted           = "turn_committed"
	ledgerEventKindSessionSnapshotImported = "session_snapshot_imported"

	defaultLedgerNamespace       = "default"
	defaultLedgerSegmentFileName = "events.jsonl"
	defaultLedgerManifestName    = "manifest.json"
	defaultLedgerCheckpointName  = "checkpoint.json"
	ledgerWorkspacePlaceholder   = "_"
)

type LedgerEvent struct {
	EventID      string           `json:"event_id"`
	TraceID      string           `json:"trace_id,omitempty"`
	Namespace    string           `json:"namespace"`
	WorkspaceID  string           `json:"workspace_id,omitempty"`
	SessionID    string           `json:"session_id"`
	TurnID       string           `json:"turn_id,omitempty"`
	MessageIndex int              `json:"message_index,omitempty"`
	Kind         string           `json:"kind"`
	OccurredAt   time.Time        `json:"occurred_at"`
	WrittenAt    time.Time        `json:"written_at"`
	DedupeKey    string           `json:"dedupe_key"`
	Payload      LedgerPayload    `json:"payload,omitempty"`
	LegacyRef    *LedgerLegacyRef `json:"legacy_ref,omitempty"`
}

type LedgerPayload struct {
	Message      *llm.Message `json:"message,omitempty"`
	StartIndex   int          `json:"start_index,omitempty"`
	MessageCount int          `json:"message_count,omitempty"`
	ArchivedAt   time.Time    `json:"archived_at,omitempty"`
	ImportedAt   time.Time    `json:"imported_at,omitempty"`
}

type LedgerLegacyRef struct {
	MonthDir     string    `json:"month_dir,omitempty"`
	SessionID    string    `json:"session_id,omitempty"`
	MessageIndex int       `json:"message_index,omitempty"`
	ArchivedAt   time.Time `json:"archived_at,omitempty"`
	SourcePath   string    `json:"source_path,omitempty"`
}

type LedgerAppendResult struct {
	Requested  int       `json:"requested"`
	Written    int       `json:"written"`
	Duplicates int       `json:"duplicates"`
	LastWrite  time.Time `json:"last_write,omitempty"`
}

type LedgerReplaySession struct {
	Namespace   string        `json:"namespace,omitempty"`
	WorkspaceID string        `json:"workspace_id,omitempty"`
	SessionID   string        `json:"session_id"`
	ArchivedAt  time.Time     `json:"archived_at,omitempty"`
	Messages    []llm.Message `json:"messages,omitempty"`
	Events      []LedgerEvent `json:"events,omitempty"`
}

type LedgerBackfillOptions struct {
	Namespace   string     `json:"namespace,omitempty"`
	WorkspaceID string     `json:"workspace_id,omitempty"`
	TimeRange   *TimeRange `json:"time_range,omitempty"`
	DryRun      bool       `json:"dry_run,omitempty"`
}

type LedgerBackfillStats struct {
	Namespace          string `json:"namespace,omitempty"`
	WorkspaceID        string `json:"workspace_id,omitempty"`
	MonthsScanned      int    `json:"months_scanned"`
	MonthsCompleted    int    `json:"months_completed"`
	ArchivesScanned    int    `json:"archives_scanned"`
	EventsRequested    int    `json:"events_requested"`
	EventsWritten      int    `json:"events_written"`
	DuplicateEvents    int    `json:"duplicate_events"`
	CheckpointsWritten int    `json:"checkpoints_written"`
}

type LedgerRuntimeStats struct {
	BaseDir       string `json:"base_dir,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	WorkspaceID   string `json:"workspace_id,omitempty"`
	DualWrite     bool   `json:"dual_write"`
	ReadEnabled   bool   `json:"read_enabled"`
	ShadowCompare bool   `json:"shadow_compare"`
}

func normalizeLedgerEvent(event LedgerEvent) LedgerEvent {
	out := event
	out.EventID = strings.TrimSpace(out.EventID)
	out.TraceID = strings.TrimSpace(out.TraceID)
	out.Namespace = normalizeLedgerNamespace(out.Namespace)
	out.WorkspaceID = strings.TrimSpace(out.WorkspaceID)
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.TurnID = strings.TrimSpace(out.TurnID)
	out.Kind = strings.TrimSpace(out.Kind)
	out.DedupeKey = strings.TrimSpace(out.DedupeKey)
	if out.OccurredAt.IsZero() {
		out.OccurredAt = time.Now().UTC()
	} else {
		out.OccurredAt = out.OccurredAt.UTC()
	}
	if out.WrittenAt.IsZero() {
		out.WrittenAt = time.Now().UTC()
	} else {
		out.WrittenAt = out.WrittenAt.UTC()
	}
	if out.MessageIndex < 0 {
		out.MessageIndex = 0
	}
	if out.Payload.Message != nil {
		cloned := llm.CloneMessages([]llm.Message{*out.Payload.Message})
		if len(cloned) > 0 {
			out.Payload.Message = &cloned[0]
		}
	}
	out.Payload.ArchivedAt = normalizeLedgerTime(out.Payload.ArchivedAt)
	out.Payload.ImportedAt = normalizeLedgerTime(out.Payload.ImportedAt)
	if out.LegacyRef != nil {
		legacyRef := *out.LegacyRef
		legacyRef.MonthDir = strings.TrimSpace(legacyRef.MonthDir)
		legacyRef.SessionID = strings.TrimSpace(legacyRef.SessionID)
		legacyRef.ArchivedAt = normalizeLedgerTime(legacyRef.ArchivedAt)
		legacyRef.SourcePath = strings.TrimSpace(legacyRef.SourcePath)
		out.LegacyRef = &legacyRef
	}
	if out.EventID == "" && out.Kind != "" && out.DedupeKey != "" {
		out.EventID = buildLedgerEventID(out.Kind, out.DedupeKey)
	}
	return out
}

func normalizeLedgerNamespace(namespace string) string {
	trimmed := strings.TrimSpace(namespace)
	if trimmed == "" {
		return defaultLedgerNamespace
	}
	return trimmed
}

func normalizeLedgerTime(ts time.Time) time.Time {
	if ts.IsZero() {
		return time.Time{}
	}
	return ts.UTC()
}

func ledgerWorkspacePathPart(workspaceID string) string {
	trimmed := strings.TrimSpace(workspaceID)
	if trimmed == "" {
		return ledgerWorkspacePlaceholder
	}
	return trimmed
}

func buildLedgerEventID(kind string, dedupeKey string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(kind), hashLedgerString(strings.TrimSpace(dedupeKey)))
}

func buildLedgerTurnID(traceID string, startIndex int) string {
	trace := strings.TrimSpace(traceID)
	if trace == "" {
		return fmt.Sprintf("turn-%d", startIndex)
	}
	return fmt.Sprintf("turn-%d-%s", startIndex, trace)
}

func ledgerMessageDedupeKey(sessionID string, messageIndex int, message llm.Message) string {
	role := strings.TrimSpace(string(message.Role))
	payloadBytes, err := json.Marshal(llm.CloneMessages([]llm.Message{message}))
	payloadHash := "empty"
	if err == nil {
		payloadHash = hashLedgerString(string(payloadBytes))
	}
	return strings.Join([]string{strings.TrimSpace(sessionID), fmt.Sprintf("%06d", messageIndex), role, payloadHash}, "|")
}

func ledgerTurnDedupeKey(sessionID string, turnID string, traceID string, startIndex int, messageCount int) string {
	return strings.Join([]string{
		strings.TrimSpace(sessionID),
		strings.TrimSpace(turnID),
		strings.TrimSpace(traceID),
		fmt.Sprintf("%06d", startIndex),
		fmt.Sprintf("%06d", messageCount),
	}, "|")
}

func ledgerImportDedupeKey(sessionID string, monthDir string, archivedAt time.Time) string {
	return strings.Join([]string{strings.TrimSpace(sessionID), strings.TrimSpace(monthDir), archivedAt.UTC().Format(time.RFC3339Nano)}, "|")
}

func hashLedgerString(value string) string {
	hash := sha1.Sum([]byte(value))
	return hex.EncodeToString(hash[:])
}
