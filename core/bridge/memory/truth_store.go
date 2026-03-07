package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
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
	sidecar        truthObjectSidecar
	sidecarWG      sync.WaitGroup
}

func NewTruthWriter(config MemoryConfig, metrics *memoryCounters) *TruthWriter {
	enabled := config.TruthEnabled || config.TruthDualWrite
	baseDir := resolveMemoryPath(config.TruthBaseDir)
	if !enabled || strings.TrimSpace(baseDir) == "" {
		return nil
	}
	writer := &TruthWriter{
		enabled:        true,
		dualWrite:      config.TruthDualWrite,
		failOpen:       config.TruthShadowFailOpen,
		baseDir:        baseDir,
		eventsDir:      filepath.Join(baseDir, truthEventsDirName),
		objectsPath:    filepath.Join(baseDir, truthObjectsSnapshotFileName),
		claimsPath:     filepath.Join(baseDir, truthClaimsSnapshotFileName),
		checkpointPath: filepath.Join(baseDir, truthReplayCheckpointFileName),
		metrics:        metrics,
		objectSnapshot: make(map[string]MemoryObject),
		claimSnapshot:  make(map[string]MemoryClaim),
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
	w.sidecar = sidecar
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
	w.enqueueSidecarSync(w.sidecar, normalized)
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
	w.enqueueSidecarSync(w.sidecar, normalized)
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

func (w *TruthWriter) loadSnapshots() error {
	if !w.Enabled() {
		return nil
	}
	objects, err := readTruthObjectSnapshot(w.objectsPath)
	if err != nil {
		return err
	}
	claims, err := readTruthClaimSnapshot(w.claimsPath)
	if err != nil {
		return err
	}
	for _, object := range objects {
		normalized := normalizeMemoryObject(object)
		w.objectSnapshot[normalized.ObjectID] = normalized
	}
	for _, claim := range claims {
		normalized := normalizeMemoryClaim(claim, claim.ObjectID)
		w.claimSnapshot[normalized.ClaimID] = normalized
	}
	return nil
}

func (w *TruthWriter) ensureLayoutLocked() error {
	if err := os.MkdirAll(w.baseDir, 0o700); err != nil {
		return fmt.Errorf("create truth base dir: %w", err)
	}
	if err := os.MkdirAll(w.eventsDir, 0o700); err != nil {
		return fmt.Errorf("create truth events dir: %w", err)
	}
	return nil
}

func (w *TruthWriter) persistSnapshotsLocked() error {
	if err := writeJSONAtomic(w.objectsPath, truthObjectsSnapshot{
		SchemaVersion: truthSchemaVersion,
		Objects:       truthSortedObjects(w.objectSnapshot),
	}); err != nil {
		return err
	}
	if err := writeJSONAtomic(w.claimsPath, truthClaimsSnapshot{
		SchemaVersion: truthSchemaVersion,
		Claims:        truthSortedClaims(w.claimSnapshot),
	}); err != nil {
		return err
	}
	return nil
}

func (w *TruthWriter) syncClaimsForObjectLocked(object MemoryObject) {
	w.replaceClaimsForObjectIDLocked(object.ObjectID, object.Claims)
	current := w.objectSnapshot[object.ObjectID]
	current.Claims = append([]MemoryClaim(nil), normalizeMemoryObject(object).Claims...)
	w.objectSnapshot[object.ObjectID] = current
}

func (w *TruthWriter) replaceClaimsForObjectIDLocked(objectID string, claims []MemoryClaim) {
	trimmedObjectID := strings.TrimSpace(objectID)
	if trimmedObjectID == "" {
		return
	}
	for claimID, existing := range w.claimSnapshot {
		if existing.ObjectID == trimmedObjectID {
			delete(w.claimSnapshot, claimID)
		}
	}
	normalizedClaims := make([]MemoryClaim, 0, len(claims))
	for _, item := range claims {
		normalized := normalizeMemoryClaim(item, trimmedObjectID)
		if normalized.ClaimID == "" {
			continue
		}
		w.claimSnapshot[normalized.ClaimID] = normalized
		normalizedClaims = append(normalizedClaims, normalized)
	}
	if object, ok := w.objectSnapshot[trimmedObjectID]; ok {
		object.Claims = normalizedClaims
		w.objectSnapshot[trimmedObjectID] = normalizeMemoryObject(object)
	}
}

func (w *TruthWriter) replayLocked() (map[string]MemoryObject, map[string]MemoryClaim, TruthWriteResult, truthReplayCheckpoint, error) {
	files, err := listTruthEventFiles(w.eventsDir)
	if err != nil {
		return nil, nil, TruthWriteResult{}, truthReplayCheckpoint{}, err
	}
	objects := make(map[string]MemoryObject)
	claims := make(map[string]MemoryClaim)
	eventCount := 0
	lastEventID := ""
	lastOccurredAt := time.Time{}
	for _, path := range files {
		events, err := readTruthEvents(path)
		if err != nil {
			return nil, nil, TruthWriteResult{}, truthReplayCheckpoint{}, err
		}
		for _, event := range events {
			normalized := normalizeMemoryObject(event.Object)
			objects[normalized.ObjectID] = normalized
			for claimID, existing := range claims {
				if existing.ObjectID == normalized.ObjectID {
					delete(claims, claimID)
				}
			}
			for _, claim := range normalized.Claims {
				normalizedClaim := normalizeMemoryClaim(claim, normalized.ObjectID)
				claims[normalizedClaim.ClaimID] = normalizedClaim
			}
			eventCount++
			lastEventID = event.EventID
			lastOccurredAt = effectiveDecisionTimestamp(event.OccurredAt, normalized.UpdatedAt, lastOccurredAt)
		}
	}
	result := TruthWriteResult{
		SchemaVersion:  truthSchemaVersion,
		EventID:        lastEventID,
		ObjectCount:    len(objects),
		ClaimCount:     len(claims),
		SourceRefCount: truthSourceRefCount(objects, claims),
		OccurredAt:     lastOccurredAt,
	}
	checkpoint := truthReplayCheckpoint{
		SchemaVersion:  truthSchemaVersion,
		LastReplayAt:   time.Now().UTC(),
		LastEventID:    lastEventID,
		EventCount:     eventCount,
		ObjectCount:    result.ObjectCount,
		ClaimCount:     result.ClaimCount,
		SourceRefCount: result.SourceRefCount,
	}
	return objects, claims, result, checkpoint, nil
}

func (w *TruthWriter) snapshotState() (map[string]MemoryObject, map[string]MemoryClaim) {
	if !w.Enabled() {
		return nil, nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return cloneTruthObjects(w.objectSnapshot), cloneTruthClaims(w.claimSnapshot)
}

func (w *TruthWriter) waitForSidecar(timeout time.Duration) bool {
	if w == nil {
		return true
	}
	done := make(chan struct{})
	go func() {
		w.sidecarWG.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (w *TruthWriter) replayState() (map[string]MemoryObject, map[string]MemoryClaim, TruthWriteResult, error) {
	if !w.Enabled() {
		return nil, nil, TruthWriteResult{}, nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	objects, claims, result, _, err := w.replayLocked()
	return objects, claims, result, err
}

func (w *TruthWriter) recordErrorLocked() {
	if w.metrics != nil {
		w.metrics.truthErrors.Add(1)
	}
}

func readTruthObjectSnapshot(path string) ([]MemoryObject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read truth object snapshot: %w", err)
	}
	var payload truthObjectsSnapshot
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode truth object snapshot: %w", err)
	}
	return payload.Objects, nil
}

func readTruthClaimSnapshot(path string) ([]MemoryClaim, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read truth claim snapshot: %w", err)
	}
	var payload truthClaimsSnapshot
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode truth claim snapshot: %w", err)
	}
	return payload.Claims, nil
}

func listTruthEventFiles(eventsDir string) ([]string, error) {
	entries, err := os.ReadDir(eventsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read truth events dir: %w", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		files = append(files, filepath.Join(eventsDir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func readTruthEvents(path string) ([]truthEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open truth events: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buffer := make([]byte, 0, 64*1024)
	scanner.Buffer(buffer, truthEventScannerBufferMaxBytes)
	events := make([]truthEvent, 0, 32)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event truthEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("decode truth event: %w", err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan truth events: %w", err)
	}
	return events, nil
}

func appendJSONLine(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create truth event parent dir: %w", err)
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal truth event: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open truth event log: %w", err)
	}
	defer file.Close()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("append truth event log: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync truth event log: %w", err)
	}
	return nil
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal truth snapshot: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create truth snapshot dir: %w", err)
	}
	tmpPath := fmt.Sprintf("%s.tmp-%d", path, time.Now().UTC().UnixNano())
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write truth snapshot temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace truth snapshot: %w", err)
	}
	return nil
}

