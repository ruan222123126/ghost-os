package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/memory/internal/indexer"
)

type IndexRuntime struct {
	inner *indexer.Runtime
	owns  map[string]bool
}

type memoryLedgerBucketSource struct {
	ledger *LedgerStore
}

type ledgerEventRecord struct {
	Event  LedgerEvent
	Offset int64
}

func NewIndexRuntime(config MemoryConfig, warm *WarmMemory, cold *ColdMemory, graph *GraphService, decision *DecisionService, vector *VectorSidecar, evolver *Evolver, metrics *memoryCounters) *IndexRuntime {
	_ = warm
	_ = vector
	_ = evolver
	_ = metrics
	if cold == nil || cold.ledger == nil {
		return &IndexRuntime{}
	}
	source := &memoryLedgerBucketSource{ledger: cold.ledger}
	checkpointStore := indexer.NewFileCheckpointStore(config.Index.CheckpointDir)
	if checkpointStore == nil {
		return &IndexRuntime{}
	}
	projectors := make([]indexer.Projector, 0, 5)
	owns := make(map[string]bool, 2)
	if projector, live := newGraphIndexProjector(config, cold, graph); projector != nil {
		projectors = append(projectors, projector)
		if live {
			owns[projector.Name()] = true
		}
	}
	if projector, live := newDecisionIndexProjector(config, cold, decision); projector != nil {
		projectors = append(projectors, projector)
		if live {
			owns[projector.Name()] = true
		}
	}
	runtime := indexer.NewRuntime(indexer.Config{
		Enabled:         config.Index.Enabled,
		Workers:         config.Index.Workers,
		BatchSize:       config.Index.BatchSize,
		PollInterval:    config.Index.PollInterval,
		ShadowCompare:   config.Index.ShadowCompare,
		LegacyColdAsync: config.Index.LegacyColdAsync,
	}, source, checkpointStore, projectors...)
	if runtime == nil {
		return &IndexRuntime{}
	}
	return &IndexRuntime{inner: runtime, owns: owns}
}

func newGraphIndexProjector(config MemoryConfig, cold *ColdMemory, graph *GraphService) (indexer.Projector, bool) {
	if graph == nil || !config.Graph.Enabled {
		return nil, false
	}
	target := graph
	live := !config.Index.ShadowCompare
	if config.Index.ShadowCompare {
		shadowConfig := config
		shadowConfig.Graph.Path = indexShadowPath(config.Graph.Path, "graph")
		target = NewGraphService(shadowConfig, cold, config.Runtime.Summarizer)
	}
	if target == nil || !target.Enabled() {
		return nil, false
	}
	return indexer.NewGraphProjector(true, func(ctx context.Context, envelope indexer.TurnEnvelope) error {
		return target.IngestArchiveMessages(envelope.SessionID, llm.CloneMessages(envelope.Messages))
	}, func(ctx context.Context, bucket indexer.Bucket, envelopes []indexer.TurnEnvelope) (indexer.ViewManifest, error) {
		_ = ctx
		stats := target.GraphStats(firstNonEmpty(config.Graph.Namespace, bucket.Namespace))
		sessionCoverage, evidenceCount := bucketEnvelopeCoverage(envelopes)
		return indexer.ViewManifest{
			Projector:       "graph",
			Namespace:       bucket.Namespace,
			Workspace:       bucket.Workspace,
			Month:           bucket.Month,
			UpdatedAt:       stats.UpdatedAt,
			SessionCoverage: sessionCoverage,
			EvidenceCount:   evidenceCount,
			Graph: &indexer.GraphViewManifest{
				NodeCount:       stats.NodeCount,
				EdgeCount:       stats.EdgeCount,
				SessionCoverage: sessionCoverage,
				EvidenceCount:   evidenceCount,
				UpdatedAt:       stats.UpdatedAt,
			},
		}, nil
	}), live
}

