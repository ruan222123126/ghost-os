package memory

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

func TestTruthNormalizeMemoryObjectClaimDedupe(t *testing.T) {
	createdAt := time.Date(2026, 3, 7, 9, 0, 0, 0, time.FixedZone("UTC+8", 8*3600))
	object := normalizeMemoryObject(MemoryObject{
		ObjectID:   "obj-test",
		ObjectType: truthObjectTypeProcedureMemo,
		RawEvidence: []MemoryEvidence{{
			Kind:       "decision.memo",
			Text:       "  remember this  ",
			Timestamp:  createdAt,
			Confidence: 1.5,
			SourceRefs: []SourceRef{{SourceKind: truthSourceKindDecisionMemo, SourceID: "memo-1"}, {SourceKind: truthSourceKindDecisionMemo, SourceID: "memo-1"}},
		}},
		Claims: []MemoryClaim{
			{Type: truthClaimTypeIntent, IntentKey: "intent.fix_config", Value: "Fix config", CreatedAt: createdAt},
			{Type: truthClaimTypeIntent, IntentKey: "intent.fix_config", Value: "Fix config", CreatedAt: createdAt},
		},
	})
	if object.SchemaVersion != truthSchemaVersion {
		t.Fatalf("unexpected schema version: %d", object.SchemaVersion)
	}
	if len(object.Claims) != 1 {
		t.Fatalf("expected duplicate claims to be deduped, got %d", len(object.Claims))
	}
	if len(object.RawEvidence) != 1 || len(object.RawEvidence[0].SourceRefs) != 1 {
		t.Fatalf("expected duplicate source refs to be deduped, got %+v", object.RawEvidence)
	}
	if !object.CreatedAt.Equal(createdAt.UTC()) {
		t.Fatalf("expected created_at to normalize to UTC, got %s", object.CreatedAt)
	}
	if object.RawEvidence[0].Confidence != 1 {
		t.Fatalf("expected confidence to clamp to 1, got %.2f", object.RawEvidence[0].Confidence)
	}
}

func TestTruthStableIDsAreDeterministic(t *testing.T) {
	claim := normalizeMemoryClaim(MemoryClaim{
		Type:      truthClaimTypeAnchor,
		AnchorKey: "pref.language.go",
		Value:     "Go",
	}, "obj-1")
	if claim.ClaimID == "" {
		t.Fatal("expected claim id to be generated")
	}
	again := normalizeMemoryClaim(MemoryClaim{
		Type:      truthClaimTypeAnchor,
		AnchorKey: "pref.language.go",
		Value:     "Go",
	}, "obj-1")
	if claim.ClaimID != again.ClaimID {
		t.Fatalf("expected stable claim ids, got %q and %q", claim.ClaimID, again.ClaimID)
	}
	object := normalizeMemoryObject(MemoryObject{
		ObjectID:   "obj-1",
		ObjectType: truthObjectTypeSemanticNote,
		Summary:    "Stable event payload",
		CreatedAt:  time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC),
	})
	if buildTruthEventID(truthEventTypeMarkdownNode, object) != buildTruthEventID(truthEventTypeMarkdownNode, object) {
		t.Fatal("expected stable event ids")
	}
}

func TestTruthWriterSnapshotOverwrite(t *testing.T) {
	writer := newTruthTestWriter(t, filepath.Join(t.TempDir(), "truth"))
	object := normalizeMemoryObject(MemoryObject{
		ObjectID:   "obj-overwrite",
		ObjectType: truthObjectTypeSemanticNote,
		Summary:    "first",
		CreatedAt:  time.Date(2026, 3, 7, 11, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 3, 7, 11, 0, 0, 0, time.UTC),
		Claims:     []MemoryClaim{{Type: truthClaimTypeAnchor, AnchorKey: "a", Value: "A"}},
	})
	if _, err := writer.UpsertObject(object); err != nil {
		t.Fatalf("upsert object: %v", err)
	}
	object.Summary = "second"
	object.UpdatedAt = object.UpdatedAt.Add(time.Minute)
	object.Claims = []MemoryClaim{{Type: truthClaimTypeAnchor, AnchorKey: "b", Value: "B"}}
	if _, err := writer.UpsertObject(object); err != nil {
		t.Fatalf("overwrite object: %v", err)
	}
	objects, err := readTruthObjectSnapshot(writer.objectsPath)
	if err != nil {
		t.Fatalf("read object snapshot: %v", err)
	}
	if len(objects) != 1 || objects[0].Summary != "second" {
		t.Fatalf("expected snapshot overwrite, got %+v", objects)
	}
	claims, err := readTruthClaimSnapshot(writer.claimsPath)
	if err != nil {
		t.Fatalf("read claim snapshot: %v", err)
	}
	if len(claims) != 2 {
		t.Fatalf("expected claim center to keep both assertions, got %+v", claims)
	}
}

