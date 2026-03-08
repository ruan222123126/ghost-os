package graph

import (
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

// GraphService 作为 graph sidecar，承接抽取、写入、检索与重建编排。
type GraphService struct {
	store            *GraphStore
	resolver         *GraphResolver
	extractor        *GraphExtractor
	cold             ColdStore
	enabled          bool
	namespace        string
	maxHops          int
	maxHits          int
	minConfidence    float64
	extractOnArchive bool
	extractOnEvolve  bool
	debugEnabled     bool
}

func NewGraphService(config MemoryConfig, cold ColdStore, summarizer Summarizer) *GraphService {
	service := &GraphService{
		enabled:          config.GraphEnabled,
		namespace:        normalizeGraphNamespace(config.GraphNamespace),
		maxHops:          config.GraphMaxHops,
		maxHits:          config.GraphMaxHits,
		minConfidence:    clamp01(maxFloat(config.GraphMinConfidence, defaultGraphMinConfidence)),
		extractOnArchive: config.GraphExtractOnArchive,
		extractOnEvolve:  config.GraphExtractOnEvolve,
		debugEnabled:     config.GraphDebugEnabled,
		cold:             cold,
		resolver:         NewGraphResolver(),
		extractor:        NewGraphExtractor(summarizer, config.EvolutionUseWorker, config.GraphMinConfidence),
	}
	if service.maxHops <= 0 {
		service.maxHops = defaultGraphMaxHops
	}
	if service.maxHits <= 0 {
		service.maxHits = defaultGraphMaxHits
	}
	if !service.enabled {
		return service
	}
	service.store = NewGraphStore(config.GraphPath)
	if err := service.store.Load(); err != nil {
		log.Printf("[MEMORY] graph sidecar load failed, fallback to empty store: %v", err)
		service.store = NewGraphStore(config.GraphPath)
	}
	return service
}

func (g *GraphService) Enabled() bool {
	return g != nil && g.enabled && g.store != nil
}

func (g *GraphService) IngestArchiveMessages(sessionID string, messages []llm.Message) error {
	if !g.Enabled() || !g.extractOnArchive || len(messages) == 0 {
		return nil
	}
	facts := g.extractor.ExtractFromMessages(messages, GraphExtractOptions{
		Namespace: g.namespace,
		SessionID: strings.TrimSpace(sessionID),
		SourceID:  "archive:" + strings.TrimSpace(sessionID),
		Timestamp: time.Now().UTC(),
	})
	_, err := g.store.ApplyFacts(g.namespace, facts, g.resolver)
	return err
}

func (g *GraphService) IngestMarkdownNode(node MarkdownNode) error {
	if !g.Enabled() || !g.extractOnEvolve {
		return nil
	}
	facts := g.extractor.ExtractFromMarkdownNode(node, g.namespace)
	_, err := g.store.ApplyFacts(g.namespace, facts, g.resolver)
	return err
}

func (g *GraphService) Retrieve(query MemoryQuery, scope SessionScope) ([]MemoryEntry, []GraphHit, error) {
	if !g.Enabled() || !query.IncludeGraph {
		return nil, nil, nil
	}
	namespace := g.resolveNamespace(query)
	queryText := buildGraphQueryText(query)
	if strings.TrimSpace(queryText) == "" {
		return nil, nil, nil
	}
	seeds := g.seedNodes(namespace, queryText, scope)
	if len(seeds) == 0 {
		return nil, nil, nil
	}
	hits := g.traverse(namespace, seeds, queryText, query)
	if len(hits) == 0 {
		return nil, nil, nil
	}
	entries := make([]MemoryEntry, 0, len(hits))
	debugHits := make([]GraphHit, 0, len(hits))
	for _, hit := range hits {
		edge, ok := g.store.Edge(hit.EdgeID)
		if !ok {
			continue
		}
		entries = append(entries, memoryEntryFromGraphHit(hit, edge))
		if query.GraphDebug && g.debugEnabled {
			debugHits = append(debugHits, hit)
		}
	}
	return entries, debugHits, nil
}

func (g *GraphService) GraphStats(namespace string) GraphStats {
	if !g.Enabled() {
		return GraphStats{Namespace: normalizeGraphNamespace(firstNonEmpty(namespace, g.namespace))}
	}
	return g.store.Stats(firstNonEmpty(namespace, g.namespace))
}

func (g *GraphService) resolveNamespace(query MemoryQuery) string {
	if query.Metadata != nil {
		if raw, ok := query.Metadata["namespace"].(string); ok && strings.TrimSpace(raw) != "" {
			return normalizeGraphNamespace(raw)
		}
	}
	return normalizeGraphNamespace(g.namespace)
}

func buildGraphQueryText(query MemoryQuery) string {
	parts := make([]string, 0, 2+len(query.Keywords))
	if semantic := strings.TrimSpace(query.SemanticQuery); semantic != "" {
		parts = append(parts, semantic)
	}
	parts = append(parts, query.Keywords...)
	return strings.TrimSpace(strings.Join(parts, " "))
}

func (g *GraphService) seedNodes(namespace string, queryText string, scope SessionScope) []GraphNode {
	allNodes := g.store.ListNodes(namespace)
	if len(allNodes) == 0 {
		return nil
	}
	lowerQuery := strings.ToLower(queryText)
	seen := make(map[string]struct{}, 8)
	out := make([]GraphNode, 0, 4)
	for _, node := range allNodes {
		if graphQueryMentionsNode(lowerQuery, node) {
			if _, ok := seen[node.ID]; ok {
				continue
			}
			seen[node.ID] = struct{}{}
			out = append(out, node)
		}
	}
	if len(out) > 0 {
		return out
	}
	recent := g.buildRecentEntityContext(namespace, scope.History)
	for term := range graphWeakReferenceByType {
		if !strings.Contains(queryText, term) {
			continue
		}
		nodeID := g.resolver.resolveWeakReference(term, recent)
		if nodeID == "" {
			continue
		}
		node, ok := g.store.Node(nodeID)
		if !ok {
			continue
		}
		if _, ok := seen[node.ID]; ok {
			continue
		}
		seen[node.ID] = struct{}{}
		out = append(out, node)
	}
	return out
}

func graphQueryMentionsNode(queryText string, node GraphNode) bool {
	if queryText == "" {
		return false
	}
	normalizedQuery := normalizeGraphLookup(queryText)
	for _, candidate := range append([]string{node.CanonicalName}, node.Aliases...) {
		for _, lookup := range graphLookupVariants(candidate) {
			if lookup == "" || len(lookup) < 1 {
				continue
			}
			if strings.Contains(queryText, lookup) || strings.Contains(normalizedQuery, lookup) {
				return true
			}
		}
	}
	return false
}

func (g *GraphService) buildRecentEntityContext(namespace string, history *agent.History) RecentEntityContext {
	context := RecentEntityContext{LatestByType: make(map[string]string)}
	if history == nil {
		return context
	}
	messages := history.Messages()
	if len(messages) == 0 {
		return context
	}
	nodes := g.store.ListNodes(namespace)
	for i := len(messages) - 1; i >= 0 && len(context.LatestByType) < 4; i-- {
		text := strings.ToLower(messageToContent(messages[i]))
		if strings.TrimSpace(text) == "" {
			continue
		}
		for _, node := range nodes {
			if _, ok := context.LatestByType[node.Type]; ok {
				continue
			}
			if graphQueryMentionsNode(text, node) {
				context.LatestByType[node.Type] = node.ID
			}
		}
	}
	return context
}

func (g *GraphService) traverse(namespace string, seeds []GraphNode, queryText string, query MemoryQuery) []GraphHit {
	maxHops := g.maxHops
	if query.GraphHops > 0 && query.GraphHops < maxHops {
		maxHops = query.GraphHops
	}
	if query.GraphHops > 0 && query.GraphHops > maxHops {
		maxHops = query.GraphHops
	}
	if maxHops <= 0 {
		maxHops = defaultGraphMaxHops
	}
	if maxHops > 2 {
		maxHops = 2
	}

	allowedPredicates := g.allowedPredicates(query.GraphPredicates)
	includeHistorical := graphQueryRequestsHistory(queryText)
	seedSet := make(map[string]struct{}, len(seeds))
	queue := make([]graphTraversalNode, 0, len(seeds))
	for _, seed := range seeds {
		seedSet[seed.ID] = struct{}{}
		queue = append(queue, graphTraversalNode{Node: seed, Hop: 0})
	}

	visitedNodes := make(map[string]int, len(seeds))
	visitedEdges := make(map[string]int, len(seeds)*2)
	hits := make([]GraphHit, 0, g.maxHits)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if hop, ok := visitedNodes[current.Node.ID]; ok && hop <= current.Hop {
			continue
		}
		visitedNodes[current.Node.ID] = current.Hop
		if current.Hop >= maxHops {
			continue
		}
		edges := append(g.store.OutEdges(namespace, current.Node.ID), g.store.InEdges(namespace, current.Node.ID)...)
		for _, edge := range edges {
			if !g.edgeAllowed(edge, allowedPredicates, includeHistorical) {
				continue
			}
			hop := current.Hop + 1
			if seenHop, ok := visitedEdges[edge.ID]; ok && seenHop <= hop {
				continue
			}
			visitedEdges[edge.ID] = hop
			subject, okSubject := g.store.Node(edge.SubjectID)
			object, okObject := g.store.Node(edge.ObjectID)
			if !okSubject || !okObject {
				continue
			}
			score := scoreGraphEdge(queryText, current.Node, subject, edge, object, hop)
			hit := GraphHit{
				Namespace:     namespace,
				NodeID:        current.Node.ID,
				EdgeID:        edge.ID,
				Subject:       graphDisplayName(subject),
				Predicate:     edge.Predicate,
				Object:        graphDisplayName(object),
				Status:        edge.Status,
				Hop:           hop,
				Score:         score,
				Content:       formatGraphFact(subject, edge.Predicate, object),
				SourceIDs:     append([]string(nil), edge.SourceIDs...),
				EvidenceCount: len(edge.Evidence),
			}
			if query.GraphDebug && g.debugEnabled {
				hit.Evidence = append([]GraphEvidence(nil), edge.Evidence...)
			}
			hits = append(hits, hit)
			nextID := edge.ObjectID
			if current.Node.ID == edge.ObjectID {
				nextID = edge.SubjectID
			}
			nextNode, ok := g.store.Node(nextID)
			if !ok {
				continue
			}
			if _, ok := seedSet[nextNode.ID]; ok && hop >= maxHops {
				continue
			}
			queue = append(queue, graphTraversalNode{Node: nextNode, Hop: hop})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		if hits[i].Hop != hits[j].Hop {
			return hits[i].Hop < hits[j].Hop
		}
		return hits[i].EdgeID < hits[j].EdgeID
	})
	if len(hits) > g.maxHits {
		hits = hits[:g.maxHits]
	}
	return hits
}

