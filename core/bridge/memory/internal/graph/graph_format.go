package graph

import (
	"fmt"
	"strings"
	"time"
)

func graphDisplayName(node GraphNode) string {
	name := cleanGraphDisplayName(node.CanonicalName)
	if strings.EqualFold(name, "self") {
		return "你"
	}
	if name == "self" {
		return "你"
	}
	return name
}

func formatGraphFact(subject GraphNode, predicate string, object GraphNode) string {
	subjectName := graphDisplayName(subject)
	objectName := graphDisplayName(object)
	switch predicate {
	case GraphPredicateOwnerOf:
		return fmt.Sprintf("%s 的负责人是 %s", objectName, subjectName)
	case GraphPredicateMemberOf:
		return fmt.Sprintf("%s 属于 %s", subjectName, objectName)
	case GraphPredicateUses:
		return fmt.Sprintf("%s 使用 %s", subjectName, objectName)
	case GraphPredicatePrefers:
		return fmt.Sprintf("%s 偏好 %s", subjectName, objectName)
	case GraphPredicateAvoids:
		return fmt.Sprintf("%s 避免 %s", subjectName, objectName)
	case GraphPredicateDependsOn:
		return fmt.Sprintf("%s 依赖 %s", subjectName, objectName)
	case GraphPredicateBlockedBy:
		return fmt.Sprintf("%s 被 %s 阻塞", subjectName, objectName)
	case GraphPredicateWorksOn:
		return fmt.Sprintf("%s 在跟进 %s", subjectName, objectName)
	default:
		return fmt.Sprintf("%s 与 %s 相关", subjectName, objectName)
	}
}

func memoryEntryFromGraphHit(hit GraphHit, edge GraphEdge) MemoryEntry {
	metadata := map[string]any{
		"layer":       "graph",
		"source":      "graph",
		"edge_id":     hit.EdgeID,
		"node_id":     hit.NodeID,
		"predicate":   hit.Predicate,
		"hop":         hit.Hop,
		"graph_score": hit.Score,
		"status":      hit.Status,
		"subject":     hit.Subject,
		"object":      hit.Object,
	}
	return normalizeEntry(MemoryEntry{
		ID:         "graph:" + strings.TrimSpace(hit.EdgeID),
		Content:    strings.TrimSpace(hit.Content),
		Summary:    strings.TrimSpace(hit.Content),
		Type:       MemoryTypeKnowledge,
		Timestamp:  graphEntryTimestamp(edge),
		Importance: clamp01(hit.Score),
		RelatedTo:  append([]string(nil), hit.SourceIDs...),
		Source:     "graph",
		Confidence: edge.Confidence,
		Metadata:   metadata,
	})
}

func graphEntryTimestamp(edge GraphEdge) time.Time {
	if !edge.LastSeenAt.IsZero() {
		return edge.LastSeenAt.UTC()
	}
	if !edge.FirstSeenAt.IsZero() {
		return edge.FirstSeenAt.UTC()
	}
	return time.Now().UTC()
}
