package memory

import (
	"bufio"
	"bytes"
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
	baseDir string
}

// NewMarkdownStore 创建 Markdown 存储实例。
func NewMarkdownStore(baseDir string) *MarkdownStore {
	return &MarkdownStore{
		baseDir: resolveMemoryPath(baseDir),
	}
}

// Save 保存记忆节点为 Markdown 文件。
func (s *MarkdownStore) Save(node MarkdownNode) error {
	node = normalizeMarkdownNode(node)
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
