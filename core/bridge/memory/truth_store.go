package memory

import (
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	truthEventsDirName              = "events"
	truthObjectsSnapshotFileName    = "objects.snapshot.json"
	truthClaimsSnapshotFileName     = "claims.snapshot.json"
	truthReplayCheckpointFileName   = "replay.checkpoint.json"
	truthEventScannerBufferMaxBytes = 8 << 20
)

type truthObjectsSnapshot struct {
	SchemaVersion int            `json:"schema_version"`
	Objects       []MemoryObject `json:"objects"`
}

type truthClaimsSnapshot struct {
	SchemaVersion int           `json:"schema_version"`
	Claims        []MemoryClaim `json:"claims"`
}

type truthReplayCheckpoint struct {
	SchemaVersion  int       `json:"schema_version"`
	LastReplayAt   time.Time `json:"last_replay_at,omitempty"`
	LastEventID    string    `json:"last_event_id,omitempty"`
	EventCount     int       `json:"event_count,omitempty"`
	ObjectCount    int       `json:"object_count,omitempty"`
	ClaimCount     int       `json:"claim_count,omitempty"`
	SourceRefCount int       `json:"source_ref_count,omitempty"`
}

type truthObjectSidecar interface {
	SyncObject(MemoryObject) error
}

type truthObjectSidecarMux struct {
	sidecars []truthObjectSidecar
}

func (m *truthObjectSidecarMux) Add(sidecar truthObjectSidecar) {
	if m == nil || sidecar == nil {
		return
	}
	for _, existing := range m.sidecars {
		if existing == sidecar {
			return
		}
	}
	m.sidecars = append(m.sidecars, sidecar)
}

func (m *truthObjectSidecarMux) Replace(sidecar truthObjectSidecar) {
	if m == nil {
		return
	}
	m.sidecars = m.sidecars[:0]
	m.Add(sidecar)
}

func (m *truthObjectSidecarMux) Snapshot() []truthObjectSidecar {
	if m == nil || len(m.sidecars) == 0 {
		return nil
	}
	return append([]truthObjectSidecar(nil), m.sidecars...)
}

func (m *truthObjectSidecarMux) SyncObject(object MemoryObject) error {
	if m == nil {
		return nil
	}
	for _, sidecar := range m.sidecars {
		if sidecar == nil {
			continue
		}
		if err := sidecar.SyncObject(object); err != nil {
			return err
		}
	}
	return nil
}

// TruthWriter 维护 schema v1 的 shadow event log 与快照。
type TruthWriter struct {
	enabled        bool
	dualWrite      bool
	failOpen       bool
	baseDir        string
	eventsDir      string
	objectsPath    string
	claimsPath     string
	checkpointPath string
	metrics        *memoryCounters

	mu             sync.Mutex
	objectSnapshot map[string]MemoryObject
	claimSnapshot  map[string]MemoryClaim
	sidecars       *truthObjectSidecarMux
	sidecarWG      sync.WaitGroup
}

func NewTruthWriter(config MemoryConfig, metrics *memoryCounters) *TruthWriter {
	config = applyLegacyMemoryConfig(config)
	enabled := config.Truth.Enabled || config.Truth.DualWrite
	baseDir := resolveMemoryPath(config.Truth.BaseDir)
	if !enabled || strings.TrimSpace(baseDir) == "" {
		return nil
	}
	writer := &TruthWriter{
		enabled:        true,
		dualWrite:      config.Truth.DualWrite,
		failOpen:       config.Truth.ShadowFailOpen,
		baseDir:        baseDir,
		eventsDir:      filepath.Join(baseDir, truthEventsDirName),
		objectsPath:    filepath.Join(baseDir, truthObjectsSnapshotFileName),
		claimsPath:     filepath.Join(baseDir, truthClaimsSnapshotFileName),
		checkpointPath: filepath.Join(baseDir, truthReplayCheckpointFileName),
		metrics:        metrics,
		objectSnapshot: make(map[string]MemoryObject),
		claimSnapshot:  make(map[string]MemoryClaim),
		sidecars:       &truthObjectSidecarMux{},
	}
	if err := writer.loadSnapshots(); err != nil {
		log.Printf("[MEMORY] truth shadow snapshot load failed, starting from empty snapshots: base=%s err=%v", writer.baseDir, err)
		writer.objectSnapshot = make(map[string]MemoryObject)
		writer.claimSnapshot = make(map[string]MemoryClaim)
	}
	return writer
}

