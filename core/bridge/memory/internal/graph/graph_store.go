package graph

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"ghost-os/bridge/memory/internal/pathutil"
)

type graphNodesPayload struct {
	Nodes []GraphNode `json:"nodes"`
}

type graphEdgesPayload struct {
	Edges []GraphEdge `json:"edges"`
}

type graphAliasesPayload struct {
	Aliases map[string]string `json:"aliases"`
}

// GraphStore 维护 graph sidecar 的内存索引与文件快照。
type GraphStore struct {
	baseDir string

	nodeByID         map[string]GraphNode
	edgeByID         map[string]GraphEdge
	nodeByCanonical  map[string]string
	nodeByAlias      map[string]string
	edgeByKey        map[string]string
	edgesBySubject   map[string][]string
	edgesByObject    map[string][]string
	edgesByPredicate map[string][]string
	updatedAt        time.Time
	mu               sync.RWMutex
}

func NewGraphStore(baseDir string) *GraphStore {
	return &GraphStore{
		baseDir:          pathutil.Resolve(baseDir),
		nodeByID:         make(map[string]GraphNode),
		edgeByID:         make(map[string]GraphEdge),
		nodeByCanonical:  make(map[string]string),
		nodeByAlias:      make(map[string]string),
		edgeByKey:        make(map[string]string),
		edgesBySubject:   make(map[string][]string),
		edgesByObject:    make(map[string][]string),
		edgesByPredicate: make(map[string][]string),
	}
}

func (s *GraphStore) enabled() bool {
	return s != nil && strings.TrimSpace(s.baseDir) != ""
}

func (s *GraphStore) Load() error {
	if !s.enabled() {
		return nil
	}

	nodesPath := filepath.Join(s.baseDir, defaultGraphNodesPathName)
	edgesPath := filepath.Join(s.baseDir, defaultGraphEdgesPathName)
	aliasesPath := filepath.Join(s.baseDir, defaultGraphAliasPathName)

	nodes, err := readGraphNodes(nodesPath)
	if err != nil {
		return err
	}
	edges, err := readGraphEdges(edgesPath)
	if err != nil {
		return err
	}
	aliases, err := readGraphAliases(aliasesPath)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodeByID = make(map[string]GraphNode, len(nodes))
	s.edgeByID = make(map[string]GraphEdge, len(edges))
	for _, node := range nodes {
		normalized := normalizeGraphNode(node)
		if normalized.ID == "" {
			continue
		}
		s.nodeByID[normalized.ID] = normalized
	}
	for _, edge := range edges {
		normalized := normalizeGraphEdge(edge)
		if normalized.ID == "" || normalized.Predicate == "" {
			continue
		}
		s.edgeByID[normalized.ID] = normalized
	}
	s.rebuildIndexesLocked()
	for aliasKey, nodeID := range aliases {
		if _, ok := s.nodeByID[nodeID]; !ok {
			continue
		}
		trimmed := strings.TrimSpace(aliasKey)
		if trimmed == "" {
			continue
		}
		s.nodeByAlias[trimmed] = nodeID
	}
	return nil
}

func readGraphNodes(path string) ([]GraphNode, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read graph nodes: %w", err)
	}
	var payload graphNodesPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode graph nodes: %w", err)
	}
	return payload.Nodes, nil
}

func readGraphEdges(path string) ([]GraphEdge, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read graph edges: %w", err)
	}
	var payload graphEdgesPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode graph edges: %w", err)
	}
	return payload.Edges, nil
}

func readGraphAliases(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read graph aliases: %w", err)
	}
	var payload graphAliasesPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode graph aliases: %w", err)
	}
	return payload.Aliases, nil
}

