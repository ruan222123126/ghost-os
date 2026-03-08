package indexer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const viewManifestFileName = "view_manifest.json"

type GraphViewManifest struct {
	NodeCount       int       `json:"node_count,omitempty"`
	EdgeCount       int       `json:"edge_count,omitempty"`
	SessionCoverage []string  `json:"session_coverage,omitempty"`
	EvidenceCount   int       `json:"evidence_count,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

type DecisionViewManifest struct {
	MemoCount    int       `json:"memo_count,omitempty"`
	RecipeCount  int       `json:"recipe_count,omitempty"`
	SupportCount int       `json:"support_count,omitempty"`
	LastHitAt    time.Time `json:"last_hit_at,omitempty"`
}

type MarkdownViewManifest struct {
	NodeCount            int       `json:"node_count,omitempty"`
	TagCount             int       `json:"tag_count,omitempty"`
	AnchorCount          int       `json:"anchor_count,omitempty"`
	CompressionCoverage  float64   `json:"compression_coverage,omitempty"`
	UpdatedAt            time.Time `json:"updated_at,omitempty"`
}

type VectorViewManifest struct {
	DocumentCount      int               `json:"document_count,omitempty"`
	FreshAt            time.Time         `json:"fresh_at,omitempty"`
	ObjectTypeCount    map[string]int    `json:"object_type_count,omitempty"`
}

type ViewManifest struct {
	Projector        string                `json:"projector,omitempty"`
	Namespace        string                `json:"namespace,omitempty"`
	Workspace        string                `json:"workspace,omitempty"`
	Month            string                `json:"month,omitempty"`
	UpdatedAt        time.Time             `json:"updated_at,omitempty"`
	SessionCoverage  []string              `json:"session_coverage,omitempty"`
	EvidenceCount    int                   `json:"evidence_count,omitempty"`
	Graph            *GraphViewManifest    `json:"graph,omitempty"`
	Decision         *DecisionViewManifest `json:"decision,omitempty"`
	Markdown         *MarkdownViewManifest `json:"markdown,omitempty"`
	Vector           *VectorViewManifest   `json:"vector,omitempty"`
}

type ViewManifestStore interface {
	LoadViewManifest(projector string, bucket Bucket) (ViewManifest, error)
	SaveViewManifest(projector string, bucket Bucket, manifest ViewManifest) error
}

func normalizeViewManifest(manifest ViewManifest) ViewManifest {
	out := manifest
	out.Projector = strings.TrimSpace(out.Projector)
	out.Namespace = strings.TrimSpace(out.Namespace)
	out.Workspace = strings.TrimSpace(out.Workspace)
	out.Month = strings.TrimSpace(out.Month)
	out.UpdatedAt = out.UpdatedAt.UTC()
	out.SessionCoverage = uniqueSessionStrings(out.SessionCoverage)
	if out.EvidenceCount < 0 {
		out.EvidenceCount = 0
	}
	if out.Graph != nil {
		graph := *out.Graph
		graph.SessionCoverage = uniqueSessionStrings(graph.SessionCoverage)
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

func (s *FileCheckpointStore) LoadViewManifest(projector string, bucket Bucket) (ViewManifest, error) {
	if s == nil || s.baseDir == "" {
		return ViewManifest{}, nil
	}
	path := s.viewPath(projector, bucket)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ViewManifest{}, nil
		}
		return ViewManifest{}, fmt.Errorf("read index view manifest: %w", err)
	}
	var manifest ViewManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ViewManifest{}, fmt.Errorf("decode index view manifest: %w", err)
	}
	return normalizeViewManifest(manifest), nil
}

func (s *FileCheckpointStore) SaveViewManifest(projector string, bucket Bucket, manifest ViewManifest) error {
	if s == nil || s.baseDir == "" {
		return nil
	}
	bucket = normalizeBucket(bucket)
	manifest = normalizeViewManifest(manifest)
	manifest.Projector = firstNonEmpty(strings.TrimSpace(projector), manifest.Projector)
	manifest.Namespace = firstNonEmpty(bucket.Namespace, manifest.Namespace)
	manifest.Workspace = firstNonEmpty(bucket.Workspace, manifest.Workspace)
	manifest.Month = firstNonEmpty(bucket.Month, manifest.Month)
	manifest.UpdatedAt = time.Now().UTC()
	path := s.viewPath(projector, bucket)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode index view manifest: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create index view manifest dir: %w", err)
	}
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write index view manifest temp: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("rename index view manifest temp: %w", err)
	}
	return nil
}

func (s *FileCheckpointStore) viewPath(projector string, bucket Bucket) string {
	bucket = normalizeBucket(bucket)
	workspace := bucket.Workspace
	if workspace == "" {
		workspace = "_"
	}
	return filepath.Join(
		s.baseDir,
		strings.TrimSpace(projector),
		bucket.Namespace,
		workspace,
		bucket.Month,
		viewManifestFileName,
	)
}