func TestTruthWriterReplayDeterministic(t *testing.T) {
	writer := newTruthTestWriter(t, filepath.Join(t.TempDir(), "truth"))
	mapper := NewTruthMapper()
	archiveObjects := mapper.MapArchiveMessages("session-replay", time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC), []llm.Message{{Role: llm.RoleUser, Text: "hello replay"}})
	for _, object := range archiveObjects {
		if err := writeTruthObjectShadow(writer, truthEventTypeArchiveMessage, object, ""); err != nil {
			t.Fatalf("write archive object: %v", err)
		}
	}
	node := MarkdownNode{
		ID:         "node-replay",
		SessionID:  "session-replay",
		CreatedAt:  time.Date(2026, 3, 7, 12, 5, 0, 0, time.UTC),
		Summary:    "Remember stable anchors",
		Content:    "Remember stable anchors for replay tests.",
		Anchors:    []MemoryAnchor{{Type: MemoryAnchorPreference, Key: "pref.go", Value: "Go", Weight: 0.8}},
		SourceIDs:  []string{"session-replay:000000"},
		Confidence: 0.8,
	}
	if err := writeTruthObjectShadow(writer, truthEventTypeMarkdownNode, mapper.MapMarkdownNode(node), ""); err != nil {
		t.Fatalf("write markdown object: %v", err)
	}
	input := decisionCaptureTestInput(DecisionOutcomeSuccess)
	memo := normalizeDecisionMemo(DecisionMemo{
		ID:               buildDecisionMemoID(input),
		Namespace:        input.Namespace,
		SessionID:        input.SessionID,
		TraceID:          input.TraceID,
		TurnID:           input.TurnID,
		IntentKey:        "intent.inspect_config",
		IntentSummary:    "Inspect config.toml",
		StrategySummary:  "Read config, validate assumptions, then respond.",
		Constraints:      []string{"Preserve existing behavior"},
		ValidationChecks: []string{"Confirm config keys"},
		AvoidPatterns:    []string{"Do not rewrite unrelated files"},
		ToolsUsed:        []DecisionToolUse{{Name: "read_file"}},
		Outcome:          DecisionOutcomeSuccess,
		OutcomeSummary:   "Inspected config successfully.",
		CreatedAt:        input.TurnStartedAt,
		LastUsedAt:       input.TurnFinishedAt,
		Confidence:       0.9,
	})
	if err := writeTruthObjectShadow(writer, truthEventTypeDecisionMemo, mapper.MapDecisionMemo(memo, input), input.TraceID); err != nil {
		t.Fatalf("write decision object: %v", err)
	}
	verify, err := NewTruthVerifier(writer).Verify()
	if err != nil {
		t.Fatalf("verify replay: %v", err)
	}
	if !verify.Match {
		t.Fatalf("expected live and replay snapshots to match: %+v", verify)
	}
	replayed, err := writer.Replay()
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if replayed.ObjectCount != verify.Live.ObjectCount || replayed.ClaimCount != verify.Live.ClaimCount || replayed.SourceRefCount != verify.Live.SourceRefCount {
		t.Fatalf("unexpected replay counts: live=%+v replay=%+v", verify.Live, replayed)
	}
	if _, err := os.Stat(writer.checkpointPath); err != nil {
		t.Fatalf("expected replay checkpoint to exist: %v", err)
	}
	files, err := listTruthEventFiles(writer.eventsDir)
	if err != nil {
		t.Fatalf("list event files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected truth event log files to be written")
	}
}

