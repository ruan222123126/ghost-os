package memory

import (
	"fmt"
	"os"
	"strings"
	"time"
)

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
	if len(w.claimSnapshot) == 0 {
		w.claimSnapshot = truthClaimsFromObjects(w.objectSnapshot)
	}
	if w.claimSnapshot == nil {
		w.claimSnapshot = make(map[string]MemoryClaim)
	}
	if w.claimStatusProjectionEnabled {
		truthProjectClaimsOntoObjects(w.objectSnapshot, w.claimSnapshot, w.legacyObjectProjectionEnabled)
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
			truthApplyEvent(objects, claims, event)
			normalizedEvent := normalizeTruthEvent(event)
			eventCount++
			lastEventID = normalizedEvent.EventID
			lastOccurredAt = effectiveDecisionTimestamp(normalizedEvent.OccurredAt, lastOccurredAt)
		}
	}
	if len(claims) == 0 {
		claims = truthClaimsFromObjects(objects)
	}
	truthProjectClaimsOntoObjects(objects, claims, w.legacyObjectProjectionEnabled)
	result := truthWriteResultFromSnapshots(objects, claims)
	result.EventID = lastEventID
	result.OccurredAt = lastOccurredAt
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
