package memory

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// MarkdownNode 表示一个 Markdown 记忆节点（带 YAML Frontmatter）。
type MarkdownNode struct {
	ID          string         `yaml:"id"`
	Namespace   string         `yaml:"namespace,omitempty"`
	WorkspaceID string         `yaml:"workspace_id,omitempty"`
	BucketKey   string         `yaml:"bucket_key,omitempty"`
	Importance  float64        `yaml:"importance"`
	CreatedAt   time.Time      `yaml:"created_at"`
	RelatedTo   []string       `yaml:"related_to,omitempty"`
	Tags        []string       `yaml:"tags,omitempty"`
	SessionID   string         `yaml:"session_id,omitempty"`
	EmbeddingID string         `yaml:"embedding_id,omitempty"`
	Summary     string         `yaml:"summary,omitempty"`
	Anchors     []MemoryAnchor `yaml:"anchors,omitempty"`
	SourceIDs   []string       `yaml:"source_ids,omitempty"`
	Confidence  float64        `yaml:"confidence,omitempty"`
	LastSeenAt  time.Time      `yaml:"last_seen_at,omitempty"`
	Content     string         `yaml:"-"` // Markdown 正文
}

// MarkdownStore 提供基于 Markdown + YAML Frontmatter 的持久化。
type MarkdownStore struct {
	baseDir          string
	defaultNamespace string
	defaultWorkspace string
	manifestPath     string
}

type markdownManifest struct {
	UpdatedAt time.Time               `json:"updated_at,omitempty"`
	Nodes     []markdownManifestEntry `json:"nodes,omitempty"`
}

