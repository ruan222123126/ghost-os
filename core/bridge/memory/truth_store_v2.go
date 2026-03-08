package memory

import (
	"path/filepath"
	"strings"
	"time"
)

func (w *TruthWriter) appendTruthEventLocked(event truthEvent) (TruthWriteResult, error) {
	normalized := normalizeTruthEvent(event)
	if normalized.OccurredAt.IsZero() {
		normalized.OccurredAt = time.Now().UTC()
	}
	if normalized.WrittenAt.IsZero() {
		normalized.WrittenAt = time.Now().UTC()
	}
	path := filepath.Join(w.eventsDir, normalized.OccurredAt.Format("2006-01-02")+".jsonl")
	if err := appendJSONLine(path, normalized); err != nil {
		return TruthWriteResult{}, err
	}
	if w.metrics != nil {
		w.metrics.truthEventsWritten.Add(1)
	}
	result := TruthWriteResult{
		SchemaVersion: truthSchemaVersion,
		EventID:       normalized.EventID,
		ObjectID:      normalized.ObjectID,
		ObjectType:    normalized.ObjectType,
		OccurredAt:    normalized.OccurredAt,
	}
	if normalized.Payload.Object != nil {
		result.ObjectCount = 1
		result.SourceRefCount += truthSingleObjectSourceRefCount(*normalized.Payload.Object)
	}
	if normalized.Payload.Evidence != nil {
		result.SourceRefCount += len(normalized.Payload.Evidence.SourceRefs)
	}
	if normalized.Payload.Claim != nil {
		result.ClaimCount = 1
		result.SourceRefCount += len(normalized.Payload.Claim.SourceRefs)
	}
	return result, nil
}

