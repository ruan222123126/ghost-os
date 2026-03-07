package memory

import (
	"strings"
	"time"
)

func normalizeGraphNode(node GraphNode) GraphNode {
	out := node
	out.ID = strings.TrimSpace(out.ID)
	out.Namespace = normalizeGraphNamespace(out.Namespace)
	out.CanonicalName = cleanGraphDisplayName(out.CanonicalName)
	out.Type = normalizeGraphNodeType(out.Type)
	out.Aliases = normalizeGraphAliases(out.Aliases, out.CanonicalName)
	out.Summary = strings.TrimSpace(out.Summary)
	out.AnchorKeys = normalizeGraphAliases(out.AnchorKeys)
	out.Confidence = clamp01(out.Confidence)
	out.SourceIDs = uniqueStrings(out.SourceIDs)
	if out.FirstSeenAt.IsZero() {
		out.FirstSeenAt = time.Now().UTC()
	} else {
		out.FirstSeenAt = out.FirstSeenAt.UTC()
	}
	if out.LastSeenAt.IsZero() {
		out.LastSeenAt = out.FirstSeenAt.UTC()
	} else {
		out.LastSeenAt = out.LastSeenAt.UTC()
	}
	if out.MentionCount < 0 {
		out.MentionCount = 0
	}
	if out.Metadata == nil {
		out.Metadata = make(map[string]any)
	}
	return out
}

func normalizeGraphEdge(edge GraphEdge) GraphEdge {
	out := edge
	out.ID = strings.TrimSpace(out.ID)
	out.Namespace = normalizeGraphNamespace(out.Namespace)
	out.SubjectID = strings.TrimSpace(out.SubjectID)
	out.Predicate = normalizeGraphPredicate(out.Predicate)
	out.ObjectID = strings.TrimSpace(out.ObjectID)
	out.Confidence = clamp01(out.Confidence)
	if out.Weight < 0 {
		out.Weight = 0
	}
	out.Status = normalizeGraphStatus(out.Status)
	if out.FirstSeenAt.IsZero() {
		out.FirstSeenAt = time.Now().UTC()
	} else {
		out.FirstSeenAt = out.FirstSeenAt.UTC()
	}
	if out.LastSeenAt.IsZero() {
		out.LastSeenAt = out.FirstSeenAt.UTC()
	} else {
		out.LastSeenAt = out.LastSeenAt.UTC()
	}
	if !out.ExpiresAt.IsZero() {
		out.ExpiresAt = out.ExpiresAt.UTC()
	}
	out.SessionIDs = uniqueStrings(out.SessionIDs)
	out.SourceIDs = uniqueStrings(out.SourceIDs)
	out.Evidence = normalizeGraphEvidence(out.Evidence)
	return out
}

func normalizeGraphEvidence(evidence []GraphEvidence) []GraphEvidence {
	if len(evidence) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(evidence))
	out := make([]GraphEvidence, 0, len(evidence))
	for _, item := range evidence {
		normalized := GraphEvidence{
			SessionID: strings.TrimSpace(item.SessionID),
			SourceID:  strings.TrimSpace(item.SourceID),
			Snippet:   summarizeLine(strings.TrimSpace(item.Snippet), 220),
			TraceID:   strings.TrimSpace(item.TraceID),
		}
		if item.Timestamp.IsZero() {
			normalized.Timestamp = time.Now().UTC()
		} else {
			normalized.Timestamp = item.Timestamp.UTC()
		}
		fingerprint := strings.Join([]string{
			normalized.SessionID,
			normalized.SourceID,
			normalizeGraphLookup(normalized.Snippet),
		}, "|")
		if _, ok := seen[fingerprint]; ok {
			continue
		}
		seen[fingerprint] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) > defaultGraphEvidenceLimit {
		out = out[len(out)-defaultGraphEvidenceLimit:]
	}
	return out
}

func normalizeGraphFact(fact GraphFact) GraphFact {
	out := fact
	out.Namespace = normalizeGraphNamespace(out.Namespace)
	out.Subject = cleanGraphDisplayName(out.Subject)
	out.SubjectType = normalizeGraphNodeType(out.SubjectType)
	out.Predicate = normalizeGraphPredicate(out.Predicate)
	out.Object = cleanGraphDisplayName(out.Object)
	out.ObjectType = normalizeGraphNodeType(out.ObjectType)
	out.Aliases = normalizeGraphAliases(out.Aliases)
	out.Confidence = clamp01(out.Confidence)
	out.Snippet = summarizeLine(strings.TrimSpace(out.Snippet), 220)
	out.AnchorKeys = normalizeGraphAliases(out.AnchorKeys)
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.SourceID = strings.TrimSpace(out.SourceID)
	out.TraceID = strings.TrimSpace(out.TraceID)
	if out.Timestamp.IsZero() {
		out.Timestamp = time.Now().UTC()
	} else {
		out.Timestamp = out.Timestamp.UTC()
	}
	if out.Metadata == nil {
		out.Metadata = make(map[string]any)
	}
	return out
}

func normalizeGraphNamespace(namespace string) string {
	trimmed := strings.TrimSpace(namespace)
	if trimmed == "" {
		return defaultGraphNamespace
	}
	return trimmed
}

func normalizeGraphPredicate(predicate string) string {
	trimmed := strings.ToLower(strings.TrimSpace(predicate))
	if _, ok := allowedGraphPredicates[trimmed]; ok {
		return trimmed
	}
	return ""
}

func normalizeGraphStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case GraphStatusConflicted:
		return GraphStatusConflicted
	case GraphStatusSuperseded:
		return GraphStatusSuperseded
	default:
		return GraphStatusActive
	}
}

func normalizeGraphNodeType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case GraphNodeTypePerson:
		return GraphNodeTypePerson
	case GraphNodeTypeProject:
		return GraphNodeTypeProject
	case GraphNodeTypeOrg:
		return GraphNodeTypeOrg
	case GraphNodeTypeRepo:
		return GraphNodeTypeRepo
	case GraphNodeTypeTech:
		return GraphNodeTypeTech
	case GraphNodeTypePreference:
		return GraphNodeTypePreference
	default:
		return GraphNodeTypeTopic
	}
}

func normalizeGraphAliases(values []string, extra ...string) []string {
	combined := make([]string, 0, len(values)+len(extra))
	combined = append(combined, values...)
	combined = append(combined, extra...)
	if len(combined) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(combined))
	out := make([]string, 0, len(combined))
	for _, value := range combined {
		trimmed := cleanGraphDisplayName(value)
		if trimmed == "" {
			continue
		}
		lookup := normalizeGraphLookup(trimmed)
		if lookup == "" {
			continue
		}
		if _, ok := seen[lookup]; ok {
			continue
		}
		seen[lookup] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
