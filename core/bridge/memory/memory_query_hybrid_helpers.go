package memory

import (
	"fmt"
	"strings"
	"time"
)

func (s *QueryService) candidatesFromEntries(entries []MemoryEntry, layer string, query MemoryQuery, now time.Time) []RecallCandidate {
	if len(entries) == 0 {
		return nil
	}
	out := make([]RecallCandidate, 0, len(entries))
	for _, entry := range entries {
		out = append(out, s.candidateFromEntry(entry, layer, query, now))
	}
	return out
}

func (s *QueryService) candidateFromEntry(entry MemoryEntry, layer string, query MemoryQuery, now time.Time) RecallCandidate {
	entry = normalizeEntry(entry)
	if layer == "" {
		layer = entryLayer(entry)
	}
	sourceRefs := s.entrySourceRefsForHydration(entry)
	candidate := RecallCandidate{
		Entry:            entry,
		Layer:            strings.TrimSpace(layer),
		ObjectID:         s.objectIDFromEntry(entry),
		BaseScore:        clamp01(memoryScore(entry, query, now, s.scoring)),
		LexicalScore:     semanticRelevanceScore(entry, query),
		GraphScore:       metadataFloat64(entry.Metadata, "graph_score"),
		DecisionScore:    maxFloat(metadataFloat64(entry.Metadata, "decision_score"), metadataFloat64(entry.Metadata, "reuse_score")),
		EnvironmentScore: 0,
		ImportanceScore:  clamp01(entry.Importance),
		Freshness:        truthlessFreshness(entry, now),
		EvidenceCount:    max(entry.EvidenceCount, len(sourceRefs)),
		SourceRefs:       sourceRefs,
		WhyMatched:       candidateWhyMatched(entry),
		MatchedBy:        []string{strings.TrimSpace(layer)},
	}
	candidate.Entry.SourceRefs = normalizeSourceRefs(append(candidate.Entry.SourceRefs, sourceRefs...))
	return normalizeRecallCandidate(candidate)
}

func (s *QueryService) candidatesFromDecisionHits(hits []DecisionHit, query MemoryQuery, now time.Time) []RecallCandidate {
	if len(hits) == 0 {
		return nil
	}
	out := make([]RecallCandidate, 0, len(hits))
	for _, hit := range hits {
		entry := memoryEntryFromDecisionHit(hit)
		candidate := s.candidateFromEntry(entry, "decision", query, now)
		candidate.DecisionScore = maxFloat(hit.Score, hit.ReuseScore)
		candidate.WhyMatched = firstNonEmpty(strings.TrimSpace(hit.WhyMatched), candidate.WhyMatched)
		candidate.MatchedBy = append(candidate.MatchedBy, hit.Type)
		out = append(out, normalizeRecallCandidate(candidate))
	}
	return out
}

func (s *QueryService) candidatesFromGraphHits(hits []GraphHit, query MemoryQuery, now time.Time) []RecallCandidate {
	if len(hits) == 0 {
		return nil
	}
	out := make([]RecallCandidate, 0, len(hits))
	for _, hit := range hits {
		entry := memoryEntryFromGraphDebugHit(hit, now)
		candidate := s.candidateFromEntry(entry, "graph", query, now)
		candidate.GraphScore = clamp01(hit.Score)
		candidate.EvidenceCount = max(candidate.EvidenceCount, hit.EvidenceCount)
		candidate.WhyMatched = summarizeLine(strings.TrimSpace(hit.Content), 180)
		out = append(out, normalizeRecallCandidate(candidate))
	}
	return out
}

func (s *QueryService) candidatesFromTruthMatches(matches []truthQueryMatch, query MemoryQuery, now time.Time) []RecallCandidate {
	if len(matches) == 0 {
		return nil
	}
	out := make([]RecallCandidate, 0, len(matches))
	for _, match := range matches {
		entry := memoryEntryFromTruthObject(match.Object, "truth", now)
		candidate := s.candidateFromEntry(entry, "truth", query, now)
		candidate.ObjectID = match.Object.ObjectID
		candidate.TruthScore = clamp01(match.Score)
		candidate.EvidenceCount = max(candidate.EvidenceCount, len(match.Object.RawEvidence))
		candidate.SourceRefs = normalizeSourceRefs(append(candidate.SourceRefs, truthObjectSourceRefs(match.Object)...))
		candidate.Freshness = maxFloat(candidate.Freshness, truthObjectFreshness(match.Object, now))
		candidate.WhyMatched = firstNonEmpty(strings.TrimSpace(match.WhyMatched), candidate.WhyMatched)
		candidate.MatchedBy = append(candidate.MatchedBy, match.MatchedBy...)
		out = append(out, normalizeRecallCandidate(candidate))
	}
	return out
}

