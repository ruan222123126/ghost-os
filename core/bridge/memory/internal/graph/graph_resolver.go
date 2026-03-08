package graph

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var graphMultiSpacePattern = regexp.MustCompile(`\s+`)

var graphWeakReferenceByType = map[string]string{
	"这个项目": GraphNodeTypeProject,
	"那个项目": GraphNodeTypeProject,
	"该项目":  GraphNodeTypeProject,
	"这个仓库": GraphNodeTypeRepo,
	"那个仓库": GraphNodeTypeRepo,
	"这个人":  GraphNodeTypePerson,
	"那个人":  GraphNodeTypePerson,
	"他":    GraphNodeTypePerson,
	"她":    GraphNodeTypePerson,
	"它":    GraphNodeTypeTopic,
}

// RecentEntityContext 仅保留近场弱指代所需的最小映射。
type RecentEntityContext struct {
	LatestByType map[string]string
}

// GraphResolver 负责实体归一化与 alias 归并。
type GraphResolver struct{}

func NewGraphResolver() *GraphResolver {
	return &GraphResolver{}
}

func cleanGraphDisplayName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"（", "(",
		"）", ")",
		"【", "[",
		"】", "]",
		"，", " ",
		"。", " ",
		"：", ":",
		"；", " ",
		"、", " ",
		"\n", " ",
		"\t", " ",
	)
	trimmed = replacer.Replace(trimmed)
	trimmed = graphMultiSpacePattern.ReplaceAllString(trimmed, " ")
	trimmed = strings.Trim(trimmed, " `\"'.,!?;:[]{}()")
	return strings.TrimSpace(trimmed)
}

func normalizeGraphLookup(raw string) string {
	trimmed := cleanGraphDisplayName(raw)
	if trimmed == "" {
		return ""
	}
	if containsHan(trimmed) {
		trimmed = strings.ReplaceAll(trimmed, " ", "")
	}
	trimmed = strings.ToLower(trimmed)
	trimmed = strings.NewReplacer(
		"（", "(",
		"）", ")",
		"，", " ",
		"。", " ",
		"：", ":",
		"；", " ",
	).Replace(trimmed)
	trimmed = graphMultiSpacePattern.ReplaceAllString(trimmed, " ")
	trimmed = strings.Trim(trimmed, " `\"'.,!?;:[]{}()")
	for _, prefix := range []string{"the ", "this ", "that ", "a ", "an ", "这个", "那个", "该"} {
		trimmed = strings.TrimPrefix(trimmed, prefix)
	}
	return strings.TrimSpace(trimmed)
}

func graphLookupVariants(raw string) []string {
	primary := normalizeGraphLookup(raw)
	if primary == "" {
		return nil
	}
	seen := map[string]struct{}{primary: {}}
	out := []string{primary}
	if containsHan(primary) {
		variant := strings.TrimPrefix(primary, "老")
		for _, suffix := range []string{"工", "总", "老师"} {
			variant = strings.TrimSuffix(variant, suffix)
		}
		variant = strings.TrimSpace(variant)
		if len([]rune(variant)) >= 1 && variant != primary {
			if _, ok := seen[variant]; !ok {
				seen[variant] = struct{}{}
				out = append(out, variant)
			}
		}
	}
	return out
}

