package memory

import "strings"

// TruthQueryOptions 是 truth-first 主链的标准查询切面；Phase 0 先只做内部适配。
type TruthQueryOptions struct {
	Subject           string `json:"subject,omitempty"`
	Predicate         string `json:"predicate,omitempty"`
	Object            string `json:"object,omitempty"`
	Value             string `json:"value,omitempty"`
	IncludeHistorical bool   `json:"include_historical,omitempty"`
	ActiveOnly        bool   `json:"active_only,omitempty"`
	ExposeConflicts   bool   `json:"expose_conflicts,omitempty"`
	Explain           bool   `json:"explain,omitempty"`
}

// TruthQueryExplain 聚合 truth-primary 命中的 lineage explain。
type TruthQueryExplain struct {
	MatchedClaimIDs    []string        `json:"matched_claim_ids,omitempty"`
	MatchedEvidenceIDs []string        `json:"matched_evidence_ids,omitempty"`
	ConflictedClaimIDs []string        `json:"conflicted_claim_ids,omitempty"`
	ProjectionSources  []ProjectionHit `json:"projection_sources,omitempty"`
	WhyMatched         string          `json:"why_matched,omitempty"`
}

// PrimaryCandidate 是 truth-first cutover 的主候选壳；当前仅作内部适配层。
type PrimaryCandidate struct {
	ObjectID           string            `json:"object_id,omitempty"`
	MatchedClaimIDs    []string          `json:"matched_claim_ids,omitempty"`
	MatchedEvidenceIDs []string          `json:"matched_evidence_ids,omitempty"`
	ConflictedClaimIDs []string          `json:"conflicted_claim_ids,omitempty"`
	Score              float64           `json:"score,omitempty"`
	WhyMatched         string            `json:"why_matched,omitempty"`
	Explain            TruthQueryExplain `json:"explain,omitempty"`
}

// ProjectionHit 是 truth primary 之上的可读 projection 命中壳。
type ProjectionHit struct {
	ProjectionType    string         `json:"projection_type,omitempty"`
	ProjectionID      string         `json:"projection_id,omitempty"`
	ObjectID          string         `json:"object_id,omitempty"`
	SourceClaimIDs    []string       `json:"source_claim_ids,omitempty"`
	SourceEvidenceIDs []string       `json:"source_evidence_ids,omitempty"`
	Summary           string         `json:"summary,omitempty"`
	Explain           map[string]any `json:"explain,omitempty"`
}

func normalizeTruthQueryOptions(options TruthQueryOptions) TruthQueryOptions {
	out := options
	out.Subject = strings.TrimSpace(out.Subject)
	out.Predicate = strings.TrimSpace(out.Predicate)
	out.Object = strings.TrimSpace(out.Object)
	out.Value = strings.TrimSpace(out.Value)
	if out.IncludeHistorical {
		out.ActiveOnly = false
	}
	if !out.IncludeHistorical && !out.ActiveOnly {
		out.ActiveOnly = true
	}
	return out
}

func normalizeTruthQueryExplain(explain TruthQueryExplain) TruthQueryExplain {
	out := explain
	out.MatchedClaimIDs = uniqueStrings(out.MatchedClaimIDs)
	out.MatchedEvidenceIDs = uniqueStrings(out.MatchedEvidenceIDs)
	out.ConflictedClaimIDs = uniqueStrings(out.ConflictedClaimIDs)
	out.ProjectionSources = cloneProjectionHits(out.ProjectionSources)
	out.WhyMatched = strings.TrimSpace(out.WhyMatched)
	return out
}

func cloneTruthQueryExplain(explain TruthQueryExplain) TruthQueryExplain {
	out := explain
	out.MatchedClaimIDs = append([]string(nil), explain.MatchedClaimIDs...)
	out.MatchedEvidenceIDs = append([]string(nil), explain.MatchedEvidenceIDs...)
	out.ConflictedClaimIDs = append([]string(nil), explain.ConflictedClaimIDs...)
	out.ProjectionSources = cloneProjectionHits(explain.ProjectionSources)
	return out
}