func newDecisionIndexProjector(config MemoryConfig, cold *ColdMemory, decision *DecisionService) (indexer.Projector, bool) {
	if !config.Decision.Enabled {
		return nil, false
	}
	shadowOnly := config.Index.ShadowCompare || config.Decision.CaptureOnTurn
	target := decision
	live := !shadowOnly
	if shadowOnly {
		shadowConfig := config
		shadowConfig.Decision.Path = indexShadowPath(config.Decision.Path, "decision")
		shadowConfig.Decision.CaptureOnTurn = true
		shadowConfig.Decision.CaptureOnTurnSet = true
		target = NewDecisionService(shadowConfig, cold)
	}
	if target == nil || !target.Enabled() {
		return nil, false
	}
	return indexer.NewDecisionProjector(true, func(ctx context.Context, envelope indexer.TurnEnvelope) error {
		return target.CaptureTurn(buildDecisionCaptureInputFromEnvelope(config, envelope))
	}, func(ctx context.Context, bucket indexer.Bucket, envelopes []indexer.TurnEnvelope) (indexer.ViewManifest, error) {
		_ = ctx
		stats := target.Stats(firstNonEmpty(config.Graph.Namespace, bucket.Namespace))
		sessionCoverage, evidenceCount := bucketEnvelopeCoverage(envelopes)
		updatedAt := time.Now().UTC()
		return indexer.ViewManifest{
			Projector:       "decision",
			Namespace:       bucket.Namespace,
			Workspace:       bucket.Workspace,
			Month:           bucket.Month,
			UpdatedAt:       updatedAt,
			SessionCoverage: sessionCoverage,
			EvidenceCount:   evidenceCount,
			Decision: &indexer.DecisionViewManifest{
				MemoCount:    stats.MemoCount,
				RecipeCount:  stats.RecipeCount,
				SupportCount: stats.SuccessCount + stats.PartialCount,
				LastHitAt:    updatedAt,
			},
		}, nil
	}), live
}

func bucketEnvelopeCoverage(envelopes []indexer.TurnEnvelope) ([]string, int) {
	sessionIDs := make([]string, 0, len(envelopes))
	evidenceCount := 0
	for _, envelope := range envelopes {
		sessionIDs = append(sessionIDs, strings.TrimSpace(envelope.SessionID))
		evidenceCount += len(envelope.Messages)
	}
	return uniqueStrings(sessionIDs), evidenceCount
}

func buildDecisionCaptureInputFromEnvelope(config MemoryConfig, envelope indexer.TurnEnvelope) DecisionCaptureInput {
	userMessage := firstUserMessageText(envelope.Messages)
	namespace := normalizeDecisionNamespace(firstNonEmpty(config.Graph.Namespace, envelope.Namespace))
	return DecisionCaptureInput{
		Namespace:      namespace,
		SessionID:      strings.TrimSpace(envelope.SessionID),
		TraceID:        strings.TrimSpace(envelope.TraceID),
		TurnID:         strings.TrimSpace(envelope.TurnID),
		UserMessage:    userMessage,
		NewMessages:    llm.CloneMessages(envelope.Messages),
		Outcome:        decisionOutcomeFromEnvelope(envelope.Messages),
		SessionEnded:   turnEnvelopeEndsSession(envelope.Messages),
		TurnStartedAt:  envelope.StartedAt,
		TurnFinishedAt: envelope.CommittedAt,
		Environment: DecisionEnvFingerprint{
			GraphNamespace: namespace,
		},
	}
}

func decisionOutcomeFromEnvelope(messages []llm.Message) string {
	hasAssistant := false
	hasTool := false
	for _, msg := range messages {
		switch msg.Role {
		case llm.RoleAssistant:
			hasAssistant = true
		case llm.RoleTool:
			hasTool = true
		}
	}
	if hasAssistant {
		return DecisionOutcomeSuccess
	}
	if hasTool {
		return DecisionOutcomePartial
	}
	return DecisionOutcomeSuccess
}

func firstUserMessageText(messages []llm.Message) string {
	for _, msg := range messages {
		if msg.Role == llm.RoleUser {
			return strings.TrimSpace(msg.Text)
		}
	}
	return ""
}