func containsHan(text string) bool {
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func graphNodeLookupKey(namespace, value string) string {
	return normalizeGraphNamespace(namespace) + "|" + normalizeGraphLookup(value)
}

func graphEdgeLookupKey(namespace, subjectID, predicate, objectID string) string {
	return strings.Join([]string{
		normalizeGraphNamespace(namespace),
		strings.TrimSpace(subjectID),
		normalizeGraphPredicate(predicate),
		strings.TrimSpace(objectID),
	}, "|")
}

func (r *GraphResolver) resolveWeakReference(term string, recent RecentEntityContext) string {
	if len(recent.LatestByType) == 0 {
		return ""
	}
	entityType, ok := graphWeakReferenceByType[strings.TrimSpace(term)]
	if !ok {
		return ""
	}
	return strings.TrimSpace(recent.LatestByType[entityType])
}

func (r *GraphResolver) inferNodeType(name string, hintedType string) string {
	if normalized := normalizeGraphNodeType(hintedType); normalized != GraphNodeTypeTopic || strings.TrimSpace(hintedType) != "" {
		return normalized
	}
	lower := strings.ToLower(cleanGraphDisplayName(name))
	switch {
	case strings.HasPrefix(lower, "repo "), strings.Contains(lower, "github"), strings.Contains(lower, "/"):
		return GraphNodeTypeRepo
	case strings.HasPrefix(lower, "project "), strings.Contains(name, "项目"):
		return GraphNodeTypeProject
	case strings.Contains(lower, "python"), strings.Contains(lower, "go"), strings.Contains(lower, "rust"), strings.Contains(lower, "java"), strings.Contains(lower, "javascript"):
		return GraphNodeTypeTech
	case strings.Contains(name, "公司"), strings.Contains(name, "团队"), strings.Contains(lower, "org"):
		return GraphNodeTypeOrg
	case strings.HasPrefix(lower, "self"), strings.HasPrefix(lower, "user"), lower == "我":
		return GraphNodeTypePerson
	default:
		return GraphNodeTypeTopic
	}
}

func (r *GraphResolver) resolveNode(store *GraphStore, namespace string, name string, aliases []string, hintedType string, anchorKeys []string, sourceID string, seenAt time.Time) GraphNode {
	if store == nil {
		return GraphNode{}
	}
	displayName := cleanGraphDisplayName(name)
	lookupName := normalizeGraphLookup(displayName)
	if displayName == "" || lookupName == "" {
		return GraphNode{}
	}
	if node, ok := store.lookupNodeLocked(namespace, displayName); ok {
		updated := mergeResolvedNode(node, displayName, aliases, hintedType, anchorKeys, sourceID, seenAt)
		store.saveNodeLocked(updated)
		return updated
	}
	createdAt := seenAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	node := normalizeGraphNode(GraphNode{
		ID:            fmt.Sprintf("graph-node-%d", time.Now().UTC().UnixNano()),
		Namespace:     namespace,
		CanonicalName: displayName,
		Type:          r.inferNodeType(displayName, hintedType),
		Aliases:       aliases,
		AnchorKeys:    anchorKeys,
		FirstSeenAt:   createdAt,
		LastSeenAt:    createdAt,
		MentionCount:  1,
		SourceIDs:     uniqueStrings([]string{sourceID}),
		Metadata:      map[string]any{},
	})
	store.saveNodeLocked(node)
	return node
}

func mergeResolvedNode(node GraphNode, displayName string, aliases []string, hintedType string, anchorKeys []string, sourceID string, seenAt time.Time) GraphNode {
	out := normalizeGraphNode(node)
	if out.CanonicalName == "" {
		out.CanonicalName = displayName
	}
	out.Aliases = normalizeGraphAliases(append(out.Aliases, aliases...))
	out.AnchorKeys = normalizeGraphAliases(append(out.AnchorKeys, anchorKeys...))
	if normalizedType := normalizeGraphNodeType(hintedType); normalizedType != GraphNodeTypeTopic || strings.TrimSpace(hintedType) != "" {
		out.Type = normalizedType
	}
	if sourceID != "" {
		out.SourceIDs = uniqueStrings(append(out.SourceIDs, sourceID))
	}
	out.MentionCount++
	if seenAt.IsZero() {
		seenAt = time.Now().UTC()
	}
	if out.FirstSeenAt.IsZero() || seenAt.Before(out.FirstSeenAt) {
		out.FirstSeenAt = seenAt.UTC()
	}
	if seenAt.After(out.LastSeenAt) {
		out.LastSeenAt = seenAt.UTC()
	}
	return normalizeGraphNode(out)
}