func TestTruthShadowDualWriteArchiveMarkdownDecision(t *testing.T) {
	manager := newTruthTestManager(t, true, "")
	messages := []llm.Message{{Role: llm.RoleUser, Text: "archive me"}, {Role: llm.RoleAssistant, Text: "done"}}
	if err := manager.cold.Archive("session-shadow", messages); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if err := manager.SaveMarkdownNode(MarkdownNode{
		ID:         "node-shadow",
		SessionID:  "session-shadow",
		CreatedAt:  time.Date(2026, 3, 7, 13, 0, 0, 0, time.UTC),
		Summary:    "Semantic note",
		Content:    "Semantic note content",
		SourceIDs:  []string{"session-shadow:000000"},
		Anchors:    []MemoryAnchor{{Type: MemoryAnchorConstraint, Key: "constraint.readonly", Value: "readonly", Weight: 0.7}},
		Confidence: 0.7,
	}); err != nil {
		t.Fatalf("save markdown node: %v", err)
	}
	if err := manager.CaptureDecisionTurn(decisionCaptureTestInput(DecisionOutcomeSuccess)); err != nil {
		t.Fatalf("capture decision turn: %v", err)
	}
	verify, err := manager.VerifyTruthReplay()
	if err != nil {
		t.Fatalf("verify truth replay: %v", err)
	}
	if !verify.Match {
		t.Fatalf("expected truth replay to match live snapshot: %+v", verify)
	}
	objects, err := readTruthObjectSnapshot(manager.truth.objectsPath)
	if err != nil {
		t.Fatalf("read truth objects: %v", err)
	}
	hasTypes := map[string]bool{}
	for _, object := range objects {
		hasTypes[object.ObjectType] = true
	}
	for _, want := range []string{truthObjectTypeEvidenceMessage, truthObjectTypeSemanticNote, truthObjectTypeProcedureMemo} {
		if !hasTypes[want] {
			t.Fatalf("expected truth object type %q in snapshot, got %+v", want, hasTypes)
		}
	}
}

func TestSaveMarkdownNodeProjectsClaimLineage(t *testing.T) {
	manager := newTruthTestManager(t, true, "")
	if err := manager.SaveMarkdownNode(MarkdownNode{
		ID:         "node-lineage",
		SessionID:  "session-lineage",
		CreatedAt:  time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC),
		Summary:    "Remember preferred implementation language",
		Content:    "Remember preferred implementation language is Go.",
		SourceIDs:  []string{"session-lineage:000000"},
		Anchors:    []MemoryAnchor{{Type: MemoryAnchorPreference, Key: "language", Value: "Go", Weight: 0.91, Reason: "user explicitly prefers Go"}},
		Confidence: 0.91,
	}); err != nil {
		t.Fatalf("save markdown node: %v", err)
	}

	loaded, err := manager.LoadMarkdownNode("node-lineage")
	if err != nil {
		t.Fatalf("load markdown node: %v", err)
	}
	if loaded.ProjectionVersion != markdownProjectionVersion {
		t.Fatalf("expected projection version %q, got %q", markdownProjectionVersion, loaded.ProjectionVersion)
	}
	if len(loaded.SourceEvidenceIDs) == 0 {
		t.Fatalf("expected source evidence ids, got %+v", loaded)
	}
	if len(loaded.SourceClaimIDs) == 0 {
		t.Fatalf("expected source claim ids, got %+v", loaded)
	}
	if len(loaded.DerivedClaimIDs) == 0 {
		t.Fatalf("expected derived claim ids, got %+v", loaded)
	}
	if len(loaded.Anchors) == 0 || loaded.Anchors[0].Type != MemoryAnchorPreference || loaded.Anchors[0].Value != "Go" {
		t.Fatalf("expected projected anchors from claims, got %+v", loaded.Anchors)
	}
	stats, err := manager.BackfillMarkdownLineage()
	if err != nil {
		t.Fatalf("backfill markdown lineage: %v", err)
	}
	if stats.NodesScanned == 0 {
		t.Fatalf("expected backfill to scan nodes, got %+v", stats)
	}
}

func TestTruthArchiveDualWriteDoesNotChangeRetrieve(t *testing.T) {
	baseline := newTruthTestManager(t, false, "")
	shadow := newTruthTestManager(t, true, "")
	messages := []llm.Message{{Role: llm.RoleUser, Text: "same archive result"}, {Role: llm.RoleAssistant, Text: "same answer"}}
	if err := baseline.cold.Archive("session-compare", messages); err != nil {
		t.Fatalf("baseline archive: %v", err)
	}
	if err := shadow.cold.Archive("session-compare", messages); err != nil {
		t.Fatalf("shadow archive: %v", err)
	}
	baseEntries, err := baseline.Query(MemoryQuery{Metadata: map[string]any{"session_id": "session-compare"}})
	if err != nil {
		t.Fatalf("baseline query: %v", err)
	}
	shadowEntries, err := shadow.Query(MemoryQuery{Metadata: map[string]any{"session_id": "session-compare"}})
	if err != nil {
		t.Fatalf("shadow query: %v", err)
	}
	if !reflect.DeepEqual(stripEntryTimes(baseEntries), stripEntryTimes(shadowEntries)) {
		t.Fatalf("expected archive retrieve to stay unchanged, base=%+v shadow=%+v", baseEntries, shadowEntries)
	}
}