func (s *GraphStore) Persist() error {
	if !s.enabled() {
		return nil
	}
	if err := os.MkdirAll(s.baseDir, 0o700); err != nil {
		return fmt.Errorf("create graph directory: %w", err)
	}

	s.mu.RLock()
	nodes := s.listNodesLocked("")
	edges := s.listEdgesLocked("")
	aliases := make(map[string]string, len(s.nodeByAlias))
	for key, value := range s.nodeByAlias {
		aliases[key] = value
	}
	s.mu.RUnlock()

	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	sort.Slice(edges, func(i, j int) bool { return edges[i].ID < edges[j].ID })

	if err := writeGraphJSON(filepath.Join(s.baseDir, defaultGraphNodesPathName), graphNodesPayload{Nodes: nodes}); err != nil {
		return err
	}
	if err := writeGraphJSON(filepath.Join(s.baseDir, defaultGraphEdgesPathName), graphEdgesPayload{Edges: edges}); err != nil {
		return err
	}
	if err := writeGraphJSON(filepath.Join(s.baseDir, defaultGraphAliasPathName), graphAliasesPayload{Aliases: aliases}); err != nil {
		return err
	}
	return nil
}

func writeGraphJSON(path string, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal graph file %s: %w", filepath.Base(path), err)
	}
	data = append(data, '\n')
	tmpPath := fmt.Sprintf("%s.tmp-%d", path, time.Now().UTC().UnixNano())
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write graph temp file %s: %w", filepath.Base(path), err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace graph file %s: %w", filepath.Base(path), err)
	}
	return nil
}

func (s *GraphStore) ApplyFacts(namespace string, facts []GraphFact, resolver *GraphResolver) (GraphApplyStats, error) {
	if !s.enabled() || len(facts) == 0 {
		return GraphApplyStats{}, nil
	}
	if resolver == nil {
		resolver = NewGraphResolver()
	}

	stats := GraphApplyStats{}
	touchedNodes := make(map[string]struct{}, len(facts)*2)
	touchedEdges := make(map[string]struct{}, len(facts))

	s.mu.Lock()
	for _, rawFact := range facts {
		fact := normalizeGraphFact(rawFact)
		if fact.Predicate == "" || fact.Subject == "" || fact.Object == "" {
			continue
		}
		ns := normalizeGraphNamespace(firstNonEmpty(fact.Namespace, namespace))
		subjectExisted := s.hasNodeLocked(ns, fact.Subject)
		objectExisted := s.hasNodeLocked(ns, fact.Object)
		subject := resolver.resolveNode(s, ns, fact.Subject, nil, fact.SubjectType, fact.AnchorKeys, fact.SourceID, fact.Timestamp)
		object := resolver.resolveNode(s, ns, fact.Object, fact.Aliases, fact.ObjectType, fact.AnchorKeys, fact.SourceID, fact.Timestamp)
		if subject.ID == "" || object.ID == "" || subject.ID == object.ID {
			continue
		}
		if !subjectExisted {
			stats.NodesUpserted++
		}
		if !objectExisted {
			stats.NodesUpserted++
		}
		touchedNodes[subject.ID] = struct{}{}
		touchedNodes[object.ID] = struct{}{}
		edge, created := s.upsertEdgeLocked(ns, subject, object, fact)
		if edge.ID == "" {
			continue
		}
		if created {
			stats.EdgesUpserted++
		}
		touchedEdges[edge.ID] = struct{}{}
		stats.FactsApplied++
	}
	s.updatedAt = time.Now().UTC()
	s.mu.Unlock()

	if stats.FactsApplied == 0 {
		return stats, nil
	}
	return stats, s.Persist()
}