func normalizePrimaryCandidate(candidate PrimaryCandidate) PrimaryCandidate {
	out := candidate
	out.ObjectID = strings.TrimSpace(out.ObjectID)
	out.MatchedClaimIDs = uniqueStrings(out.MatchedClaimIDs)
	out.MatchedEvidenceIDs = uniqueStrings(out.MatchedEvidenceIDs)
	out.ConflictedClaimIDs = uniqueStrings(out.ConflictedClaimIDs)
	out.Score = clamp01(out.Score)
	out.WhyMatched = strings.TrimSpace(out.WhyMatched)
	out.Explain = normalizeTruthQueryExplain(out.Explain)
	if out.Explain.WhyMatched == "" {
		out.Explain.WhyMatched = out.WhyMatched
	}
	if len(out.Explain.MatchedClaimIDs) == 0 {
		out.Explain.MatchedClaimIDs = append([]string(nil), out.MatchedClaimIDs...)
	}
	if len(out.Explain.MatchedEvidenceIDs) == 0 {
		out.Explain.MatchedEvidenceIDs = append([]string(nil), out.MatchedEvidenceIDs...)
	}
	if len(out.Explain.ConflictedClaimIDs) == 0 {
		out.Explain.ConflictedClaimIDs = append([]string(nil), out.ConflictedClaimIDs...)
	}
	return out
}

func clonePrimaryCandidate(candidate PrimaryCandidate) PrimaryCandidate {
	out := candidate
	out.MatchedClaimIDs = append([]string(nil), candidate.MatchedClaimIDs...)
	out.MatchedEvidenceIDs = append([]string(nil), candidate.MatchedEvidenceIDs...)
	out.ConflictedClaimIDs = append([]string(nil), candidate.ConflictedClaimIDs...)
	out.Explain = cloneTruthQueryExplain(candidate.Explain)
	return out
}

func clonePrimaryCandidates(candidates []PrimaryCandidate) []PrimaryCandidate {
	if len(candidates) == 0 {
		return nil
	}
	out := make([]PrimaryCandidate, len(candidates))
	for i := range candidates {
		out[i] = clonePrimaryCandidate(candidates[i])
	}
	return out
}

func normalizeProjectionHit(hit ProjectionHit) ProjectionHit {
	out := hit
	out.ProjectionType = strings.TrimSpace(out.ProjectionType)
	out.ProjectionID = strings.TrimSpace(out.ProjectionID)
	out.ObjectID = strings.TrimSpace(out.ObjectID)
	out.SourceClaimIDs = uniqueStrings(out.SourceClaimIDs)
	out.SourceEvidenceIDs = uniqueStrings(out.SourceEvidenceIDs)
	out.Summary = strings.TrimSpace(out.Summary)
	out.Explain = cloneMetadata(out.Explain)
	return out
}

func cloneProjectionHit(hit ProjectionHit) ProjectionHit {
	out := hit
	out.SourceClaimIDs = append([]string(nil), hit.SourceClaimIDs...)
	out.SourceEvidenceIDs = append([]string(nil), hit.SourceEvidenceIDs...)
	out.Explain = cloneMetadata(hit.Explain)
	return out
}

func cloneProjectionHits(hits []ProjectionHit) []ProjectionHit {
	if len(hits) == 0 {
		return nil
	}
	out := make([]ProjectionHit, len(hits))
	for i := range hits {
		out[i] = cloneProjectionHit(hits[i])
	}
	return out
}

func truthQueryOptionsFromMemoryQuery(query MemoryQuery, plan *QueryIntentPlan) TruthQueryOptions {
	options := TruthQueryOptions{
		Subject:           firstNonEmpty(metadataString(query.Metadata, "subject"), metadataString(query.Metadata, "truth_subject")),
		Predicate:         firstNonEmpty(metadataString(query.Metadata, "predicate"), metadataString(query.Metadata, "truth_predicate"), firstNonEmpty(query.DecisionTypes...)),
		Object:            firstNonEmpty(metadataString(query.Metadata, "object"), metadataString(query.Metadata, "truth_object")),
		Value:             firstNonEmpty(metadataString(query.Metadata, "value"), metadataString(query.Metadata, "truth_value"), query.SemanticQuery),
		IncludeHistorical: metadataBool(query.Metadata, "truth_include_historical"),
		ExposeConflicts:   metadataBool(query.Metadata, "truth_expose_conflicts") || query.TruthDebug,
		Explain:           query.TruthDebug || query.RerankDebug,
	}
	if activeOnly, ok := metadataBoolOK(query.Metadata, "truth_active_only"); ok {
		options.ActiveOnly = activeOnly
	} else {
		options.ActiveOnly = !options.IncludeHistorical
	}
	if plan != nil {
		if options.Subject == "" {
			options.Subject = firstNonEmpty(plan.Entities...)
		}
		if options.Predicate == "" {
			options.Predicate = firstNonEmpty(plan.Constraints...)
		}
		if options.Value == "" {
			options.Value = firstNonEmpty(plan.Terms...)
		}
	}
	return normalizeTruthQueryOptions(options)
}