type markdownManifestEntry struct {
	NodeID       string    `json:"node_id,omitempty"`
	Namespace    string    `json:"namespace,omitempty"`
	WorkspaceID  string    `json:"workspace_id,omitempty"`
	BucketKey    string    `json:"bucket_key,omitempty"`
	SessionID    string    `json:"session_id,omitempty"`
	Month        string    `json:"month,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	LastSeenAt   time.Time `json:"last_seen_at,omitempty"`
	TagCount     int       `json:"tag_count,omitempty"`
	AnchorCount  int       `json:"anchor_count,omitempty"`
}

// NewMarkdownStore 创建 Markdown 存储实例。
func NewMarkdownStore(baseDir string) *MarkdownStore {
	return NewMarkdownStoreWithConfig(baseDir, "", "")
}

func NewMarkdownStoreWithConfig(baseDir string, namespace string, workspaceID string) *MarkdownStore {
	resolved := resolveMemoryPath(baseDir)
	return &MarkdownStore{
		baseDir:          resolved,
		defaultNamespace: normalizeLedgerNamespace(namespace),
		defaultWorkspace: strings.TrimSpace(workspaceID),
		manifestPath:     filepath.Join(resolved, "markdown_manifest.json"),
	}
}

// Save 保存记忆节点为 Markdown 文件。
func (s *MarkdownStore) Save(node MarkdownNode) error {
	node = s.normalizeForStore(node)
	if node.ID == "" {
		return fmt.Errorf("node id is required")
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
	if err := s.updateManifest(node); err != nil {
		return err
	}
	return nil
}

// Load 从 Markdown 文件加载记忆节点。
func (s *MarkdownStore) Load(id string) (MarkdownNode, error) {
	if id == "" {
		return MarkdownNode{}, fmt.Errorf("node id is required")
	}
	if s.baseDir == "" {
		return MarkdownNode{}, fmt.Errorf("markdown store base dir is empty")
	}

	filePath := filepath.Join(s.baseDir, id+".md")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return MarkdownNode{}, fmt.Errorf("read markdown file: %w", err)
	}

	return s.unmarshal(data)
}

// List 列出所有记忆节点 ID。
func (s *MarkdownStore) List() ([]string, error) {
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
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *MarkdownStore) ListByBucketPlan(plan *BucketPlan, query MemoryQuery) ([]string, error) {
	if plan == nil || len(plan.SelectedBuckets) == 0 {
		return s.List()
	}
	manifest, err := s.loadManifest()
	if err != nil {
		return nil, err
	}
	if len(manifest.Nodes) == 0 {
		return s.List()
	}
	selected := make(map[string]BucketManifest, len(plan.SelectedBuckets))
	for _, bucket := range plan.SelectedBuckets {
		selected[bucket.Bucket.Key.String()] = bucket.Bucket
	}
	sessionHints := effectiveBucketSessionHints(query, SessionScope{})
	ids := make([]string, 0, len(manifest.Nodes))
	for _, node := range manifest.Nodes {
		key := normalizeBucketKey(BucketKey{Namespace: node.Namespace, WorkspaceID: node.WorkspaceID, Month: node.Month}).String()
		bucket, ok := selected[key]
		if !ok {
			continue
		}
		if len(sessionHints) > 0 && !containsString(sessionHints, node.SessionID) && !bucketManifestHasSession(bucket, node.SessionID) {
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

// marshal 将节点序列化为 Markdown + YAML Frontmatter。
func (s *MarkdownStore) marshal(node MarkdownNode) ([]byte, error) {
	var buf bytes.Buffer

	// 写入 YAML Frontmatter
	buf.WriteString("---\n")
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(node); err != nil {
		return nil, err
	}
	encoder.Close()
	buf.WriteString("---\n\n")

	// 写入 Markdown 正文
	buf.WriteString(strings.TrimSpace(node.Content))
	buf.WriteString("\n")

	return buf.Bytes(), nil
}

// unmarshal 从 Markdown + YAML Frontmatter 解析节点。
func (s *MarkdownStore) unmarshal(data []byte) (MarkdownNode, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	// 检查第一行是否为 "---"
	if !scanner.Scan() || scanner.Text() != "---" {
		return MarkdownNode{}, fmt.Errorf("invalid markdown format: missing frontmatter")
	}

	// 读取 YAML Frontmatter
	var yamlBuf bytes.Buffer
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			break
		}
		yamlBuf.WriteString(line)
		yamlBuf.WriteString("\n")
	}

	var node MarkdownNode
	if err := yaml.Unmarshal(yamlBuf.Bytes(), &node); err != nil {
		return MarkdownNode{}, fmt.Errorf("unmarshal yaml frontmatter: %w", err)
	}

	// 读取 Markdown 正文
	var contentBuf bytes.Buffer
	for scanner.Scan() {
		contentBuf.WriteString(scanner.Text())
		contentBuf.WriteString("\n")
	}
	node.Content = strings.TrimSpace(contentBuf.String())
	return normalizeMarkdownNode(node), nil
}

func normalizeMarkdownNode(node MarkdownNode) MarkdownNode {
	out := node
	out.ID = strings.TrimSpace(out.ID)
	out.Namespace = normalizeLedgerNamespace(out.Namespace)
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
		out.BucketKey = normalizeBucketKey(BucketKey{Namespace: out.Namespace, WorkspaceID: out.WorkspaceID, Month: bucketMonthFromTime(firstNonZeroTime(out.LastSeenAt, out.CreatedAt))}).String()
	}
	out.RelatedTo = uniqueStrings(out.RelatedTo)
	out.Tags = uniqueStrings(out.Tags)
	out.SourceIDs = uniqueStrings(out.SourceIDs)
	out.Anchors = normalizeAnchors(out.Anchors)
	if len(out.SourceIDs) == 0 && len(out.RelatedTo) > 0 {
		out.SourceIDs = append([]string(nil), out.RelatedTo...)
	}
	if len(out.RelatedTo) == 0 && len(out.SourceIDs) > 0 {
		out.RelatedTo = append([]string(nil), out.SourceIDs...)
	}
	return out
}

func (s *MarkdownStore) normalizeForStore(node MarkdownNode) MarkdownNode {
	out := normalizeMarkdownNode(node)
	out.Namespace = firstNonEmpty(out.Namespace, s.defaultNamespace)
	out.WorkspaceID = firstNonEmpty(out.WorkspaceID, s.defaultWorkspace)
	if out.BucketKey == "" {
		out.BucketKey = normalizeBucketKey(BucketKey{Namespace: out.Namespace, WorkspaceID: out.WorkspaceID, Month: bucketMonthFromTime(firstNonZeroTime(out.LastSeenAt, out.CreatedAt))}).String()
	}
	return normalizeMarkdownNode(out)
}

func (s *MarkdownStore) updateManifest(node MarkdownNode) error {
	if s == nil || s.manifestPath == "" {
		return nil
	}
	manifest, err := s.loadManifest()
	if err != nil {
		return err
	}
	node = s.normalizeForStore(node)
	entry := markdownManifestEntry{
		NodeID:      node.ID,
		Namespace:   node.Namespace,
		WorkspaceID: node.WorkspaceID,
		BucketKey:   node.BucketKey,
		SessionID:   node.SessionID,
		Month:       bucketMonthFromTime(firstNonZeroTime(node.LastSeenAt, node.CreatedAt)),
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

func (s *MarkdownStore) loadManifest() (markdownManifest, error) {
	if s == nil || s.manifestPath == "" {
		return markdownManifest{}, nil
	}
	data, err := os.ReadFile(s.manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return markdownManifest{}, nil
		}
		return markdownManifest{}, fmt.Errorf("read markdown manifest: %w", err)
	}
	var manifest markdownManifest
	if err := yaml.Unmarshal(data, &manifest); err == nil {
		return manifest, nil
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return markdownManifest{}, fmt.Errorf("decode markdown manifest: %w", err)
	}
	return manifest, nil
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value.UTC()
		}
	}
	return time.Time{}
}