func truthSortedObjects(objects map[string]MemoryObject) []MemoryObject {
	if len(objects) == 0 {
		return nil
	}
	out := make([]MemoryObject, 0, len(objects))
	for _, object := range objects {
		out = append(out, normalizeMemoryObject(object))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ObjectID < out[j].ObjectID
	})
	return out
}

func truthSortedClaims(claims map[string]MemoryClaim) []MemoryClaim {
	if len(claims) == 0 {
		return nil
	}
	out := make([]MemoryClaim, 0, len(claims))
	for _, claim := range claims {
		out = append(out, normalizeMemoryClaim(claim, claim.ObjectID))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ClaimID < out[j].ClaimID
	})
	return out
}

func truthSingleObjectSourceRefCount(object MemoryObject) int {
	count := len(object.SourceRefs)
	for _, evidence := range object.RawEvidence {
		count += len(evidence.SourceRefs)
	}
	for _, ref := range object.EmbeddingRefs {
		count += len(ref.SourceRefs)
	}
	return count
}

func truthSourceRefCount(objects map[string]MemoryObject, claims map[string]MemoryClaim) int {
	count := 0
	for _, object := range objects {
		count += truthSingleObjectSourceRefCount(object)
	}
	for _, claim := range claims {
		count += len(claim.SourceRefs)
	}
	return count
}

