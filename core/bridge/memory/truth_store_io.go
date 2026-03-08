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
	"time"
)

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
		events = append(events, normalizeTruthEvent(event))
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
	for _, evidence := range normalized.RawEvidence {
		if _, err := writer.AppendEvidenceEvent(eventType, normalized, evidence, traceID); err != nil {
			return err
		}
	}
	for _, claim := range normalized.Claims {
		if _, err := writer.AppendClaimEvent(claim, traceID); err != nil {
			return err
		}
	}
	if _, err := writer.UpsertClaims(normalized.Claims); err != nil {
		return err
	}
	projection := normalized
	projection.Claims = nil
	projection.RawEvidence = nil
	if _, err := writer.AppendEvent(truthEventTypeObjectProjected, projection, traceID); err != nil {
		return err
	}
	if _, err := writer.UpsertObject(normalized); err != nil {
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