func primaryCandidateFromTruthMatch(match truthQueryMatch) PrimaryCandidate {
	matchedClaimIDs := truthMatchedClaimIDs(match.Object.Claims)
	matchedEvidenceIDs := truthMatchedEvidenceIDs(match.Object)
	conflictedClaimIDs := truthConflictedClaimIDs(match.Object.Claims)
	return normalizePrimaryCandidate(PrimaryCandidate{
		ObjectID:           match.Object.ObjectID,
		MatchedClaimIDs:    matchedClaimIDs,
		MatchedEvidenceIDs: matchedEvidenceIDs,
		ConflictedClaimIDs: conflictedClaimIDs,
		Score:              match.Score,
		WhyMatched:         match.WhyMatched,
		Explain: TruthQueryExplain{
			MatchedClaimIDs:    matchedClaimIDs,
			MatchedEvidenceIDs: matchedEvidenceIDs,
			ConflictedClaimIDs: conflictedClaimIDs,
			WhyMatched:         match.WhyMatched,
		},
	})
}

func truthPrimaryCandidatesFromMatches(matches []truthQueryMatch) []PrimaryCandidate {
	if len(matches) == 0 {
		return nil
	}
	out := make([]PrimaryCandidate, 0, len(matches))
	for _, match := range matches {
		out = append(out, primaryCandidateFromTruthMatch(match))
	}
	return out
}

func projectionHitFromEntry(entry MemoryEntry) (ProjectionHit, bool) {
	entry = normalizeEntry(entry)
	projectionType := projectionTypeFromEntry(entry)
	if projectionType == "" {
		return ProjectionHit{}, false
	}
	explain := projectionExplainFromEntry(entry, projectionType)
	hit := ProjectionHit{
		ProjectionType:    projectionType,
		ProjectionID:      projectionIDFromEntry(entry, projectionType),
		ObjectID:          firstNonEmpty(metadataString(entry.Metadata, "object_id"), metadataString(explain, "object_id")),
		SourceClaimIDs:    projectionClaimIDs(entry, explain),
		SourceEvidenceIDs: projectionEvidenceIDs(entry, explain),
		Summary:           firstNonEmpty(entry.Summary, compactRecallLine(entry), entry.Content),
		Explain:           explain,
	}
	if hit.ProjectionID == "" {
		hit.ProjectionID = entry.ID
	}
	return normalizeProjectionHit(hit), true
}

func projectionHitsFromEntries(entries []MemoryEntry) []ProjectionHit {
	if len(entries) == 0 {
		return nil
	}
	out := make([]ProjectionHit, 0, len(entries))
	for _, entry := range entries {
		hit, ok := projectionHitFromEntry(entry)
		if !ok {
			continue
		}
		out = append(out, hit)
	}
	return out
}

func projectionTypeFromEntry(entry MemoryEntry) string {
	switch layer := entryLayer(entry); layer {
	case "markdown", "decision", "graph":
		return layer
	}
	switch strings.TrimSpace(entry.Source) {
	case "markdown", "decision", "graph":
		return strings.TrimSpace(entry.Source)
	}
	if metadataString(entry.Metadata, "node_id") != "" {
		return "markdown"
	}
	if metadataString(entry.Metadata, "memo_id") != "" || metadataString(entry.Metadata, "recipe_id") != "" {
		return "decision"
	}
	if metadataString(entry.Metadata, "edge_id") != "" || metadataString(entry.Metadata, "predicate") != "" {
		return "graph"
	}
	return ""
}

func projectionIDFromEntry(entry MemoryEntry, projectionType string) string {
	switch projectionType {
	case "markdown":
		return firstNonEmpty(metadataString(entry.Metadata, "node_id"), entry.ID)
	case "decision":
		return firstNonEmpty(metadataString(entry.Metadata, "memo_id"), metadataString(entry.Metadata, "recipe_id"), entry.ID)
	case "graph":
		return firstNonEmpty(metadataString(entry.Metadata, "edge_id"), metadataString(entry.Metadata, "node_id"), entry.ID)
	default:
		return strings.TrimSpace(entry.ID)
	}
}

