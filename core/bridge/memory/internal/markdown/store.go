package markdown

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	ianchor "ghost-os/bridge/memory/internal/anchor"
	ibucket "ghost-os/bridge/memory/internal/bucket"
	"gopkg.in/yaml.v3"
)

const ProjectionVersion = "claim-projection/v1"

var errInvalidNodeID = errors.New("invalid markdown node id")

type Node struct {
	ID                string           `yaml:"id"`
	Namespace         string           `yaml:"namespace,omitempty"`
	WorkspaceID       string           `yaml:"workspace_id,omitempty"`
	BucketKey         string           `yaml:"bucket_key,omitempty"`
	Importance        float64          `yaml:"importance"`
	CreatedAt         time.Time        `yaml:"created_at"`
	RelatedTo         []string         `yaml:"related_to,omitempty"`
	Tags              []string         `yaml:"tags,omitempty"`
	SessionID         string           `yaml:"session_id,omitempty"`
	EmbeddingID       string           `yaml:"embedding_id,omitempty"`
	Summary           string           `yaml:"summary,omitempty"`
	Anchors           []ianchor.Anchor `yaml:"anchors,omitempty"`
	SourceIDs         []string         `yaml:"source_ids,omitempty"`
	SourceEvidenceIDs []string         `yaml:"source_evidence_ids,omitempty"`
	SourceClaimIDs    []string         `yaml:"source_claim_ids,omitempty"`
	DerivedClaimIDs   []string         `yaml:"derived_claim_ids,omitempty"`
	ProjectionVersion string           `yaml:"projection_version,omitempty"`
	ProjectionPartial bool             `yaml:"projection_partial,omitempty"`
	Confidence        float64          `yaml:"confidence,omitempty"`
	LastSeenAt        time.Time        `yaml:"last_seen_at,omitempty"`
	Content           string           `yaml:"-"`
}

type Store struct {
	baseDir          string
	defaultNamespace string
	defaultWorkspace string
	manifestPath     string
}

type Query struct {
	SessionHints []string
	TimeRange    *TimeRange
}

type TimeRange struct {
	Start time.Time
	End   time.Time
}

func (r *TimeRange) Contains(ts time.Time) bool {
	if r == nil {
		return true
	}
	t := ts.UTC()
	if !r.Start.IsZero() && t.Before(r.Start.UTC()) {
		return false
	}
	if !r.End.IsZero() && t.After(r.End.UTC()) {
		return false
	}
	return true
}

type manifest struct {
	UpdatedAt time.Time       `json:"updated_at,omitempty"`
	Nodes     []manifestEntry `json:"nodes,omitempty"`
}