func (s *GraphStore) upsertEdgeLocked(namespace string, subject GraphNode, object GraphNode, fact GraphFact) (GraphEdge, bool) {
	key := graphEdgeLookupKey(namespace, subject.ID, fact.Predicate, object.ID)
	if edgeID, ok := s.edgeByKey[key]; ok {
		edge := s.edgeByID[edgeID]
		merged := mergeGraphEdge(edge, fact)
		merged = s.reconcileSingleValueLocked(merged)
		s.saveEdgeLocked(merged)
		return merged, false
	}

	now := fact.Timestamp.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	edge := normalizeGraphEdge(GraphEdge{
		ID:          fmt.Sprintf("graph-edge-%d", time.Now().UTC().UnixNano()),
		Namespace:   namespace,
		SubjectID:   subject.ID,
		Predicate:   fact.Predicate,
		ObjectID:    object.ID,
		Confidence:  maxFloat(fact.Confidence, defaultGraphMinConfidence),
		Weight:      maxFloat(fact.Confidence, 0.5),
		Status:      GraphStatusActive,
		FirstSeenAt: now,
		LastSeenAt:  now,
		SessionIDs:  uniqueStrings([]string{fact.SessionID}),
		SourceIDs:   uniqueStrings([]string{fact.SourceID}),
		Metadata:    mergeGraphMetadata(nil, fact.Metadata),
		Evidence: []GraphEvidence{{
			SessionID: fact.SessionID,
			SourceID:  fact.SourceID,
			Snippet:   fact.Snippet,
			Timestamp: now,
			TraceID:   fact.TraceID,
		}},
	})
	edge = s.reconcileSingleValueLocked(edge)
	s.saveEdgeLocked(edge)
	return edge, true
}

func mergeGraphEdge(edge GraphEdge, fact GraphFact) GraphEdge {
	out := normalizeGraphEdge(edge)
	when := fact.Timestamp.UTC()
	if when.IsZero() {
		when = time.Now().UTC()
	}
	if out.FirstSeenAt.IsZero() || when.Before(out.FirstSeenAt) {
		out.FirstSeenAt = when
	}
	if when.After(out.LastSeenAt) {
		out.LastSeenAt = when
	}
	if out.Weight <= 0 {
		out.Weight = 1
	}
	out.Weight += maxFloat(fact.Confidence, 0.5)
	out.Confidence = clamp01(math.Max(out.Confidence, (out.Confidence+fact.Confidence)/2))
	out.SessionIDs = uniqueStrings(append(out.SessionIDs, fact.SessionID))
	out.SourceIDs = uniqueStrings(append(out.SourceIDs, fact.SourceID))
	out.Metadata = mergeGraphMetadata(out.Metadata, fact.Metadata)
	out.Evidence = normalizeGraphEvidence(append(out.Evidence, GraphEvidence{
		SessionID: fact.SessionID,
		SourceID:  fact.SourceID,
		Snippet:   fact.Snippet,
		Timestamp: when,
		TraceID:   fact.TraceID,
	}))
	if out.Status == GraphStatusSuperseded {
		out.Status = GraphStatusActive
	}
	return normalizeGraphEdge(out)
}

func mergeGraphMetadata(base map[string]any, incoming map[string]any) map[string]any {
	if len(base) == 0 && len(incoming) == 0 {
		return nil
	}
	out := cloneGraphMetadata(base)
	if out == nil {
		out = make(map[string]any, len(incoming))
	}
	for key, value := range incoming {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		incomingStrings := graphMetadataStrings(value)
		if len(incomingStrings) > 0 {
			merged := uniqueStrings(append(graphMetadataStrings(out[trimmedKey]), incomingStrings...))
			if len(merged) > 0 {
				out[trimmedKey] = merged
			}
			continue
		}
		switch typed := value.(type) {
		case bool:
			if typed || out[trimmedKey] == nil {
				out[trimmedKey] = typed
			}
		case string:
			if strings.TrimSpace(typed) == "" {
				continue
			}
			if strings.TrimSpace(graphMetadataString(out[trimmedKey])) == "" {
				out[trimmedKey] = strings.TrimSpace(typed)
			}
		case map[string]any:
			out[trimmedKey] = mergeGraphMetadata(graphMetadataMap(out[trimmedKey]), typed)
		default:
			if out[trimmedKey] == nil {
				out[trimmedKey] = typed
			}
		}
	}
	return out
}

func graphMetadataMap(value any) map[string]any {
	mapValue, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return mapValue
}

func graphMetadataString(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func graphMetadataStrings(value any) []string {
	switch typed := value.(type) {
	case []string:
		return uniqueStrings(append([]string(nil), typed...))
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				out = append(out, strings.TrimSpace(text))
			}
		}
		return uniqueStrings(out)
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{strings.TrimSpace(typed)}
	default:
		return nil
	}
}