func turnEnvelopeEndsSession(messages []llm.Message) bool {
	for _, msg := range messages {
		if msg.Role != llm.RoleAssistant {
			continue
		}
		var payload struct {
			Signal string `json:"signal"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(msg.Text)), &payload); err == nil && strings.EqualFold(strings.TrimSpace(payload.Signal), "END_SESSION") {
			return true
		}
	}
	return false
}

func indexShadowPath(baseDir string, projector string) string {
	trimmedBaseDir := strings.TrimSpace(baseDir)
	trimmedProjector := strings.TrimSpace(projector)
	if trimmedBaseDir == "" {
		return ""
	}
	if trimmedProjector == "" {
		return filepath.Join(trimmedBaseDir, "_index_shadow")
	}
	return filepath.Join(trimmedBaseDir, "_index_shadow", trimmedProjector)
}

func (s *memoryLedgerBucketSource) ListBuckets(ctx context.Context) ([]indexer.Bucket, error) {
	_ = ctx
	if s == nil || s.ledger == nil {
		return nil, nil
	}
	rootDir := s.ledger.namespaceRoot(s.ledger.Namespace(), s.ledger.WorkspaceID())
	monthDirs, err := listLedgerMonthDirs(rootDir)
	if err != nil {
		return nil, err
	}
	buckets := make([]indexer.Bucket, 0, len(monthDirs))
	for _, monthDir := range monthDirs {
		month := filepath.Base(monthDir)
		buckets = append(buckets, indexer.Bucket{
			Namespace:   s.ledger.Namespace(),
			Workspace:   s.ledger.WorkspaceID(),
			Month:       month,
			RootDir:     monthDir,
			SegmentPath: filepath.Join(monthDir, defaultLedgerSegmentFileName),
		})
	}
	return buckets, nil
}

func (s *memoryLedgerBucketSource) LoadBucket(ctx context.Context, bucket indexer.Bucket) ([]indexer.Event, error) {
	_ = ctx
	if s == nil || s.ledger == nil {
		return nil, nil
	}
	records, err := readLedgerSegmentRecords(bucket.SegmentPath, true)
	if err != nil {
		return nil, err
	}
	events := make([]indexer.Event, 0, len(records))
	for _, record := range records {
		event := normalizeLedgerEvent(record.Event)
		payload := indexer.EventPayload{StartIndex: event.Payload.StartIndex, MessageCount: event.Payload.MessageCount}
		if event.Payload.Message != nil {
			cloned := llm.CloneMessages([]llm.Message{*event.Payload.Message})
			if len(cloned) > 0 {
				payload.Message = &cloned[0]
			}
		}
		events = append(events, indexer.Event{
			EventID:      event.EventID,
			TraceID:      event.TraceID,
			Namespace:    event.Namespace,
			Workspace:    event.WorkspaceID,
			SessionID:    event.SessionID,
			TurnID:       event.TurnID,
			Kind:         event.Kind,
			MessageIndex: event.MessageIndex,
			OccurredAt:   event.OccurredAt,
			Offset:       record.Offset,
			Payload:      payload,
		})
	}
	return events, nil
}

func readLedgerSegmentRecords(path string, repairTail bool) ([]ledgerEventRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read ledger segment: %w", err)
	}
	if len(data) == 0 {
		return nil, nil
	}
	lines := bytes.Split(data, []byte{'\n'})
	records := make([]ledgerEventRecord, 0, len(lines))
	lastGoodEnd := 0
	offset := 0
	for index, line := range lines {
		lineEnd := offset + len(line)
		if index < len(lines)-1 {
			lineEnd++
		}
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			lastGoodEnd = lineEnd
			offset = lineEnd
			continue
		}
		var event LedgerEvent
		if err := json.Unmarshal(trimmed, &event); err != nil {
			if repairTail && ledgerOnlyTailRemains(lines[index+1:]) {
				if writeErr := os.WriteFile(path, data[:lastGoodEnd], 0o600); writeErr != nil {
					return nil, fmt.Errorf("repair ledger tail: %w", writeErr)
				}
				return records, nil
			}
			return nil, fmt.Errorf("decode ledger event: %w", err)
		}
		records = append(records, ledgerEventRecord{Event: normalizeLedgerEvent(event), Offset: int64(lineEnd)})
		lastGoodEnd = lineEnd
		offset = lineEnd
	}
	return records, nil
}