func TestTruthQueryRegressionBuildContextAndQueryResultsUnchanged(t *testing.T) {
	baseline := newTruthTestManager(t, false, "")
	shadow := newTruthTestManager(t, true, "")
	storedAt := time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC)
	storeWarm := func(manager *MemoryManager) {
		if err := manager.warm.Store(MemoryEntry{
			ID:         "warm-regression-1",
			Content:    "Ghost-OS keeps query paths frozen during week one truth dual-write.",
			Timestamp:  storedAt,
			Importance: 0.8,
			Metadata:   map[string]any{"session_id": "session-regression", "role": "assistant"},
		}); err != nil {
			t.Fatalf("store warm entry: %v", err)
		}
		if err := manager.SaveMarkdownNode(MarkdownNode{
			ID:         "node-regression",
			SessionID:  "session-regression",
			CreatedAt:  storedAt,
			Summary:    "Week one keeps read path frozen",
			Content:    "Week one keeps read path frozen while shadow truth writes happen in parallel.",
			SourceIDs:  []string{"session-regression:000000"},
			Confidence: 0.8,
		}); err != nil {
			t.Fatalf("save markdown: %v", err)
		}
		input := decisionCaptureTestInput(DecisionOutcomeSuccess)
		input.SessionID = "session-regression"
		input.TraceID = "trace-regression"
		input.TurnID = "turn-regression"
		input.TurnStartedAt = storedAt
		input.TurnFinishedAt = storedAt.Add(2 * time.Minute)
		if err := manager.CaptureDecisionTurn(input); err != nil {
			t.Fatalf("capture decision: %v", err)
		}
	}
	storeWarm(baseline)
	storeWarm(shadow)
	baseWindow, err := baseline.BuildContextWindow("session-regression", "Explain Ghost-OS truth dual-write")
	if err != nil {
		t.Fatalf("baseline build context: %v", err)
	}
	shadowWindow, err := shadow.BuildContextWindow("session-regression", "Explain Ghost-OS truth dual-write")
	if err != nil {
		t.Fatalf("shadow build context: %v", err)
	}
	if !reflect.DeepEqual(baseWindow, shadowWindow) {
		t.Fatalf("expected BuildContextWindow output to stay unchanged, base=%+v shadow=%+v", baseWindow, shadowWindow)
	}
	query := MemoryQuery{
		Keywords:        []string{"truth", "dual-write"},
		IncludeMarkdown: true,
		IncludeDecision: true,
		SemanticQuery:   "truth dual-write",
	}
	baseResult, err := baseline.QueryResultWithScope(query, SessionScope{SessionID: "session-regression"})
	if err != nil {
		t.Fatalf("baseline query result: %v", err)
	}
	shadowResult, err := shadow.QueryResultWithScope(query, SessionScope{SessionID: "session-regression"})
	if err != nil {
		t.Fatalf("shadow query result: %v", err)
	}
	if !reflect.DeepEqual(stripEntryTimes(baseResult.Entries), stripEntryTimes(shadowResult.Entries)) {
		t.Fatalf("expected QueryResultWithScope entries to stay unchanged, base=%+v shadow=%+v", baseResult.Entries, shadowResult.Entries)
	}
	if !reflect.DeepEqual(baseResult.DecisionHits, shadowResult.DecisionHits) {
		t.Fatalf("expected decision hits to stay unchanged, base=%+v shadow=%+v", baseResult.DecisionHits, shadowResult.DecisionHits)
	}
	if shadow.truth != nil && !shadow.truth.waitForSidecar(2*time.Second) {
		t.Fatal("timed out waiting for truth sidecar sync")
	}
	truthDebugQuery := MemoryQuery{
		Keywords:        []string{"week", "truth", "writes"},
		IncludeMarkdown: true,
		IncludeDecision: true,
		SemanticQuery:   "week one keeps read path frozen while shadow truth writes happen in parallel",
	}
	shadowSpecificResult, err := shadow.QueryResultWithScope(truthDebugQuery, SessionScope{SessionID: "session-regression"})
	if err != nil {
		t.Fatalf("shadow scoped query result: %v", err)
	}
	truthDebugQuery.IncludeTruth = true
	truthDebugQuery.TruthDebug = true
	shadowDebugResult, err := shadow.QueryResultWithScope(truthDebugQuery, SessionScope{SessionID: "session-regression"})
	if err != nil {
		t.Fatalf("shadow truth debug query result: %v", err)
	}
	if len(shadowDebugResult.TruthHits) == 0 {
		t.Fatalf("expected truth debug hits from shadow read path, got %+v", shadowDebugResult)
	}
	if !reflect.DeepEqual(stripEntryTimes(shadowSpecificResult.Entries), stripEntryTimes(shadowDebugResult.Entries)) {
		t.Fatalf("expected truth debug side channel to keep live entries unchanged, base=%+v shadow_debug=%+v", shadowSpecificResult.Entries, shadowDebugResult.Entries)
	}
}