func (s *GraphStore) reconcileSingleValueLocked(edge GraphEdge) GraphEdge {
	if _, ok := singleValueGraphPredicates[edge.Predicate]; !ok {
		return normalizeGraphEdge(edge)
	}
	out := normalizeGraphEdge(edge)
	competitors := append(s.activeEdgesBySubjectPredicateLocked(edge.Namespace, edge.SubjectID, edge.Predicate), s.activeEdgesByObjectPredicateLocked(edge.Namespace, edge.ObjectID, edge.Predicate)...)
	for _, existing := range competitors {
		if existing.ID == out.ID || existing.ObjectID == out.ObjectID {
			if existing.ID == out.ID {
				continue
			}
		}
		if existing.ID == out.ID {
			continue
		}
		if singleValueEdgesConflict(existing, out) {
			existing.Status = GraphStatusConflicted
			out.Status = GraphStatusConflicted
			s.saveEdgeLocked(existing)
			continue
		}
		if newerOrStrongerEdge(existing, out) {
			existing.Status = GraphStatusSuperseded
			s.saveEdgeLocked(existing)
			if out.Status != GraphStatusConflicted {
				out.Status = GraphStatusActive
			}
			continue
		}
		out.Status = GraphStatusSuperseded
	}
	return normalizeGraphEdge(out)
}

func singleValueEdgesConflict(existing GraphEdge, incoming GraphEdge) bool {
	confDelta := math.Abs(existing.Confidence - incoming.Confidence)
	timeDelta := existing.LastSeenAt.Sub(incoming.LastSeenAt)
	if timeDelta < 0 {
		timeDelta = -timeDelta
	}
	return confDelta <= 0.08 && timeDelta <= 24*time.Hour
}

func newerOrStrongerEdge(existing GraphEdge, incoming GraphEdge) bool {
	if incoming.LastSeenAt.After(existing.LastSeenAt.Add(time.Minute)) {
		return true
	}
	return incoming.Confidence >= existing.Confidence+0.05
}

func (s *GraphStore) activeEdgesBySubjectPredicateLocked(namespace, subjectID, predicate string) []GraphEdge {
	ids := s.edgesBySubject[strings.TrimSpace(subjectID)]
	if len(ids) == 0 {
		return nil
	}
	out := make([]GraphEdge, 0, len(ids))
	for _, edgeID := range ids {
		edge, ok := s.edgeByID[edgeID]
		if !ok {
			continue
		}
		if edge.Namespace != normalizeGraphNamespace(namespace) || edge.Predicate != normalizeGraphPredicate(predicate) {
			continue
		}
		if edge.Status != GraphStatusActive {
			continue
		}
		out = append(out, edge)
	}
	return out
}

func (s *GraphStore) activeEdgesByObjectPredicateLocked(namespace, objectID, predicate string) []GraphEdge {
	ids := s.edgesByObject[strings.TrimSpace(objectID)]
	if len(ids) == 0 {
		return nil
	}
	out := make([]GraphEdge, 0, len(ids))
	for _, edgeID := range ids {
		edge, ok := s.edgeByID[edgeID]
		if !ok {
			continue
		}
		if edge.Namespace != normalizeGraphNamespace(namespace) || edge.Predicate != normalizeGraphPredicate(predicate) {
			continue
		}
		if edge.Status != GraphStatusActive {
			continue
		}
		out = append(out, edge)
	}
	return out
}

func (s *GraphStore) lookupNodeLocked(namespace string, value string) (GraphNode, bool) {
	for _, variant := range graphLookupVariants(value) {
		key := normalizeGraphNamespace(namespace) + "|" + variant
		if nodeID, ok := s.nodeByCanonical[key]; ok {
			node, exists := s.nodeByID[nodeID]
			return node, exists
		}
		if nodeID, ok := s.nodeByAlias[key]; ok {
			node, exists := s.nodeByID[nodeID]
			return node, exists
		}
	}
	return GraphNode{}, false
}