type manifestEntry struct {
	NodeID      string    `json:"node_id,omitempty"`
	Namespace   string    `json:"namespace,omitempty"`
	WorkspaceID string    `json:"workspace_id,omitempty"`
	BucketKey   string    `json:"bucket_key,omitempty"`
	SessionID   string    `json:"session_id,omitempty"`
	Month       string    `json:"month,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	LastSeenAt  time.Time `json:"last_seen_at,omitempty"`
	TagCount    int       `json:"tag_count,omitempty"`
	AnchorCount int       `json:"anchor_count,omitempty"`
}

func NewStore(baseDir string) *Store {
	return NewStoreWithConfig(baseDir, "", "")
}

func NewStoreWithConfig(baseDir string, namespace string, workspaceID string) *Store {
	trimmedBaseDir := strings.TrimSpace(baseDir)
	return &Store{
		baseDir:          trimmedBaseDir,
		defaultNamespace: normalizeNamespace(namespace),
		defaultWorkspace: strings.TrimSpace(workspaceID),
		manifestPath:     filepath.Join(trimmedBaseDir, "markdown_manifest.json"),
	}
}

func (s *Store) Save(node Node) error {
	node = s.normalizeForStore(node)
	if err := validateNodeID(node.ID); err != nil {
		return err
	}
	if s.baseDir == "" {
		return fmt.Errorf("markdown store base dir is empty")
	}
	if err := os.MkdirAll(s.baseDir, 0o700); err != nil {
		return fmt.Errorf("create markdown store directory: %w", err)
	}
	filePath := filepath.Join(s.baseDir, node.ID+".md")
	content, err := s.marshal(node)
	if err != nil {
		return fmt.Errorf("marshal markdown node: %w", err)
	}
	tmpPath := fmt.Sprintf("%s.tmp-%d", filePath, time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, content, 0o600); err != nil {
		return fmt.Errorf("write markdown temp file: %w", err)
	}
	if err := os.Rename(tmpPath, filePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace markdown file: %w", err)
	}
	return s.updateManifest(node)
}

func (s *Store) Load(id string) (Node, error) {
	if err := validateNodeID(id); err != nil {
		return Node{}, err
	}
	if s.baseDir == "" {
		return Node{}, fmt.Errorf("markdown store base dir is empty")
	}
	data, err := os.ReadFile(filepath.Join(s.baseDir, id+".md"))
	if err != nil {
		return Node{}, fmt.Errorf("read markdown file: %w", err)
	}
	return s.unmarshal(data)
}

func (s *Store) List() ([]string, error) {
	if s.baseDir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read markdown store directory: %w", err)
	}
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		id := strings.TrimSuffix(name, ".md")
		if err := validateNodeID(id); err != nil {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *Store) ListByBucketPlan(selectedBuckets []ibucket.Manifest, query Query) ([]string, error) {
	if len(selectedBuckets) == 0 {
		return s.List()
	}
	manifest, err := s.loadManifest()
	if err != nil {
		return nil, err
	}
	if len(manifest.Nodes) == 0 {
		return s.List()
	}
	selected := make(map[string]ibucket.Manifest, len(selectedBuckets))
	for _, bucket := range selectedBuckets {
		normalized := ibucket.NormalizeManifest(bucket)
		selected[normalized.Key.String()] = normalized
	}
	sessionHints := uniqueStrings(query.SessionHints)
	ids := make([]string, 0, len(manifest.Nodes))
	for _, node := range manifest.Nodes {
		key := ibucket.Key{Namespace: node.Namespace, WorkspaceID: node.WorkspaceID, Month: node.Month}.String()
		bucket, ok := selected[key]
		if !ok {
			continue
		}
		if len(sessionHints) > 0 && !containsString(sessionHints, node.SessionID) && !ibucket.HasSession(bucket, node.SessionID) {
			continue
		}
		if query.TimeRange != nil {
			ts := firstNonZeroTime(node.LastSeenAt, node.CreatedAt)
			if !ts.IsZero() && !query.TimeRange.Contains(ts) {
				continue
			}
		}
		ids = append(ids, node.NodeID)
	}
	sort.Strings(ids)
	return uniqueStrings(ids), nil
}

func validateNodeID(id string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return fmt.Errorf("node id is required")
	}
	if trimmed == "." || trimmed == ".." {
		return errInvalidNodeID
	}
	for _, ch := range trimmed {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch >= '0' && ch <= '9':
		case ch == '-' || ch == '_':
		default:
			return errInvalidNodeID
		}
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return errInvalidNodeID
	}
	return nil
}

func NormalizeNode(node Node) Node {
	out := node
	out.ID = strings.TrimSpace(out.ID)
	out.Namespace = normalizeNamespace(out.Namespace)
	out.WorkspaceID = strings.TrimSpace(out.WorkspaceID)
	out.BucketKey = strings.TrimSpace(out.BucketKey)
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.EmbeddingID = strings.TrimSpace(out.EmbeddingID)
	out.Content = strings.TrimSpace(out.Content)
	out.Summary = strings.TrimSpace(out.Summary)
	out.Importance = clamp01(out.Importance)
	out.Confidence = clamp01(out.Confidence)
	if out.CreatedAt.IsZero() {
		out.CreatedAt = time.Now().UTC()
	} else {
		out.CreatedAt = out.CreatedAt.UTC()
	}
	if !out.LastSeenAt.IsZero() {
		out.LastSeenAt = out.LastSeenAt.UTC()
	}
	if out.BucketKey == "" {
		out.BucketKey = ibucket.Key{
			Namespace:   out.Namespace,
			WorkspaceID: out.WorkspaceID,
			Month:       ibucket.MonthFromTime(firstNonZeroTime(out.LastSeenAt, out.CreatedAt)),
		}.String()
	}
	out.RelatedTo = uniqueStrings(out.RelatedTo)
	out.Tags = uniqueStrings(out.Tags)
	out.SourceIDs = uniqueStrings(out.SourceIDs)
	out.SourceEvidenceIDs = uniqueStrings(out.SourceEvidenceIDs)
	out.SourceClaimIDs = uniqueStrings(out.SourceClaimIDs)
	out.DerivedClaimIDs = uniqueStrings(out.DerivedClaimIDs)
	out.ProjectionVersion = strings.TrimSpace(out.ProjectionVersion)
	out.Anchors = ianchor.NormalizeAll(out.Anchors)
	if len(out.SourceIDs) == 0 && len(out.RelatedTo) > 0 {
		out.SourceIDs = append([]string(nil), out.RelatedTo...)
	}
	if len(out.RelatedTo) == 0 && len(out.SourceIDs) > 0 {
		out.RelatedTo = append([]string(nil), out.SourceIDs...)
	}
	if out.ProjectionVersion == "" && (len(out.SourceClaimIDs) > 0 || len(out.SourceEvidenceIDs) > 0 || len(out.DerivedClaimIDs) > 0) {
		out.ProjectionVersion = ProjectionVersion
	}
	return out
}

func (s *Store) normalizeForStore(node Node) Node {
	out := NormalizeNode(node)
	out.Namespace = firstNonEmpty(out.Namespace, s.defaultNamespace)
	out.WorkspaceID = firstNonEmpty(out.WorkspaceID, s.defaultWorkspace)
	if out.BucketKey == "" {
		out.BucketKey = ibucket.Key{
			Namespace:   out.Namespace,
			WorkspaceID: out.WorkspaceID,
			Month:       ibucket.MonthFromTime(firstNonZeroTime(out.LastSeenAt, out.CreatedAt)),
		}.String()
	}
	return NormalizeNode(out)
}

func (s *Store) marshal(node Node) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("---\n")
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(node); err != nil {
		return nil, err
	}
	encoder.Close()
	buf.WriteString("---\n\n")
	buf.WriteString(strings.TrimSpace(node.Content))
	buf.WriteString("\n")
	return buf.Bytes(), nil
}

func (s *Store) unmarshal(data []byte) (Node, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	if !scanner.Scan() || scanner.Text() != "---" {
		return Node{}, fmt.Errorf("invalid markdown format: missing frontmatter")
	}
	var yamlBuf bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			break
		}
		yamlBuf.WriteString(line)
		yamlBuf.WriteString("\n")
	}
	var node Node
	if err := yaml.Unmarshal(yamlBuf.Bytes(), &node); err != nil {
		return Node{}, fmt.Errorf("unmarshal yaml frontmatter: %w", err)
	}
	var contentBuf bytes.Buffer
	for scanner.Scan() {
		contentBuf.WriteString(scanner.Text())
		contentBuf.WriteString("\n")
	}
	node.Content = strings.TrimSpace(contentBuf.String())
	return NormalizeNode(node), nil
}

func (s *Store) updateManifest(node Node) error {
	if s == nil || s.manifestPath == "" {
		return nil
	}
	manifest, err := s.loadManifest()
	if err != nil {
		return err
	}
	node = s.normalizeForStore(node)
	entry := manifestEntry{
		NodeID:      node.ID,
		Namespace:   node.Namespace,
		WorkspaceID: node.WorkspaceID,
		BucketKey:   node.BucketKey,
		SessionID:   node.SessionID,
		Month:       ibucket.MonthFromTime(firstNonZeroTime(node.LastSeenAt, node.CreatedAt)),
		CreatedAt:   node.CreatedAt,
		LastSeenAt:  node.LastSeenAt,
		TagCount:    len(node.Tags),
		AnchorCount: len(node.Anchors),
	}
	replaced := false
	for index := range manifest.Nodes {
		if manifest.Nodes[index].NodeID != entry.NodeID {
			continue
		}
		manifest.Nodes[index] = entry
		replaced = true
		break
	}
	if !replaced {
		manifest.Nodes = append(manifest.Nodes, entry)
	}
	manifest.UpdatedAt = time.Now().UTC()
	sort.SliceStable(manifest.Nodes, func(i, j int) bool {
		return manifest.Nodes[i].NodeID < manifest.Nodes[j].NodeID
	})
	return writeJSONAtomic(s.manifestPath, manifest)
}

func (s *Store) loadManifest() (manifest, error) {
	if s == nil || s.manifestPath == "" {
		return manifest{}, nil
	}
	data, err := os.ReadFile(s.manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return manifest{}, nil
		}
		return manifest{}, fmt.Errorf("read markdown manifest: %w", err)
	}
	var out manifest
	if err := yaml.Unmarshal(data, &out); err == nil {
		return out, nil
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return manifest{}, fmt.Errorf("decode markdown manifest: %w", err)
	}
	return out, nil
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	tempPath := fmt.Sprintf("%s.tmp-%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tempPath, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write temp json: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace json file: %w", err)
	}
	return nil
}

func normalizeNamespace(namespace string) string {
	trimmed := strings.TrimSpace(namespace)
	if trimmed == "" {
		return "default"
	}
	return trimmed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value.UTC()
		}
	}
	return time.Time{}
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func containsString(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func clamp01(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}
