package memory

import "time"

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