func (s *GraphStore) hasNodeLocked(namespace string, value string) bool {
	_, ok := s.lookupNodeLocked(namespace, value)
	return ok
}

func (s *GraphStore) saveNodeLocked(node GraphNode) {
	normalized := normalizeGraphNode(node)
	if normalized.ID == "" {
		return
	}
	s.nodeByID[normalized.ID] = normalized
	for _, variant := range graphLookupVariants(normalized.CanonicalName) {
		s.nodeByCanonical[normalizeGraphNamespace(normalized.Namespace)+"|"+variant] = normalized.ID
	}
	for _, alias := range normalized.Aliases {
		for _, variant := range graphLookupVariants(alias) {
			s.nodeByAlias[normalizeGraphNamespace(normalized.Namespace)+"|"+variant] = normalized.ID
		}
	}
}

func (s *GraphStore) saveEdgeLocked(edge GraphEdge) {
	normalized := normalizeGraphEdge(edge)
	if normalized.ID == "" || normalized.Predicate == "" {
		return
	}
	s.edgeByID[normalized.ID] = normalized
	key := graphEdgeLookupKey(normalized.Namespace, normalized.SubjectID, normalized.Predicate, normalized.ObjectID)
	s.edgeByKey[key] = normalized.ID
	s.rebuildEdgeIndexesLocked()
}

func (s *GraphStore) rebuildIndexesLocked() {
	s.nodeByCanonical = make(map[string]string, len(s.nodeByID))
	s.nodeByAlias = make(map[string]string, len(s.nodeByID)*2)
	for _, node := range s.nodeByID {
		for _, variant := range graphLookupVariants(node.CanonicalName) {
			s.nodeByCanonical[normalizeGraphNamespace(node.Namespace)+"|"+variant] = node.ID
		}
		for _, alias := range node.Aliases {
			for _, variant := range graphLookupVariants(alias) {
				s.nodeByAlias[normalizeGraphNamespace(node.Namespace)+"|"+variant] = node.ID
			}
		}
	}
	s.rebuildEdgeIndexesLocked()
}

func (s *GraphStore) rebuildEdgeIndexesLocked() {
	s.edgeByKey = make(map[string]string, len(s.edgeByID))
	s.edgesBySubject = make(map[string][]string, len(s.edgeByID))
	s.edgesByObject = make(map[string][]string, len(s.edgeByID))
	s.edgesByPredicate = make(map[string][]string, len(s.edgeByID))
	for _, edge := range s.edgeByID {
		key := graphEdgeLookupKey(edge.Namespace, edge.SubjectID, edge.Predicate, edge.ObjectID)
		s.edgeByKey[key] = edge.ID
		s.edgesBySubject[edge.SubjectID] = appendUniqueString(s.edgesBySubject[edge.SubjectID], edge.ID)
		s.edgesByObject[edge.ObjectID] = appendUniqueString(s.edgesByObject[edge.ObjectID], edge.ID)
		predicateKey := normalizeGraphNamespace(edge.Namespace) + "|" + edge.Predicate
		s.edgesByPredicate[predicateKey] = appendUniqueString(s.edgesByPredicate[predicateKey], edge.ID)
	}
}

func appendUniqueString(values []string, value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return values
	}
	for _, existing := range values {
		if existing == trimmed {
			return values
		}
	}
	return append(values, trimmed)
}

func (s *GraphStore) listNodesLocked(namespace string) []GraphNode {
	out := make([]GraphNode, 0, len(s.nodeByID))
	ns := normalizeGraphNamespace(namespace)
	for _, node := range s.nodeByID {
		if namespace != "" && node.Namespace != ns {
			continue
		}
		out = append(out, node)
	}
	return out
}

func (s *GraphStore) listEdgesLocked(namespace string) []GraphEdge {
	out := make([]GraphEdge, 0, len(s.edgeByID))
	ns := normalizeGraphNamespace(namespace)
	for _, edge := range s.edgeByID {
		if namespace != "" && edge.Namespace != ns {
			continue
		}
		out = append(out, edge)
	}
	return out
}