func projectionClaimIDs(entry MemoryEntry, explain map[string]any) []string {
	values := metadataStrings(entry.Metadata, "source_claim_ids")
	values = append(values, metadataStrings(entry.Metadata, "matched_claim_ids")...)
	values = append(values, metadataStrings(entry.Metadata, "derived_claim_ids")...)
	values = append(values, metadataStrings(explain, "source_claim_ids")...)
	values = append(values, metadataStrings(explain, "matched_claim_ids")...)
	values = append(values, metadataStrings(explain, "derived_claim_ids")...)
	return uniqueStrings(values)
}

func projectionEvidenceIDs(entry MemoryEntry, explain map[string]any) []string {
	values := metadataStrings(entry.Metadata, "source_evidence_ids")
	values = append(values, metadataStrings(entry.Metadata, "matched_evidence_ids")...)
	values = append(values, metadataStrings(explain, "source_evidence_ids")...)
	values = append(values, metadataStrings(explain, "matched_evidence_ids")...)
	return uniqueStrings(values)
}

func projectionExplainFromEntry(entry MemoryEntry, projectionType string) map[string]any {
	explain := cloneMetadata(entry.Explain)
	if explain == nil {
		explain = map[string]any{}
	}
	explain["projection_type"] = projectionType
	explain["projection_id"] = projectionIDFromEntry(entry, projectionType)
	explain["layer"] = firstNonEmpty(metadataString(entry.Metadata, "layer"), entry.Source, projectionType)
	for _, key := range []string{
		"object_id",
		"source_claim_ids",
		"source_evidence_ids",
		"matched_claim_ids",
		"matched_evidence_ids",
		"conflicted_claim_ids",
		"derived_claim_ids",
		"legacy_source_ids",
	} {
		mergeProjectionExplainList(explain, key, metadataStrings(entry.Metadata, key))
	}
	if legacy := metadataStrings(entry.Metadata, "source_ids"); len(legacy) > 0 && len(metadataStrings(explain, "legacy_source_ids")) == 0 {
		explain["legacy_source_ids"] = legacy
	}
	for _, key := range []string{"memo_id", "recipe_id", "node_id", "edge_id", "predicate", "subject", "object", "decision_why_matched", "decision_hit_type"} {
		if value := metadataString(entry.Metadata, key); value != "" {
			explain[key] = value
		}
	}
	if summary := strings.TrimSpace(entry.Summary); summary != "" {
		explain["summary"] = summary
	}
	return explain
}

func mergeProjectionExplainList(explain map[string]any, key string, values []string) {
	if len(values) == 0 {
		return
	}
	merged := append(metadataStrings(explain, key), values...)
	if len(merged) == 0 {
		return
	}
	explain[key] = uniqueStrings(merged)
}

func truthMatchedClaimIDs(claims []MemoryClaim) []string {
	if len(claims) == 0 {
		return nil
	}
	out := make([]string, 0, len(claims))
	for _, claim := range claims {
		if strings.TrimSpace(claim.ClaimID) == "" {
			continue
		}
		if normalizeTruthClaimStatus(claim.Status) == truthClaimStatusSuperseded {
			continue
		}
		out = append(out, claim.ClaimID)
	}
	return uniqueStrings(out)
}

func truthConflictedClaimIDs(claims []MemoryClaim) []string {
	if len(claims) == 0 {
		return nil
	}
	out := make([]string, 0, len(claims))
	for _, claim := range claims {
		if normalizeTruthClaimStatus(claim.Status) != truthClaimStatusConflicted {
			continue
		}
		out = append(out, claim.ClaimID)
	}
	return uniqueStrings(out)
}

func truthMatchedEvidenceIDs(object MemoryObject) []string {
	values := make([]string, 0, len(object.RawEvidence)+len(object.Claims))
	for _, evidence := range object.RawEvidence {
		values = append(values, evidence.EvidenceID)
	}
	for _, claim := range object.Claims {
		if normalizeTruthClaimStatus(claim.Status) == truthClaimStatusSuperseded {
			continue
		}
		for _, ref := range claim.EvidenceRefs {
			values = append(values, ref.EvidenceID)
		}
	}
	return uniqueStrings(values)
}

func metadataBool(metadata map[string]any, key string) bool {
	value, _ := metadataBoolOK(metadata, key)
	return value
}

func metadataBoolOK(metadata map[string]any, key string) (bool, bool) {
	if len(metadata) == 0 {
		return false, false
	}
	raw, ok := metadata[key]
	if !ok {
		return false, false
	}
	switch typed := raw.(type) {
	case bool:
		return typed, true
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "on":
			return true, true
		case "0", "false", "no", "off":
			return false, true
		}
	}
	return false, false
}