func (w *TruthWriter) AppendEvidenceEvent(eventType string, object MemoryObject, evidence MemoryEvidence, traceID string) (TruthWriteResult, error) {
	if !w.Enabled() {
		return TruthWriteResult{}, nil
	}
	normalizedObject := normalizeMemoryObject(object)
	normalizedEvidence := normalizeMemoryEvidence(evidence, normalizedObject.ObjectID)
	event := truthEvent{
		SchemaVersion: truthSchemaVersion,
		EventType:     truthCanonicalEventType(eventType, normalizedObject, normalizedEvidence),
		Namespace:     truthPrimaryNamespace(normalizedObject, normalizedEvidence.SourceRefs, nil),
		WorkspaceID:   truthPrimaryWorkspaceID(normalizedObject, normalizedEvidence.SourceRefs, nil),
		SessionID:     truthPrimarySessionID(normalizedObject.SourceRefs, normalizedEvidence.SourceRefs, nil),
		TurnID:        truthPrimaryTurnID(normalizedObject.SourceRefs, normalizedEvidence.SourceRefs, nil),
		TraceID:       strings.TrimSpace(traceID),
		OccurredAt:    effectiveDecisionTimestamp(normalizedEvidence.Timestamp, normalizedObject.UpdatedAt, normalizedObject.CreatedAt),
		ObjectID:      normalizedObject.ObjectID,
		ObjectType:    normalizedObject.ObjectType,
		Payload: truthEventPayload{
			Object:   &normalizedObject,
			Evidence: &normalizedEvidence,
		},
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureLayoutLocked(); err != nil {
		w.recordErrorLocked()
		return TruthWriteResult{}, err
	}
	return w.appendTruthEventLocked(event)
}

func (w *TruthWriter) AppendClaimEvent(claim MemoryClaim, traceID string) (TruthWriteResult, error) {
	if !w.Enabled() {
		return TruthWriteResult{}, nil
	}
	normalized := normalizeMemoryClaim(claim, claim.ObjectID)
	event := truthEvent{
		SchemaVersion: truthSchemaVersion,
		EventType:     truthEventTypeClaimAsserted,
		Namespace:     truthPrimaryNamespace(MemoryObject{}, normalized.SourceRefs, &normalized),
		WorkspaceID:   truthPrimaryWorkspaceID(MemoryObject{}, normalized.SourceRefs, &normalized),
		SessionID:     truthPrimarySessionID(nil, normalized.SourceRefs, &normalized),
		TurnID:        truthPrimaryTurnID(nil, normalized.SourceRefs, &normalized),
		TraceID:       strings.TrimSpace(traceID),
		OccurredAt:    effectiveDecisionTimestamp(normalized.AssertedAt, normalized.CreatedAt, normalized.ValidFrom),
		ObjectID:      normalized.ObjectID,
		Payload: truthEventPayload{
			Claim: &normalized,
		},
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureLayoutLocked(); err != nil {
		w.recordErrorLocked()
		return TruthWriteResult{}, err
	}
	return w.appendTruthEventLocked(event)
}

func (w *TruthWriter) AppendClaimStatusEvent(claim MemoryClaim, traceID string) (TruthWriteResult, error) {
	if !w.Enabled() {
		return TruthWriteResult{}, nil
	}
	normalized := normalizeMemoryClaim(claim, claim.ObjectID)
	event := truthEvent{
		SchemaVersion: truthSchemaVersion,
		EventType:     truthEventTypeClaimStatusChanged,
		Namespace:     truthPrimaryNamespace(MemoryObject{}, normalized.SourceRefs, &normalized),
		WorkspaceID:   truthPrimaryWorkspaceID(MemoryObject{}, normalized.SourceRefs, &normalized),
		SessionID:     truthPrimarySessionID(nil, normalized.SourceRefs, &normalized),
		TurnID:        truthPrimaryTurnID(nil, normalized.SourceRefs, &normalized),
		TraceID:       strings.TrimSpace(traceID),
		OccurredAt:    effectiveDecisionTimestamp(normalized.AssertedAt, normalized.CreatedAt),
		ObjectID:      normalized.ObjectID,
		Payload: truthEventPayload{
			Claim:      &normalized,
			ClaimRef:   &ClaimRef{ClaimID: normalized.ClaimID, Role: "status_target"},
			Status:     normalized.Status,
			ValidFrom:  normalized.ValidFrom,
			ValidTo:    normalized.ValidTo,
			Supersedes: append([]string(nil), normalized.Supersedes...),
		},
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.ensureLayoutLocked(); err != nil {
		w.recordErrorLocked()
		return TruthWriteResult{}, err
	}
	return w.appendTruthEventLocked(event)
}

func truthPrimaryNamespace(object MemoryObject, refs []SourceRef, claim *MemoryClaim) string {
	for _, ref := range refs {
		if strings.TrimSpace(ref.Namespace) != "" {
			return strings.TrimSpace(ref.Namespace)
		}
	}
	for _, ref := range object.SourceRefs {
		if strings.TrimSpace(ref.Namespace) != "" {
			return strings.TrimSpace(ref.Namespace)
		}
	}
	if claim != nil {
		for _, ref := range claim.SourceRefs {
			if strings.TrimSpace(ref.Namespace) != "" {
				return strings.TrimSpace(ref.Namespace)
			}
		}
	}
	return ""
}

func truthPrimaryWorkspaceID(object MemoryObject, refs []SourceRef, claim *MemoryClaim) string {
	for _, ref := range refs {
		if strings.TrimSpace(ref.WorkspaceID) != "" {
			return strings.TrimSpace(ref.WorkspaceID)
		}
	}
	for _, ref := range object.SourceRefs {
		if strings.TrimSpace(ref.WorkspaceID) != "" {
			return strings.TrimSpace(ref.WorkspaceID)
		}
	}
	if claim != nil {
		for _, ref := range claim.SourceRefs {
			if strings.TrimSpace(ref.WorkspaceID) != "" {
				return strings.TrimSpace(ref.WorkspaceID)
			}
		}
	}
	return ""
}

func truthPrimarySessionID(objectRefs []SourceRef, refs []SourceRef, claim *MemoryClaim) string {
	for _, ref := range refs {
		if strings.TrimSpace(ref.SessionID) != "" {
			return strings.TrimSpace(ref.SessionID)
		}
	}
	for _, ref := range objectRefs {
		if strings.TrimSpace(ref.SessionID) != "" {
			return strings.TrimSpace(ref.SessionID)
		}
	}
	if claim != nil {
		for _, ref := range claim.SourceRefs {
			if strings.TrimSpace(ref.SessionID) != "" {
				return strings.TrimSpace(ref.SessionID)
			}
		}
	}
	return ""
}

func truthPrimaryTurnID(objectRefs []SourceRef, refs []SourceRef, claim *MemoryClaim) string {
	for _, ref := range refs {
		if strings.TrimSpace(ref.TurnID) != "" {
			return strings.TrimSpace(ref.TurnID)
		}
	}
	for _, ref := range objectRefs {
		if strings.TrimSpace(ref.TurnID) != "" {
			return strings.TrimSpace(ref.TurnID)
		}
	}
	if claim != nil {
		for _, ref := range claim.SourceRefs {
			if strings.TrimSpace(ref.TurnID) != "" {
				return strings.TrimSpace(ref.TurnID)
			}
		}
	}
	return ""
}

func (w *TruthWriter) upsertClaimsLocked(claims []MemoryClaim) ClaimArbitrationResult {
	if w.claimSnapshot == nil {
		w.claimSnapshot = make(map[string]MemoryClaim)
	}
	result := truthArbitrateClaims(w.claimSnapshot, claims)
	if w.claimStatusProjectionEnabled {
		truthProjectClaimsOntoObjects(w.objectSnapshot, w.claimSnapshot, w.legacyObjectProjectionEnabled)
	}
	if w.metrics != nil {
		w.metrics.truthClaimsUpserted.Add(uint64(len(claims)))
	}
	return result
}

func (w *TruthWriter) appendClaimStatusEventsLocked(changes []MemoryClaim) error {
	for _, claim := range changes {
		event := truthEvent{
			SchemaVersion: truthSchemaVersion,
			EventType:     truthEventTypeClaimStatusChanged,
			Namespace:     truthPrimaryNamespace(MemoryObject{}, claim.SourceRefs, &claim),
			WorkspaceID:   truthPrimaryWorkspaceID(MemoryObject{}, claim.SourceRefs, &claim),
			SessionID:     truthPrimarySessionID(nil, claim.SourceRefs, &claim),
			TurnID:        truthPrimaryTurnID(nil, claim.SourceRefs, &claim),
			TraceID:       firstNonEmpty(claim.MetadataValue("trace_id"), firstTraceID(claim.SourceRefs)),
			OccurredAt:    effectiveDecisionTimestamp(claim.AssertedAt, claim.CreatedAt),
			ObjectID:      claim.ObjectID,
			Payload: truthEventPayload{
				Claim:      &claim,
				ClaimRef:   &ClaimRef{ClaimID: claim.ClaimID, Role: "status_target"},
				Status:     claim.Status,
				ValidFrom:  claim.ValidFrom,
				ValidTo:    claim.ValidTo,
				Supersedes: append([]string(nil), claim.Supersedes...),
			},
		}
		if _, err := w.appendTruthEventLocked(event); err != nil {
			return err
		}
	}
	return nil
}

func truthWriteResultFromSnapshots(objects map[string]MemoryObject, claims map[string]MemoryClaim) TruthWriteResult {
	active, conflicted, superseded, unverified := truthClaimStatusTotals(claims)
	return TruthWriteResult{
		SchemaVersion:  truthSchemaVersion,
		ObjectCount:    len(objects),
		ClaimCount:     len(claims),
		ActiveClaims:   active,
		Conflicts:      conflicted,
		Superseded:     superseded,
		Unverified:     unverified,
		SourceRefCount: truthSourceRefCount(objects, claims),
	}
}

func firstTraceID(refs []SourceRef) string {
	for _, ref := range refs {
		if strings.TrimSpace(ref.TraceID) != "" {
			return strings.TrimSpace(ref.TraceID)
		}
	}
	return ""
}

func (claim MemoryClaim) MetadataValue(key string) string {
	if len(claim.Metadata) == 0 {
		return ""
	}
	if value, ok := claim.Metadata[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}