func cloneTruthObjects(objects map[string]MemoryObject) map[string]MemoryObject {
	if len(objects) == 0 {
		return nil
	}
	out := make(map[string]MemoryObject, len(objects))
	for key, object := range objects {
		out[key] = normalizeMemoryObject(object)
	}
	return out
}

func (w *TruthWriter) enqueueSidecarSync(sidecar truthObjectSidecar, object MemoryObject) {
	if w == nil || sidecar == nil {
		return
	}
	w.sidecarWG.Add(1)
	go func() {
		defer w.sidecarWG.Done()
		if err := sidecar.SyncObject(object); err != nil {
			log.Printf("[MEMORY] truth sidecar sync failed, continuing: object_id=%s err=%v", strings.TrimSpace(object.ObjectID), err)
		}
	}()
}

func cloneTruthClaims(claims map[string]MemoryClaim) map[string]MemoryClaim {
	if len(claims) == 0 {
		return nil
	}
	out := make(map[string]MemoryClaim, len(claims))
	for key, claim := range claims {
		out[key] = normalizeMemoryClaim(claim, claim.ObjectID)
	}
	return out
}

func writeTruthObjectShadow(writer *TruthWriter, eventType string, object MemoryObject, traceID string) error {
	if writer == nil || !writer.DualWriteEnabled() {
		return nil
	}
	normalized := normalizeMemoryObject(object)
	if _, err := writer.AppendEvent(eventType, normalized, traceID); err != nil {
		return err
	}
	if _, err := writer.UpsertObject(normalized); err != nil {
		return err
	}
	if _, err := writer.UpsertClaims(normalized.Claims); err != nil {
		return err
	}
	return nil
}

func handleTruthShadowWriteError(writer *TruthWriter, traceID string, objectID string, err error) error {
	if err == nil {
		return nil
	}
	trace := strings.TrimSpace(traceID)
	if trace == "" {
		trace = "-"
	}
	if writer == nil || writer.FailOpen() {
		log.Printf("[MEMORY] truth shadow write failed, continuing: trace_id=%s object_id=%s err=%v", trace, strings.TrimSpace(objectID), err)
		return nil
	}
	return err
}