func (w *TruthWriter) Enabled() bool {
	return w != nil && w.enabled && strings.TrimSpace(w.baseDir) != ""
}

func (w *TruthWriter) DualWriteEnabled() bool {
	return w.Enabled() && w.dualWrite
}

func (w *TruthWriter) FailOpen() bool {
	if w == nil {
		return true
	}
	return w.failOpen
}

func (w *TruthWriter) BaseDir() string {
	if w == nil {
		return ""
	}
	return w.baseDir
}

func (w *TruthWriter) SetObjectSidecar(sidecar truthObjectSidecar) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.sidecars == nil {
		w.sidecars = &truthObjectSidecarMux{}
	}
	w.sidecars.Replace(sidecar)
}

func (w *TruthWriter) AddObjectSidecar(sidecar truthObjectSidecar) {
	if w == nil || sidecar == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.sidecars == nil {
		w.sidecars = &truthObjectSidecarMux{}
	}
	w.sidecars.Add(sidecar)
}

func (w *TruthWriter) AppendEvent(eventType string, object MemoryObject, traceID string) (TruthWriteResult, error) {
	if !w.Enabled() {
		return TruthWriteResult{}, nil
	}
	normalized := normalizeMemoryObject(object)
	event := truthEvent{
		SchemaVersion: truthSchemaVersion,
		EventType:     strings.TrimSpace(eventType),
		ObjectID:      normalized.ObjectID,
		ObjectType:    normalized.ObjectType,
		OccurredAt:    effectiveDecisionTimestamp(normalized.UpdatedAt, normalized.CreatedAt),
		WrittenAt:     time.Now().UTC(),
		TraceID:       strings.TrimSpace(traceID),
		Object:        normalized,
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	event.EventID = buildTruthEventID(event.EventType, normalized)

	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureLayoutLocked(); err != nil {
		w.recordErrorLocked()
		return TruthWriteResult{}, err
	}
	path := filepath.Join(w.eventsDir, event.OccurredAt.Format("2006-01-02")+".jsonl")
	if err := appendJSONLine(path, event); err != nil {
		w.recordErrorLocked()
		return TruthWriteResult{}, err
	}
	if w.metrics != nil {
		w.metrics.truthEventsWritten.Add(1)
	}
	w.enqueueSidecarSync(w.sidecars, normalized)
	return TruthWriteResult{
		SchemaVersion:  truthSchemaVersion,
		EventID:        event.EventID,
		ObjectID:       normalized.ObjectID,
		ObjectType:     normalized.ObjectType,
		ObjectCount:    1,
		ClaimCount:     len(normalized.Claims),
		SourceRefCount: truthSingleObjectSourceRefCount(normalized),
		OccurredAt:     event.OccurredAt,
	}, nil
}

func (w *TruthWriter) UpsertObject(object MemoryObject) (TruthWriteResult, error) {
	if !w.Enabled() {
		return TruthWriteResult{}, nil
	}
	normalized := normalizeMemoryObject(object)
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureLayoutLocked(); err != nil {
		w.recordErrorLocked()
		return TruthWriteResult{}, err
	}
	w.objectSnapshot[normalized.ObjectID] = normalized
	w.syncClaimsForObjectLocked(normalized)
	if err := w.persistSnapshotsLocked(); err != nil {
		w.recordErrorLocked()
		return TruthWriteResult{}, err
	}
	if w.metrics != nil {
		w.metrics.truthObjectsUpserted.Add(1)
	}
	w.enqueueSidecarSync(w.sidecars, normalized)
	return TruthWriteResult{
		SchemaVersion:  truthSchemaVersion,
		ObjectID:       normalized.ObjectID,
		ObjectType:     normalized.ObjectType,
		ObjectCount:    1,
		ClaimCount:     len(normalized.Claims),
		SourceRefCount: truthSingleObjectSourceRefCount(normalized),
		OccurredAt:     effectiveDecisionTimestamp(normalized.UpdatedAt, normalized.CreatedAt),
	}, nil
}

func (w *TruthWriter) UpdateEmbeddingRef(objectID string, ref EmbeddingRef) error {
	if !w.Enabled() {
		return nil
	}
	trimmedObjectID := strings.TrimSpace(objectID)
	normalizedRef := normalizeEmbeddingRef(ref, trimmedObjectID)
	if trimmedObjectID == "" || normalizedRef.EmbeddingID == "" {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureLayoutLocked(); err != nil {
		w.recordErrorLocked()
		return err
	}
	object, ok := w.objectSnapshot[trimmedObjectID]
	if !ok {
		return nil
	}
	merged := make([]EmbeddingRef, 0, len(object.EmbeddingRefs)+1)
	replaced := false
	for _, current := range object.EmbeddingRefs {
		normalizedCurrent := normalizeEmbeddingRef(current, trimmedObjectID)
		if normalizedCurrent.RefID == "" {
			continue
		}
		if normalizedCurrent.RefID == normalizedRef.RefID || (normalizedCurrent.Source == normalizedRef.Source && normalizedCurrent.EmbeddingID == normalizedRef.EmbeddingID) {
			merged = append(merged, normalizedRef)
			replaced = true
			continue
		}
		merged = append(merged, normalizedCurrent)
	}
	if !replaced {
		merged = append(merged, normalizedRef)
	}
	object.EmbeddingRefs = merged
	object.UpdatedAt = effectiveDecisionTimestamp(time.Now().UTC(), object.UpdatedAt, object.CreatedAt)
	object = normalizeMemoryObject(object)
	w.objectSnapshot[trimmedObjectID] = object
	if err := w.persistSnapshotsLocked(); err != nil {
		w.recordErrorLocked()
		return err
	}
	return nil
}

func (w *TruthWriter) UpsertClaims(claims []MemoryClaim) (TruthWriteResult, error) {
	if !w.Enabled() {
		return TruthWriteResult{}, nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureLayoutLocked(); err != nil {
		w.recordErrorLocked()
		return TruthWriteResult{}, err
	}
	grouped := make(map[string][]MemoryClaim)
	totalRefs := 0
	for _, item := range claims {
		normalized := normalizeMemoryClaim(item, item.ObjectID)
		if normalized.ObjectID == "" || normalized.ClaimID == "" {
			continue
		}
		grouped[normalized.ObjectID] = append(grouped[normalized.ObjectID], normalized)
		totalRefs += len(normalized.SourceRefs)
	}
	for objectID, items := range grouped {
		w.replaceClaimsForObjectIDLocked(objectID, items)
	}
	if err := w.persistSnapshotsLocked(); err != nil {
		w.recordErrorLocked()
		return TruthWriteResult{}, err
	}
	if w.metrics != nil {
		w.metrics.truthClaimsUpserted.Add(uint64(len(claims)))
	}
	return TruthWriteResult{
		SchemaVersion:  truthSchemaVersion,
		ObjectCount:    len(grouped),
		ClaimCount:     len(claims),
		SourceRefCount: totalRefs,
	}, nil
}

func (w *TruthWriter) Replay() (TruthWriteResult, error) {
	if !w.Enabled() {
		return TruthWriteResult{}, nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureLayoutLocked(); err != nil {
		w.recordErrorLocked()
		return TruthWriteResult{}, err
	}
	objects, claims, result, checkpoint, err := w.replayLocked()
	if err != nil {
		w.recordErrorLocked()
		return TruthWriteResult{}, err
	}
	w.objectSnapshot = objects
	w.claimSnapshot = claims
	if err := w.persistSnapshotsLocked(); err != nil {
		w.recordErrorLocked()
		return TruthWriteResult{}, err
	}
	if err := writeJSONAtomic(w.checkpointPath, checkpoint); err != nil {
		w.recordErrorLocked()
		return TruthWriteResult{}, err
	}
	if w.metrics != nil {
		w.metrics.truthReplays.Add(1)
	}
	return result, nil
}