func (s *QueryService) candidatesFromVectorHits(hits []VectorHit, query MemoryQuery, now time.Time) []RecallCandidate {
	if len(hits) == 0 || s.truth == nil || !s.truth.Enabled() {
		return nil
	}
	out := make([]RecallCandidate, 0, len(hits))
	for _, hit := range hits {
		object, ok := s.truth.LookupObject(hit.ObjectID)
		if !ok {
			continue
		}
		entry := memoryEntryFromTruthObject(object, "vector", now)
		candidate := s.candidateFromEntry(entry, "vector", query, now)
		candidate.ObjectID = hit.ObjectID
		candidate.SemanticScore = clamp01(hit.Score)
		candidate.EvidenceCount = max(candidate.EvidenceCount, hit.EvidenceCount)
		candidate.SourceRefs = normalizeSourceRefs(append(candidate.SourceRefs, hit.SourceRefs...))
		candidate.Freshness = maxFloat(candidate.Freshness, clamp01(hit.Freshness))
		candidate.WhyMatched = firstNonEmpty(hit.Summary, candidate.WhyMatched)
		candidate.MatchedBy = append(candidate.MatchedBy, "vector")
		out = append(out, normalizeRecallCandidate(candidate))
		if s.metrics != nil {
			s.metrics.vectorPromotedHits.Add(1)
		}
	}
	return out
}

func mergeRecallCandidates(candidates []RecallCandidate) []RecallCandidate {
	if len(candidates) == 0 {
		return nil
	}
	merged := make(map[string]RecallCandidate, len(candidates))
	for _, candidate := range candidates {
		normalized := normalizeRecallCandidate(candidate)
		key := recallCandidateKey(normalized)
		if existing, ok := merged[key]; ok {
			merged[key] = mergeRecallCandidate(existing, normalized)
			continue
		}
		merged[key] = normalized
	}
	out := make([]RecallCandidate, 0, len(merged))
	for _, candidate := range merged {
		candidate.Layer = primaryRecallLayer(candidate)
		candidate.Entry.Source = candidate.Layer
		if candidate.Entry.Metadata == nil {
			candidate.Entry.Metadata = make(map[string]any, 2)
		}
		candidate.Entry.Metadata["layer"] = candidate.Layer
		candidate.Entry.Metadata["source"] = candidate.Layer
		out = append(out, normalizeRecallCandidate(candidate))
	}
	return out
}

func mergeRecallCandidate(left RecallCandidate, right RecallCandidate) RecallCandidate {
	merged := cloneRecallCandidate(left)
	if recallLayerPriority(right.Layer) > recallLayerPriority(merged.Layer) || right.RerankScore > merged.RerankScore || len(strings.TrimSpace(merged.Entry.Summary)) == 0 {
		merged.Entry = cloneEntry(right.Entry)
		merged.Layer = right.Layer
	}
	if merged.ObjectID == "" {
		merged.ObjectID = right.ObjectID
	}
	merged.BaseScore = maxFloat(merged.BaseScore, right.BaseScore)
	merged.LexicalScore = maxFloat(merged.LexicalScore, right.LexicalScore)
	merged.SemanticScore = maxFloat(merged.SemanticScore, right.SemanticScore)
	merged.GraphScore = maxFloat(merged.GraphScore, right.GraphScore)
	merged.DecisionScore = maxFloat(merged.DecisionScore, right.DecisionScore)
	merged.TruthScore = maxFloat(merged.TruthScore, right.TruthScore)
	merged.EnvironmentScore = maxFloat(merged.EnvironmentScore, right.EnvironmentScore)
	merged.ImportanceScore = maxFloat(merged.ImportanceScore, right.ImportanceScore)
	merged.Freshness = maxFloat(merged.Freshness, right.Freshness)
	merged.EvidenceCount = max(merged.EvidenceCount, right.EvidenceCount)
	merged.SourceRefs = normalizeSourceRefs(append(merged.SourceRefs, right.SourceRefs...))
	merged.ConflictCount = max(merged.ConflictCount, right.ConflictCount)
	merged.SupportCount = max(merged.SupportCount, right.SupportCount)
	merged.MatchedBy = uniqueStrings(append(merged.MatchedBy, right.MatchedBy...))
	merged.WhyMatched = joinCandidateReasons(merged.WhyMatched, right.WhyMatched)
	merged.TruthStatus = strongerTruthStatus(merged.TruthStatus, right.TruthStatus)
	return normalizeRecallCandidate(merged)
}

