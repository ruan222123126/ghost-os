package memory

import (
	"strings"
	"time"
)

const (
	GraphNodeTypePerson     = "person"
	GraphNodeTypeProject    = "project"
	GraphNodeTypeOrg        = "org"
	GraphNodeTypeRepo       = "repo"
	GraphNodeTypeTopic      = "topic"
	GraphNodeTypeTech       = "tech"
	GraphNodeTypePreference = "preference"

	GraphPredicateOwnerOf   = "owner_of"
	GraphPredicateMemberOf  = "member_of"
	GraphPredicateUses      = "uses"
	GraphPredicatePrefers   = "prefers"
	GraphPredicateAvoids    = "avoids"
	GraphPredicateDependsOn = "depends_on"
	GraphPredicateRelatedTo = "related_to"
	GraphPredicateBlockedBy = "blocked_by"
	GraphPredicateWorksOn   = "works_on"

	GraphStatusActive     = "active"
	GraphStatusConflicted = "conflicted"
	GraphStatusSuperseded = "superseded"

	defaultGraphMaxHops       = 1
	defaultGraphMaxHits       = 6
	defaultGraphMinConfidence = 0.72
	defaultGraphNamespace     = "default"
	defaultGraphEvidenceLimit = 6
	defaultGraphAliasPathName = "aliases.json"
	defaultGraphNodesPathName = "nodes.json"
	defaultGraphEdgesPathName = "edges.json"
)

var allowedGraphPredicates = map[string]struct{}{
	GraphPredicateOwnerOf:   {},
	GraphPredicateMemberOf:  {},
	GraphPredicateUses:      {},
	GraphPredicatePrefers:   {},
	GraphPredicateAvoids:    {},
	GraphPredicateDependsOn: {},
	GraphPredicateRelatedTo: {},
	GraphPredicateBlockedBy: {},
	GraphPredicateWorksOn:   {},
}

var singleValueGraphPredicates = map[string]struct{}{
	GraphPredicateOwnerOf: {},
}

// GraphNode 表示聚合后的实体节点。
type GraphNode struct {
	ID            string         `json:"id"`
	Namespace     string         `json:"namespace"`
	CanonicalName string         `json:"canonical_name"`
	Type          string         `json:"type"`
	Aliases       []string       `json:"aliases,omitempty"`
	Summary       string         `json:"summary,omitempty"`
	AnchorKeys    []string       `json:"anchor_keys,omitempty"`
	Confidence    float64        `json:"confidence,omitempty"`
	FirstSeenAt   time.Time      `json:"first_seen_at,omitempty"`
	LastSeenAt    time.Time      `json:"last_seen_at,omitempty"`
	MentionCount  int            `json:"mention_count,omitempty"`
	SourceIDs     []string       `json:"source_ids,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// GraphEdge 表示带时序、证据和状态的关系边。
type GraphEdge struct {
	ID          string          `json:"id"`
	Namespace   string          `json:"namespace"`
	SubjectID   string          `json:"subject_id"`
	Predicate   string          `json:"predicate"`
	ObjectID    string          `json:"object_id"`
	Confidence  float64         `json:"confidence,omitempty"`
	Weight      float64         `json:"weight,omitempty"`
	Status      string          `json:"status,omitempty"`
	FirstSeenAt time.Time       `json:"first_seen_at,omitempty"`
	LastSeenAt  time.Time       `json:"last_seen_at,omitempty"`
	SessionIDs  []string        `json:"session_ids,omitempty"`
	SourceIDs   []string        `json:"source_ids,omitempty"`
	Evidence    []GraphEvidence `json:"evidence,omitempty"`
	ExpiresAt   time.Time       `json:"expires_at,omitempty"`
}

// GraphEvidence 保存关系来源，便于调试和追溯。
type GraphEvidence struct {
	SessionID string    `json:"session_id,omitempty"`
	SourceID  string    `json:"source_id,omitempty"`
	Snippet   string    `json:"snippet,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
	TraceID   string    `json:"trace_id,omitempty"`
}

// GraphFact 是 extractor 输出的中间结构。
type GraphFact struct {
	Namespace   string         `json:"namespace,omitempty"`
	Subject     string         `json:"subject"`
	SubjectType string         `json:"subject_type,omitempty"`
	Predicate   string         `json:"predicate"`
	Object      string         `json:"object"`
	ObjectType  string         `json:"object_type,omitempty"`
	Aliases     []string       `json:"aliases,omitempty"`
	Confidence  float64        `json:"confidence,omitempty"`
	Snippet     string         `json:"snippet,omitempty"`
	Timestamp   time.Time      `json:"timestamp,omitempty"`
	AnchorKeys  []string       `json:"anchor_keys,omitempty"`
	SessionID   string         `json:"session_id,omitempty"`
	SourceID    string         `json:"source_id,omitempty"`
	TraceID     string         `json:"trace_id,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// GraphHit 是 graph 检索的调试视图。
type GraphHit struct {
	Namespace     string          `json:"namespace,omitempty"`
	NodeID        string          `json:"node_id,omitempty"`
	EdgeID        string          `json:"edge_id,omitempty"`
	Subject       string          `json:"subject"`
	Predicate     string          `json:"predicate"`
	Object        string          `json:"object"`
	Status        string          `json:"status,omitempty"`
	Hop           int             `json:"hop,omitempty"`
	Score         float64         `json:"score,omitempty"`
	Content       string          `json:"content,omitempty"`
	SourceIDs     []string        `json:"source_ids,omitempty"`
	Evidence      []GraphEvidence `json:"evidence,omitempty"`
	EvidenceCount int             `json:"evidence_count,omitempty"`
}

// GraphStats 表示图谱当前快照统计。
type GraphStats struct {
	Namespace       string    `json:"namespace,omitempty"`
	NodeCount       int       `json:"node_count"`
	EdgeCount       int       `json:"edge_count"`
	ActiveEdgeCount int       `json:"active_edge_count"`
	AliasCount      int       `json:"alias_count"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

// GraphApplyStats 表示一次事实写入结果。
type GraphApplyStats struct {
	FactsApplied  int `json:"facts_applied"`
	NodesUpserted int `json:"nodes_upserted"`
	EdgesUpserted int `json:"edges_upserted"`
}

// GraphRebuildOptions 控制图谱回填/重建窗口。
type GraphRebuildOptions struct {
	Namespace   string     `json:"namespace,omitempty"`
	TimeRange   *TimeRange `json:"time_range,omitempty"`
	MaxSessions int        `json:"max_sessions,omitempty"`
	DryRun      bool       `json:"dry_run,omitempty"`
}

// GraphRebuildStats 描述重建扫描结果。
type GraphRebuildStats struct {
	Namespace       string `json:"namespace,omitempty"`
	SessionsScanned int    `json:"sessions_scanned"`
	MarkdownScanned int    `json:"markdown_scanned"`
	FactsExtracted  int    `json:"facts_extracted"`
	NodesUpserted   int    `json:"nodes_upserted"`
	EdgesUpserted   int    `json:"edges_upserted"`
}

// MemoryQueryResult 允许在不破坏旧接口的情况下带回 graph/decision 命中结果。
type MemoryQueryResult struct {
	Entries      []MemoryEntry `json:"entries"`
	GraphHits    []GraphHit    `json:"graph_hits,omitempty"`
	DecisionHits []DecisionHit `json:"decision_hits,omitempty"`
}

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
