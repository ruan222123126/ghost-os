package memory

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ghost-os/bridge/memory/internal/indexer"
)

type BucketKey struct {
	Namespace   string `json:"namespace,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Month       string `json:"month,omitempty"`
}

func (k BucketKey) String() string {
	normalized := normalizeBucketKey(k)
	return strings.Join([]string{normalized.Namespace, firstNonEmpty(normalized.WorkspaceID, "_"), normalized.Month}, "/")
}

type BucketSessionSummary struct {
	SessionID      string    `json:"session_id,omitempty"`
	MinOccurredAt  time.Time `json:"min_occurred_at,omitempty"`
	MaxOccurredAt  time.Time `json:"max_occurred_at,omitempty"`
	FirstOffset    int64     `json:"first_offset,omitempty"`
	LastOffset     int64     `json:"last_offset,omitempty"`
	Count          int       `json:"count,omitempty"`
	TraceCount     int       `json:"trace_count,omitempty"`
	UserTurns      int       `json:"user_turns,omitempty"`
	AssistantTurns int       `json:"assistant_turns,omitempty"`
	Kinds          []string  `json:"kinds,omitempty"`
	TopTerms       []string  `json:"top_terms,omitempty"`
}

type BucketViewManifest struct {
	Projector       string                   `json:"projector,omitempty"`
	UpdatedAt       time.Time                `json:"updated_at,omitempty"`
	SessionCoverage []string                 `json:"session_coverage,omitempty"`
	EvidenceCount   int                      `json:"evidence_count,omitempty"`
	Graph           *indexer.GraphViewManifest `json:"graph,omitempty"`
	Decision        *indexer.DecisionViewManifest `json:"decision,omitempty"`
	Markdown        *indexer.MarkdownViewManifest `json:"markdown,omitempty"`
	Vector          *indexer.VectorViewManifest `json:"vector,omitempty"`
}

type BucketManifest struct {
	Key             BucketKey                        `json:"key,omitempty"`
	RootDir         string                           `json:"root_dir,omitempty"`
	SegmentFile     string                           `json:"segment_file,omitempty"`
	MinOccurredAt   time.Time                        `json:"min_occurred_at,omitempty"`
	MaxOccurredAt   time.Time                        `json:"max_occurred_at,omitempty"`
	FirstOffset     int64                            `json:"first_offset,omitempty"`
	LastOffset      int64                            `json:"last_offset,omitempty"`
	Count           int                              `json:"count,omitempty"`
	SessionIDs      []string                         `json:"session_ids,omitempty"`
	Sessions        []BucketSessionSummary           `json:"sessions,omitempty"`
	TopTerms        []string                         `json:"top_terms,omitempty"`
	Kinds           []string                         `json:"kinds,omitempty"`
	TraceCount      int                              `json:"trace_count,omitempty"`
	UserTurns       int                              `json:"user_turns,omitempty"`
	AssistantTurns  int                              `json:"assistant_turns,omitempty"`
	UpdatedAt       time.Time                        `json:"updated_at,omitempty"`
	Views           map[string]BucketViewManifest    `json:"views,omitempty"`
}

type BucketCandidate struct {
	Bucket          BucketManifest `json:"bucket,omitempty"`
	Score           float64        `json:"score,omitempty"`
	Reasons         []string       `json:"reasons,omitempty"`
	MatchedSessions []string       `json:"matched_sessions,omitempty"`
	MatchedTerms    []string       `json:"matched_terms,omitempty"`
	EstimatedCost   float64        `json:"estimated_cost,omitempty"`
}

type BucketPlan struct {
	Mode             string            `json:"mode,omitempty"`
	CandidateBuckets []BucketCandidate `json:"candidate_buckets,omitempty"`
	SelectedBuckets  []BucketCandidate `json:"selected_buckets,omitempty"`
	DroppedBuckets   []BucketCandidate `json:"dropped_buckets,omitempty"`
	Reason           []string          `json:"reason,omitempty"`
	EstimatedCost    float64           `json:"estimated_cost,omitempty"`
}

type BucketShadowReport struct {
	Mode             string    `json:"mode,omitempty"`
	LegacyEntryCount int       `json:"legacy_entry_count,omitempty"`
	BucketEntryCount int       `json:"bucket_entry_count,omitempty"`
	TopOverlap       float64   `json:"top_overlap,omitempty"`
	SelectedBuckets  []string  `json:"selected_buckets,omitempty"`
	ShadowOnly       []string  `json:"shadow_only,omitempty"`
	LatencyMs        int64     `json:"latency_ms,omitempty"`
}

func normalizeBucketKey(key BucketKey) BucketKey {
	out := key
	out.Namespace = normalizeLedgerNamespace(out.Namespace)
	out.WorkspaceID = strings.TrimSpace(out.WorkspaceID)
	out.Month = strings.TrimSpace(out.Month)
	return out
}

func bucketMonthFromTime(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format("2006-01")
}

func normalizeBucketSessionSummary(summary BucketSessionSummary) BucketSessionSummary {
	out := summary
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.MinOccurredAt = out.MinOccurredAt.UTC()
	out.MaxOccurredAt = out.MaxOccurredAt.UTC()
	out.Kinds = uniqueStrings(out.Kinds)
	out.TopTerms = uniqueStrings(out.TopTerms)
	return out
}

func normalizeBucketViewManifest(manifest BucketViewManifest) BucketViewManifest {
	out := manifest
	out.Projector = strings.TrimSpace(out.Projector)
	out.UpdatedAt = out.UpdatedAt.UTC()
	out.SessionCoverage = uniqueStrings(out.SessionCoverage)
	if out.Graph != nil {
		graph := *out.Graph
		graph.SessionCoverage = uniqueStrings(graph.SessionCoverage)
		graph.UpdatedAt = graph.UpdatedAt.UTC()
		out.Graph = &graph
	}
	if out.Decision != nil {
		decision := *out.Decision
		decision.LastHitAt = decision.LastHitAt.UTC()
		out.Decision = &decision
	}
	if out.Markdown != nil {
		markdown := *out.Markdown
		markdown.UpdatedAt = markdown.UpdatedAt.UTC()
		out.Markdown = &markdown
	}
	if out.Vector != nil {
		vector := *out.Vector
		vector.FreshAt = vector.FreshAt.UTC()
		out.Vector = &vector
	}
	return out
}

func normalizeBucketManifest(manifest BucketManifest) BucketManifest {
	out := manifest
	out.Key = normalizeBucketKey(out.Key)
	out.RootDir = strings.TrimSpace(out.RootDir)
	out.SegmentFile = strings.TrimSpace(out.SegmentFile)
	out.MinOccurredAt = out.MinOccurredAt.UTC()
	out.MaxOccurredAt = out.MaxOccurredAt.UTC()
	out.SessionIDs = uniqueStrings(out.SessionIDs)
	out.TopTerms = uniqueStrings(out.TopTerms)
	out.Kinds = uniqueStrings(out.Kinds)
	out.UpdatedAt = out.UpdatedAt.UTC()
	if len(out.Sessions) > 0 {
		normalized := make([]BucketSessionSummary, 0, len(out.Sessions))
		for _, session := range out.Sessions {
			normalized = append(normalized, normalizeBucketSessionSummary(session))
		}
		out.Sessions = normalized
	}
	if len(out.Views) > 0 {
		normalized := make(map[string]BucketViewManifest, len(out.Views))
		for layer, view := range out.Views {
			normalized[strings.TrimSpace(layer)] = normalizeBucketViewManifest(view)
		}
		out.Views = normalized
	}
	if out.RootDir != "" && out.Key.Month == "" {
		out.Key.Month = filepath.Base(out.RootDir)
	}
	return out
}

func bucketManifestFromLedger(monthDir string, manifest ledgerManifest) BucketManifest {
	sessions := make([]BucketSessionSummary, 0, len(manifest.Sessions))
	for _, session := range manifest.Sessions {
		sessions = append(sessions, BucketSessionSummary{
			SessionID:      session.SessionID,
			MinOccurredAt:  session.MinOccurredAt,
			MaxOccurredAt:  session.MaxOccurredAt,
			FirstOffset:    session.FirstOffset,
			LastOffset:     session.LastOffset,
			Count:          session.Count,
			TraceCount:     session.TraceCount,
			UserTurns:      session.UserTurns,
			AssistantTurns: session.AssistantTurns,
			Kinds:          append([]string(nil), session.Kinds...),
			TopTerms:       append([]string(nil), session.TopTerms...),
		})
	}
	return normalizeBucketManifest(BucketManifest{
		Key: BucketKey{
			Namespace:   manifest.Namespace,
			WorkspaceID: manifest.WorkspaceID,
			Month:       manifest.Month,
		},
		RootDir:        monthDir,
		SegmentFile:    manifest.SegmentFile,
		MinOccurredAt:  manifest.MinOccurred,
		MaxOccurredAt:  manifest.MaxOccurred,
		FirstOffset:    manifest.FirstOffset,
		LastOffset:     manifest.LastOffset,
		Count:          manifest.Count,
		SessionIDs:     append([]string(nil), manifest.SessionIDs...),
		Sessions:       sessions,
		TopTerms:       append([]string(nil), manifest.TopTerms...),
		Kinds:          append([]string(nil), manifest.Kinds...),
		TraceCount:     manifest.TraceCount,
		UserTurns:      manifest.UserTurns,
		AssistantTurns: manifest.AssistantTurns,
		UpdatedAt:      manifest.UpdatedAt,
		Views:          make(map[string]BucketViewManifest),
	})
}

func sortBucketCandidates(candidates []BucketCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		leftTime := candidates[i].Bucket.MaxOccurredAt
		rightTime := candidates[j].Bucket.MaxOccurredAt
		if !leftTime.Equal(rightTime) {
			return leftTime.After(rightTime)
		}
		return candidates[i].Bucket.Key.String() < candidates[j].Bucket.Key.String()
	})
}