func finalizeCandidateEntry(candidate RecallCandidate) MemoryEntry {
	entry := cloneEntry(candidate.Entry)
	entry.Source = candidate.Layer
	if entry.Metadata == nil {
		entry.Metadata = make(map[string]any, 2)
	}
	entry.Metadata["layer"] = candidate.Layer
	entry.Metadata["source"] = candidate.Layer
	if candidate.ObjectID != "" {
		entry.Metadata["object_id"] = candidate.ObjectID
	}
	entry.Confidence = clamp01(entry.Confidence)
	entry.Freshness = candidate.Freshness
	entry.EvidenceCount = candidate.EvidenceCount
	entry.SourceRefs = normalizeSourceRefs(append(entry.SourceRefs, candidate.SourceRefs...))
	entry.TruthStatus = normalizeTruthStatus(candidate.TruthStatus)
	entry.WhyMatched = candidate.WhyMatched
	entry.RerankScore = candidate.RerankScore
	return normalizeEntry(entry)
}

func memoryEntryFromTruthObject(object MemoryObject, layer string, now time.Time) MemoryEntry {
	metadata := map[string]any{
		"layer":       strings.TrimSpace(layer),
		"source":      strings.TrimSpace(layer),
		"object_id":   strings.TrimSpace(object.ObjectID),
		"object_type": strings.TrimSpace(object.ObjectType),
	}
	summary := truthObjectSummary(object)
	content := summary
	if len(object.RawEvidence) > 0 {
		content = firstNonEmpty(strings.TrimSpace(object.RawEvidence[0].Summary), strings.TrimSpace(object.RawEvidence[0].Text), summary)
	}
	return normalizeEntry(MemoryEntry{
		ID:            strings.TrimSpace(layer) + ":" + strings.TrimSpace(object.ObjectID),
		Content:       strings.TrimSpace(content),
		Summary:       summary,
		Type:          MemoryTypeKnowledge,
		Timestamp:     effectiveDecisionTimestamp(object.UpdatedAt, object.CreatedAt, now),
		Importance:    clamp01(maxFloat(object.Confidence, truthObjectFreshness(object, now))),
		Source:        strings.TrimSpace(layer),
		Confidence:    clamp01(object.Confidence),
		Freshness:     truthObjectFreshness(object, now),
		EvidenceCount: len(object.RawEvidence),
		SourceRefs:    truthObjectSourceRefs(object),
		Metadata:      metadata,
	})
}

func memoryEntryFromGraphDebugHit(hit GraphHit, now time.Time) MemoryEntry {
	timestamp := now
	for _, evidence := range hit.Evidence {
		timestamp = effectiveDecisionTimestamp(evidence.Timestamp, timestamp)
	}
	metadata := map[string]any{
		"layer":       "graph",
		"source":      "graph",
		"edge_id":     strings.TrimSpace(hit.EdgeID),
		"node_id":     strings.TrimSpace(hit.NodeID),
		"predicate":   strings.TrimSpace(hit.Predicate),
		"hop":         hit.Hop,
		"graph_score": clamp01(hit.Score),
		"status":      strings.TrimSpace(hit.Status),
		"subject":     strings.TrimSpace(hit.Subject),
		"object":      strings.TrimSpace(hit.Object),
		"source_ids":  append([]string(nil), hit.SourceIDs...),
	}
	return normalizeEntry(MemoryEntry{
		ID:         firstNonEmpty("graph:"+strings.TrimSpace(hit.EdgeID), "graph:"+strings.ReplaceAll(normalizeRecallText(hit.Content), " ", "-")),
		Content:    strings.TrimSpace(hit.Content),
		Summary:    summarizeLine(strings.TrimSpace(hit.Content), 180),
		Type:       MemoryTypeKnowledge,
		Timestamp:  timestamp,
		Importance: clamp01(hit.Score),
		Source:     "graph",
		Confidence: clamp01(hit.Score),
		Metadata:   metadata,
	})
}

func (s *QueryService) entrySourceRefsForHydration(entry MemoryEntry) []SourceRef {
	refs := append([]SourceRef(nil), entry.SourceRefs...)
	sessionID := metadataString(entry.Metadata, "session_id")
	if entryLayer(entry) == "cold" || strings.EqualFold(strings.TrimSpace(entry.Source), "archive") {
		refs = append(refs, SourceRef{SessionID: sessionID, SourceKind: truthSourceKindArchiveMessage, SourceID: entry.ID})
	}
	return normalizeSourceRefs(refs)
}

func (s *QueryService) objectIDFromEntry(entry MemoryEntry) string {
	if s.truth == nil || !s.truth.Enabled() {
		return ""
	}
	if objectID := firstNonEmpty(metadataString(entry.Metadata, "object_id"), metadataString(entry.Explain, "object_id")); objectID != "" {
		return objectID
	}
	for _, claimID := range entryClaimLineageIDs(entry) {
		if objectID := s.truth.resolvePrimaryObjectIDByClaimID(claimID); objectID != "" {
			return objectID
		}
	}
	for _, evidenceID := range entryEvidenceLineageIDs(entry) {
		if objectID := s.truth.resolvePrimaryObjectIDByEvidenceID(evidenceID); objectID != "" {
			return objectID
		}
	}
	for _, ref := range s.entrySourceRefsForHydration(entry) {
		if objectID := s.truth.ResolvePrimaryObjectIDBySourceRef(ref); objectID != "" {
			return objectID
		}
	}
	return ""
}