type graphTraversalNode struct {
	Node GraphNode
	Hop  int
}

func (g *GraphService) edgeAllowed(edge GraphEdge, allowedPredicates map[string]struct{}, includeHistorical bool) bool {
	if edge.Confidence > 0 && edge.Confidence < g.minConfidence {
		return false
	}
	if len(allowedPredicates) > 0 {
		if _, ok := allowedPredicates[edge.Predicate]; !ok {
			return false
		}
	}
	if includeHistorical {
		return true
	}
	return edge.Status == GraphStatusActive
}

func (g *GraphService) allowedPredicates(predicates []string) map[string]struct{} {
	if len(predicates) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(predicates))
	for _, predicate := range predicates {
		normalized := normalizeGraphPredicate(predicate)
		if normalized == "" {
			continue
		}
		out[normalized] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func graphQueryRequestsHistory(queryText string) bool {
	lower := strings.ToLower(strings.TrimSpace(queryText))
	for _, token := range []string{"history", "historical", "before", "previous", "之前", "历史", "曾经"} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func scoreGraphEdge(queryText string, seed GraphNode, subject GraphNode, edge GraphEdge, object GraphNode, hop int) float64 {
	terms := queryTerms(MemoryQuery{SemanticQuery: queryText})
	entityMatch := 0.72
	if graphQueryMentionsNode(strings.ToLower(queryText), subject) || graphQueryMentionsNode(strings.ToLower(queryText), object) {
		entityMatch = 1
	}
	if seed.ID == subject.ID || seed.ID == object.ID {
		entityMatch = 1
	}
	edgeConfidence := clamp01(edge.Confidence)
	temporal := scoreGraphTemporal(edge)
	anchor := scoreGraphAnchor(terms, subject, object)
	evidence := scoreGraphEvidence(edge)
	hopPenalty := 0.0
	if hop > 1 {
		hopPenalty = 0.6
	}
	score := entityMatch*0.3 + edgeConfidence*0.2 + temporal*0.2 + anchor*0.15 + evidence*0.1 - hopPenalty*0.05
	return clamp01(score)
}

func scoreGraphTemporal(edge GraphEdge) float64 {
	when := edge.LastSeenAt.UTC()
	if when.IsZero() {
		when = edge.FirstSeenAt.UTC()
	}
	if when.IsZero() {
		return 0.4
	}
	age := time.Since(when)
	if age < 0 {
		age = 0
	}
	halfLife := 30 * 24 * time.Hour
	return clamp01(math.Exp2(-age.Hours()/halfLife.Hours()) + 0.1)
}

func scoreGraphAnchor(terms []string, nodes ...GraphNode) float64 {
	if len(terms) == 0 {
		return 0.5
	}
	best := 0.0
	for _, node := range nodes {
		for _, anchorKey := range node.AnchorKeys {
			lookup := normalizeGraphLookup(anchorKey)
			if lookup == "" {
				continue
			}
			for _, term := range terms {
				if strings.Contains(lookup, term) || strings.Contains(term, lookup) {
					if best < 0.9 {
						best = 0.9
					}
				}
			}
		}
	}
	if best == 0 {
		return 0.4
	}
	return best
}

func scoreGraphEvidence(edge GraphEdge) float64 {
	if len(edge.Evidence) == 0 {
		return 0.2
	}
	score := 0.35 + clamp01(float64(len(edge.Evidence))/3)*0.45
	for _, evidence := range edge.Evidence {
		if strings.TrimSpace(evidence.Snippet) != "" {
			score += 0.1
			break
		}
	}
	return clamp01(score)
}