func (s *GraphStore) ListNodes(namespace string) []GraphNode {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	out := s.listNodesLocked(namespace)
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastSeenAt.Equal(out[j].LastSeenAt) {
			return out[i].CanonicalName < out[j].CanonicalName
		}
		return out[i].LastSeenAt.After(out[j].LastSeenAt)
	})
	return out
}

func (s *GraphStore) ListEdges(namespace string) []GraphEdge {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	out := s.listEdgesLocked(namespace)
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastSeenAt.Equal(out[j].LastSeenAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].LastSeenAt.After(out[j].LastSeenAt)
	})
	return out
}

func (s *GraphStore) Node(nodeID string) (GraphNode, bool) {
	if s == nil {
		return GraphNode{}, false
	}
	s.mu.RLock()
	node, ok := s.nodeByID[strings.TrimSpace(nodeID)]
	s.mu.RUnlock()
	return node, ok
}

func (s *GraphStore) Edge(edgeID string) (GraphEdge, bool) {
	if s == nil {
		return GraphEdge{}, false
	}
	s.mu.RLock()
	edge, ok := s.edgeByID[strings.TrimSpace(edgeID)]
	s.mu.RUnlock()
	return edge, ok
}

func (s *GraphStore) OutEdges(namespace, nodeID string) []GraphEdge {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	ids := append([]string(nil), s.edgesBySubject[strings.TrimSpace(nodeID)]...)
	out := make([]GraphEdge, 0, len(ids))
	ns := normalizeGraphNamespace(namespace)
	for _, edgeID := range ids {
		edge, ok := s.edgeByID[edgeID]
		if !ok || edge.Namespace != ns {
			continue
		}
		out = append(out, edge)
	}
	s.mu.RUnlock()
	return out
}

func (s *GraphStore) InEdges(namespace, nodeID string) []GraphEdge {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	ids := append([]string(nil), s.edgesByObject[strings.TrimSpace(nodeID)]...)
	out := make([]GraphEdge, 0, len(ids))
	ns := normalizeGraphNamespace(namespace)
	for _, edgeID := range ids {
		edge, ok := s.edgeByID[edgeID]
		if !ok || edge.Namespace != ns {
			continue
		}
		out = append(out, edge)
	}
	s.mu.RUnlock()
	return out
}

func (s *GraphStore) LookupNode(namespace string, value string) (GraphNode, bool) {
	if s == nil {
		return GraphNode{}, false
	}
	s.mu.RLock()
	node, ok := s.lookupNodeLocked(namespace, value)
	s.mu.RUnlock()
	return node, ok
}

func (s *GraphStore) Stats(namespace string) GraphStats {
	if s == nil {
		return GraphStats{Namespace: normalizeGraphNamespace(namespace)}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := GraphStats{Namespace: normalizeGraphNamespace(namespace), UpdatedAt: s.updatedAt}
	for _, node := range s.nodeByID {
		if namespace != "" && node.Namespace != stats.Namespace {
			continue
		}
		stats.NodeCount++
		stats.AliasCount += len(node.Aliases)
	}
	for _, edge := range s.edgeByID {
		if namespace != "" && edge.Namespace != stats.Namespace {
			continue
		}
		stats.EdgeCount++
		if edge.Status == GraphStatusActive {
			stats.ActiveEdgeCount++
		}
	}
	return stats
}

func (s *GraphStore) ResetNamespace(namespace string) error {
	if !s.enabled() {
		return nil
	}
	ns := normalizeGraphNamespace(namespace)
	s.mu.Lock()
	for nodeID, node := range s.nodeByID {
		if node.Namespace == ns {
			delete(s.nodeByID, nodeID)
		}
	}
	for edgeID, edge := range s.edgeByID {
		if edge.Namespace == ns {
			delete(s.edgeByID, edgeID)
		}
	}
	s.updatedAt = time.Now().UTC()
	s.rebuildIndexesLocked()
	s.mu.Unlock()
	return s.Persist()
}