func recallCandidateKey(candidate RecallCandidate) string {
	if projectionType := recallProjectionType(candidate); projectionType != "" {
		projectionID := projectionIDFromEntry(candidate.Entry, projectionType)
		if projectionID != "" {
			return "projection:" + projectionType + ":" + projectionID
		}
	}
	if candidate.ObjectID != "" {
		return "object:" + candidate.ObjectID
	}
	if sessionID := metadataString(candidate.Entry.Metadata, "session_id"); sessionID != "" {
		return "session:" + sessionID + ":" + candidate.Entry.ID
	}
	return "fingerprint:" + normalizeRecallText(firstNonEmpty(candidate.Entry.Summary, candidate.Entry.Content, candidate.Entry.ID))
}

func recallProjectionType(candidate RecallCandidate) string {
	for _, layer := range []string{candidate.Layer, entryLayer(candidate.Entry), candidate.Entry.Source} {
		switch strings.TrimSpace(layer) {
		case "markdown", "decision", "graph":
			return strings.TrimSpace(layer)
		}
	}
	return ""
}

func entryClaimLineageIDs(entry MemoryEntry) []string {
	keys := []string{"source_claim_ids", "matched_claim_ids", "derived_claim_ids", "conflicted_claim_ids"}
	values := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		values = append(values, metadataStrings(entry.Metadata, key)...)
		values = append(values, metadataStrings(entry.Explain, key)...)
	}
	return uniqueStrings(values)
}

func entryEvidenceLineageIDs(entry MemoryEntry) []string {
	keys := []string{"source_evidence_ids", "matched_evidence_ids"}
	values := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		values = append(values, metadataStrings(entry.Metadata, key)...)
		values = append(values, metadataStrings(entry.Explain, key)...)
	}
	return uniqueStrings(values)
}

func primaryRecallLayer(candidate RecallCandidate) string {
	layer := strings.TrimSpace(candidate.Layer)
	for _, marker := range candidate.MatchedBy {
		if recallLayerPriority(marker) > recallLayerPriority(layer) {
			layer = marker
		}
	}
	if layer == "" {
		layer = entryLayer(candidate.Entry)
	}
	if layer == "" {
		layer = "truth"
	}
	return layer
}

func recallLayerPriority(layer string) int {
	switch strings.TrimSpace(layer) {
	case "decision":
		return 80
	case "graph":
		return 70
	case "truth":
		return 60
	case "vector":
		return 50
	case "markdown":
		return 40
	case "warm":
		return 30
	case "cold":
		return 20
	case "hot":
		return 10
	default:
		return 0
	}
}

func strongerTruthStatus(left string, right string) string {
	left = normalizeTruthStatus(left)
	right = normalizeTruthStatus(right)
	weight := func(status string) int {
		switch status {
		case truthStatusConflicted:
			return 4
		case truthStatusVerified:
			return 3
		case truthStatusSupported:
			return 2
		default:
			return 1
		}
	}
	if weight(right) > weight(left) {
		return right
	}
	return left
}

func joinCandidateReasons(parts ...string) string {
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		items = append(items, trimmed)
	}
	return strings.Join(uniqueStrings(items), "; ")
}

func candidateWhyMatched(entry MemoryEntry) string {
	if entry.WhyMatched != "" {
		return strings.TrimSpace(entry.WhyMatched)
	}
	if value := metadataString(entry.Metadata, "decision_why_matched"); value != "" {
		return value
	}
	if value := metadataString(entry.Metadata, "predicate"); value != "" {
		subject := metadataString(entry.Metadata, "subject")
		object := metadataString(entry.Metadata, "object")
		return strings.TrimSpace(fmt.Sprintf("%s %s %s", subject, value, object))
	}
	return summarizeLine(firstNonEmpty(entry.Summary, entry.Content), 160)
}

func metadataFloat64(metadata map[string]any, key string) float64 {
	if len(metadata) == 0 {
		return 0
	}
	raw, ok := metadata[key]
	if !ok {
		return 0
	}
	switch typed := raw.(type) {
	case float64:
		return clamp01(typed)
	case float32:
		return clamp01(float64(typed))
	case int:
		return clamp01(float64(typed))
	case int64:
		return clamp01(float64(typed))
	default:
		return 0
	}
}
