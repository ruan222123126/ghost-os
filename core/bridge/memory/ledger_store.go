package memory

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"ghost-os/bridge/llm"
)

type LedgerStore struct {
	baseDir     string
	namespace   string
	workspaceID string
	metrics     *memoryCounters
	mu          sync.Mutex
}

func NewLedgerStore(baseDir string, namespace string, workspaceID string, metrics *memoryCounters) *LedgerStore {
	return &LedgerStore{
		baseDir:     resolveMemoryPath(baseDir),
		namespace:   normalizeLedgerNamespace(namespace),
		workspaceID: strings.TrimSpace(workspaceID),
		metrics:     metrics,
	}
}

func (s *LedgerStore) BaseDir() string {
	if s == nil {
		return ""
	}
	return s.baseDir
}

func (s *LedgerStore) Namespace() string {
	if s == nil {
		return defaultLedgerNamespace
	}
	return normalizeLedgerNamespace(s.namespace)
}

func (s *LedgerStore) WorkspaceID() string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(s.workspaceID)
}

func (s *LedgerStore) AppendTurn(sessionID string, turnID string, traceID string, startIndex int, messages []llm.Message, occurredAt time.Time) (LedgerAppendResult, error) {
	if s == nil || s.baseDir == "" || len(messages) == 0 {
		return LedgerAppendResult{}, nil
	}
	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		return LedgerAppendResult{}, fmt.Errorf("session id is required")
	}
	if startIndex < 0 {
		startIndex = 0
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	} else {
		occurredAt = occurredAt.UTC()
	}
	resolvedTurnID := strings.TrimSpace(turnID)
	if resolvedTurnID == "" {
		resolvedTurnID = buildLedgerTurnID(traceID, startIndex)
	}

	events := make([]LedgerEvent, 0, len(messages)+1)
	for index, message := range messages {
		messageIndex := startIndex + index
		cloned := llm.CloneMessages([]llm.Message{message})
		if len(cloned) == 0 {
			continue
		}
		messageCopy := cloned[0]
		events = append(events, normalizeLedgerEvent(LedgerEvent{
			TraceID:      strings.TrimSpace(traceID),
			Namespace:    s.Namespace(),
			WorkspaceID:  s.WorkspaceID(),
			SessionID:    resolvedSessionID,
			TurnID:       resolvedTurnID,
			MessageIndex: messageIndex,
			Kind:         ledgerEventKindMessageAppended,
			OccurredAt:   occurredAt.Add(time.Duration(index) * time.Millisecond),
			WrittenAt:    time.Now().UTC(),
			DedupeKey:    ledgerMessageDedupeKey(resolvedSessionID, messageIndex, messageCopy),
			Payload:      LedgerPayload{Message: &messageCopy},
		}))
	}
	commitEvent := normalizeLedgerEvent(LedgerEvent{
		TraceID:     strings.TrimSpace(traceID),
		Namespace:   s.Namespace(),
		WorkspaceID: s.WorkspaceID(),
		SessionID:   resolvedSessionID,
		TurnID:      resolvedTurnID,
		Kind:        ledgerEventKindTurnCommitted,
		OccurredAt:  occurredAt.Add(time.Duration(len(messages)) * time.Millisecond),
		WrittenAt:   time.Now().UTC(),
		DedupeKey:   ledgerTurnDedupeKey(resolvedSessionID, resolvedTurnID, traceID, startIndex, len(messages)),
		Payload: LedgerPayload{
			StartIndex:   startIndex,
			MessageCount: len(messages),
		},
	})
	events = append(events, commitEvent)
	return s.Append(events...)
}

func (s *LedgerStore) Append(events ...LedgerEvent) (LedgerAppendResult, error) {
	if s == nil || s.baseDir == "" || len(events) == 0 {
		return LedgerAppendResult{}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	result := LedgerAppendResult{Requested: len(events)}
	sessionSeen := make(map[string]map[string]struct{}, 4)
	for _, event := range events {
		normalized := normalizeLedgerEvent(event)
		if normalized.SessionID == "" {
			continue
		}
		if _, ok := sessionSeen[normalized.SessionID]; ok {
			continue
		}
		replay, err := s.replaySessionLocked(normalized.SessionID)
		if err != nil {
			return result, err
		}
		dedupes := make(map[string]struct{}, len(replay.Events))
		for _, current := range replay.Events {
			dedupes[current.DedupeKey] = struct{}{}
		}
		sessionSeen[normalized.SessionID] = dedupes
	}

	bucketEvents := make(map[string][]LedgerEvent, 4)
	bucketDirs := make(map[string]string, 4)
	queued := make(map[string]struct{}, len(events))
	for _, event := range events {
		normalized := normalizeLedgerEvent(event)
		if normalized.SessionID == "" || normalized.Kind == "" || normalized.DedupeKey == "" {
			continue
		}
		if _, ok := queued[normalized.DedupeKey]; ok {
			result.Duplicates++
			continue
		}
		if dedupes, ok := sessionSeen[normalized.SessionID]; ok {
			if _, exists := dedupes[normalized.DedupeKey]; exists {
				result.Duplicates++
				continue
			}
			dedupes[normalized.DedupeKey] = struct{}{}
		}
		queued[normalized.DedupeKey] = struct{}{}
		bucketDir := s.monthDir(normalized.Namespace, normalized.WorkspaceID, normalized.OccurredAt)
		bucketKey := normalized.Namespace + "|" + normalized.WorkspaceID + "|" + normalized.OccurredAt.Format("2006-01")
		bucketDirs[bucketKey] = bucketDir
		bucketEvents[bucketKey] = append(bucketEvents[bucketKey], normalized)
	}

	bucketKeys := make([]string, 0, len(bucketEvents))
	for key := range bucketEvents {
		bucketKeys = append(bucketKeys, key)
	}
	sort.Strings(bucketKeys)

	for _, key := range bucketKeys {
		monthDir := bucketDirs[key]
		segmentPath := filepath.Join(monthDir, defaultLedgerSegmentFileName)
		existingEvents, _, err := readLedgerSegment(segmentPath, true)
		if err != nil {
			return result, err
		}
		newEvents := bucketEvents[key]
		if err := appendLedgerSegment(segmentPath, newEvents); err != nil {
			return result, err
		}
		manifest, err := buildLedgerManifest(monthDir, newEvents[0].Namespace, newEvents[0].WorkspaceID, append(existingEvents, newEvents...))
		if err != nil {
			return result, err
		}
		if err := writeLedgerManifest(filepath.Join(monthDir, defaultLedgerManifestName), manifest); err != nil {
			return result, err
		}
		result.Written += len(newEvents)
	}
	if result.Written > 0 {
		result.LastWrite = time.Now().UTC()
		if s.metrics != nil {
			s.metrics.ledgerEventsWritten.Add(uint64(result.Written))
		}
	}
	return result, nil
}

func (s *LedgerStore) monthDir(namespace string, workspaceID string, occurredAt time.Time) string {
	return filepath.Join(
		s.baseDir,
		normalizeLedgerNamespace(namespace),
		ledgerWorkspacePathPart(workspaceID),
		occurredAt.UTC().Format("2006-01"),
	)
}

func (s *LedgerStore) namespaceRoot(namespace string, workspaceID string) string {
	return filepath.Join(s.baseDir, normalizeLedgerNamespace(namespace), ledgerWorkspacePathPart(workspaceID))
}