func TestTruthShadowWriteFailOpenLogsTraceID(t *testing.T) {
	baseDir := t.TempDir()
	blockedTruthPath := filepath.Join(baseDir, "truth-blocked")
	if err := os.WriteFile(blockedTruthPath, []byte("not-a-dir"), 0o600); err != nil {
		t.Fatalf("create blocked truth path: %v", err)
	}
	manager := newTruthTestManager(t, true, blockedTruthPath)
	input := decisionCaptureTestInput(DecisionOutcomeSuccess)
	input.TraceID = "trace-fail-open"
	var buf bytes.Buffer
	prevWriter := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(prevWriter)
		log.SetFlags(prevFlags)
	}()
	if err := manager.CaptureDecisionTurn(input); err != nil {
		t.Fatalf("expected fail-open shadow write, got error: %v", err)
	}
	if manager.Metrics().TruthErrors == 0 {
		t.Fatal("expected truth error metrics to increment")
	}
	if !strings.Contains(buf.String(), "trace_id=trace-fail-open") {
		t.Fatalf("expected log to include trace_id, got %q", buf.String())
	}
}

func newTruthTestManager(t *testing.T, truthEnabled bool, truthBaseDir string) *MemoryManager {
	t.Helper()
	baseDir := t.TempDir()
	config := MemoryConfig{
		WarmCapacity:      32,
		WarmPath:          filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:       filepath.Join(baseDir, "cold"),
		AutoRecallEnabled: true,
		AutoRecallLimit:   4,
		DecisionEnabled:   true,
		DecisionPath:      filepath.Join(baseDir, "decision"),
	}
	if truthEnabled {
		config.TruthEnabled = true
		config.TruthDualWrite = true
		config.TruthBaseDir = truthBaseDir
	}
	return NewMemoryManager(config)
}

func decisionCaptureTestInput(outcome string) DecisionCaptureInput {
	startedAt := time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(2 * time.Minute)
	return DecisionCaptureInput{
		Namespace:   "workspace:test",
		SessionID:   "session-capture-1",
		TraceID:     "trace-capture-1",
		TurnID:      "turn-1",
		UserMessage: "inspect config.toml",
		NewMessages: []llm.Message{
			{Role: llm.RoleUser, Text: "inspect config.toml"},
			{Role: llm.RoleAssistant, Text: "Configuration looks healthy."},
		},
		Outcome:        outcome,
		TurnStartedAt:  startedAt,
		TurnFinishedAt: finishedAt,
		Environment: DecisionEnvFingerprint{
			Domain: "coding",
		},
	}
}

func newTruthTestWriter(t *testing.T, baseDir string) *TruthWriter {
	t.Helper()
	writer := NewTruthWriter(MemoryConfig{
		TruthEnabled:        true,
		TruthDualWrite:      true,
		TruthBaseDir:        baseDir,
		TruthShadowFailOpen: true,
	}, &memoryCounters{})
	if writer == nil {
		t.Fatal("expected truth writer to be initialized")
	}
	return writer
}

func stripEntryTimes(entries []MemoryEntry) []MemoryEntry {
	if len(entries) == 0 {
		return nil
	}
	out := make([]MemoryEntry, len(entries))
	for i, entry := range entries {
		entry.Timestamp = time.Time{}
		entry.LastAccessedAt = time.Time{}
		entry.ExpiresAt = time.Time{}
		out[i] = entry
	}
	return out
}
