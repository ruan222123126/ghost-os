package memory

import (
	"fmt"
	"log"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

// QueryService 收口自动召回与级联查询逻辑。
type QueryService struct {
	warm     *WarmMemory
	cold     *ColdMemory
	graph    *GraphService
	decision *DecisionService

	autoRecallEnabled bool
	autoRecallLimit   int
	scoring           memoryScoringConfig
	metrics           *memoryCounters
}

func NewQueryService(config MemoryConfig, warm *WarmMemory, cold *ColdMemory, graph *GraphService, decision *DecisionService, metrics *memoryCounters) *QueryService {
	return &QueryService{
		warm:              warm,
		cold:              cold,
		graph:             graph,
		decision:          decision,
		autoRecallEnabled: config.AutoRecallEnabled,
		autoRecallLimit:   config.AutoRecallLimit,
		scoring:           newMemoryScoringConfig(config),
		metrics:           metrics,
	}
}

func (s *QueryService) BuildContextWindow(scope SessionScope, userInput string) ([]llm.Message, error) {
	if !s.autoRecallEnabled {
		return nil, nil
	}

	query := MemoryQuery{
		Limit:             max(s.autoRecallLimit*6, s.autoRecallLimit+8),
		Keywords:          extractKeywords(userInput),
		SemanticQuery:     strings.TrimSpace(userInput),
		IncludeMarkdown:   true,
		IncludeGraph:      true,
		IncludeDecision:   true,
		DecisionReuseOnly: true,
		DecisionTypes:     []string{DecisionHitTypeRecipe, DecisionHitTypeMemo, DecisionHitTypeWarning},
		EnvironmentStrict: scope.Environment != nil,
		PreferRecent:      true,
	}
	if scope.Environment != nil {
		env := cloneDecisionEnvFingerprint(*scope.Environment)
		query.Environment = &env
	}
	if sid := strings.TrimSpace(scope.SessionID); sid != "" {
		query.Metadata = map[string]any{"session_id": sid}
	}

	result, err := s.QueryResultWithScope(query, scope)
	if err != nil {
		return nil, err
	}
	entries := result.Entries
	if len(entries) == 0 {
		return nil, nil
	}

	visible := visibleContextFingerprints(scope.History)
	lines := make([]string, 0, minInt(len(entries), s.autoRecallLimit)+1)
	for _, entry := range entries {
		if recallEntryVisible(entry, visible) {
			continue
		}
		line := compactRecallLine(entry)
		if line == "" {
			continue
		}
		lines = append(lines, "- "+line)
		if len(lines) >= s.autoRecallLimit {
			break
		}
	}
	if len(lines) == 0 {
		return nil, nil
	}

	return []llm.Message{{
		Role: llm.RoleSystem,
		Text: strings.Join(append([]string{"Recalled context:"}, lines...), "\n"),
	}}, nil
}

func visibleContextFingerprints(history *agent.History) map[string]struct{} {
	if history == nil {
		return nil
	}

	messages := history.Messages()
	if len(messages) == 0 {
		return nil
	}

	fingerprints := make(map[string]struct{}, len(messages))
	for _, msg := range messages {
		if msg.Role == llm.RoleSystem {
			continue
		}
		fingerprint := normalizeRecallText(messageToContent(msg))
		if fingerprint == "" {
			continue
		}
		fingerprints[fingerprint] = struct{}{}
	}
	if len(fingerprints) == 0 {
		return nil
	}
	return fingerprints
}

func recallEntryVisible(entry MemoryEntry, visible map[string]struct{}) bool {
	if len(visible) == 0 {
		return false
	}
	_, ok := visible[normalizeRecallText(firstNonEmpty(entry.Summary, entry.Content))]
	return ok
}

func normalizeRecallText(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return ""
	}
	return strings.Join(strings.Fields(normalized), " ")
}

func (s *QueryService) QueryResult(query MemoryQuery) (MemoryQueryResult, error) {
	return s.QueryResultWithScope(query, SessionScope{})
}

func (s *QueryService) Query(query MemoryQuery) ([]MemoryEntry, error) {
	result, err := s.QueryResult(query)
	if err != nil {
		return nil, err
	}
	return result.Entries, nil
}

func (s *QueryService) QueryWithScope(query MemoryQuery, scope SessionScope) ([]MemoryEntry, error) {
	result, err := s.QueryResultWithScope(query, scope)
	if err != nil {
		return nil, err
	}
	return result.Entries, nil
}

func (s *QueryService) QueryResultWithScope(query MemoryQuery, scope SessionScope) (MemoryQueryResult, error) {
	results := make([]MemoryEntry, 0, 32)
	seenIDs := make(map[string]struct{}, 32)
	seenFingerprints := make(map[string]struct{}, 32)
	now := time.Now().UTC()

	collect := func(entries []MemoryEntry) {
		for _, entry := range entries {
			if _, ok := seenIDs[entry.ID]; ok {
				continue
			}
			seenIDs[entry.ID] = struct{}{}
			fingerprint := normalizeRecallText(firstNonEmpty(entry.Summary, entry.Content))
			if fingerprint != "" {
				seenFingerprints[fingerprint] = struct{}{}
			}
			results = append(results, entry)
		}
	}
	collectDistinct := func(entries []MemoryEntry) {
		for _, entry := range entries {
			if _, ok := seenIDs[entry.ID]; ok {
				continue
			}
			fingerprint := normalizeRecallText(firstNonEmpty(entry.Summary, entry.Content))
			if fingerprint != "" {
				if _, ok := seenFingerprints[fingerprint]; ok {
					continue
				}
				seenFingerprints[fingerprint] = struct{}{}
			}
			seenIDs[entry.ID] = struct{}{}
			results = append(results, entry)
		}
	}

	hotEntries := queryHot(scope, query)
	if len(hotEntries) > 0 {
		s.metrics.l1Hits.Add(uint64(len(hotEntries)))
	}
	collect(hotEntries)

	warmQuery := query
	warmQuery.Limit = 0
	warmEntries, err := s.warm.RetrieveCandidates(warmQuery)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	if len(warmEntries) > 0 {
		s.metrics.l2Hits.Add(uint64(len(warmEntries)))
	}
	collect(warmEntries)

	var decisionHits []DecisionHit
	if s.decision != nil && query.IncludeDecision {
		decisionEntries, hits, err := s.decision.Retrieve(query, scope)
		if err != nil {
			log.Printf("[MEMORY] decision recall failed, falling back to other layers: %v", err)
		} else {
			collectDistinct(decisionEntries)
			decisionHits = hits
		}
	}

	var graphHits []GraphHit
	if s.graph != nil && query.IncludeGraph {
		graphEntries, hits, err := s.graph.Retrieve(query, scope)
		if err != nil {
			log.Printf("[MEMORY] graph recall failed, falling back to text layers: %v", err)
		} else {
			if len(graphEntries) > 0 {
				s.metrics.graphHits.Add(uint64(len(graphEntries)))
			}
			collectDistinct(graphEntries)
			graphHits = hits
		}
	}

	coldQuery := query
	coldQuery.Limit = 0
	coldEntries, err := s.cold.Retrieve(coldQuery)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	if len(coldEntries) > 0 {
		s.metrics.l3Hits.Add(uint64(len(coldEntries)))
	}
	collect(coldEntries)

	if query.IncludeMarkdown {
		markdownQuery := query
		markdownQuery.Limit = 0
		markdownEntries, err := s.queryMarkdown(markdownQuery)
		if err != nil {
			return MemoryQueryResult{}, err
		}
		if len(markdownEntries) > 0 {
			s.metrics.markdownHits.Add(uint64(len(markdownEntries)))
		}
		collect(markdownEntries)
	}

	results = rankMemoryEntries(results, query, now, s.scoring)
	if query.Limit > 0 && len(results) > query.Limit {
		results = results[:query.Limit]
	}

	warmHits := make([]string, 0, len(results))
	decisionMemoHits := make([]string, 0, len(results))
	for _, entry := range results {
		if entryLayer(entry) != "warm" {
			if entryLayer(entry) == "decision" {
				memoID, _ := entry.Metadata["memo_id"].(string)
				if strings.TrimSpace(memoID) != "" {
					decisionMemoHits = append(decisionMemoHits, strings.TrimSpace(memoID))
				}
			}
			continue
		}
		warmHits = append(warmHits, entry.ID)
	}
	s.warm.RecordAccess(warmHits, now)
	if s.decision != nil {
		s.decision.RecordMemoAccess(decisionMemoHits, now)
	}
	return MemoryQueryResult{Entries: results, GraphHits: graphHits, DecisionHits: decisionHits}, nil
}

func queryHot(scope SessionScope, query MemoryQuery) []MemoryEntry {
	if scope.History == nil {
		return nil
	}

	sessionID := strings.TrimSpace(scope.SessionID)
	messages := scope.History.Messages()
	if len(messages) == 0 {
		return nil
	}

	base := time.Now().UTC()
	out := make([]MemoryEntry, 0, len(messages))
	for i, msg := range messages {
		content := messageToContent(msg)
		entry := MemoryEntry{
			ID:             fmt.Sprintf("hot:%s:%06d", sessionID, i),
			Content:        content,
			Type:           MemoryTypeMessage,
			Timestamp:      base.Add(time.Duration(i) * time.Millisecond),
			LastAccessedAt: base.Add(time.Duration(i) * time.Millisecond),
			Source:         "hot",
			Summary:        summarizeLine(content, 220),
			Confidence:     1,
			Metadata: map[string]any{
				"layer":        "hot",
				"source":       "hot",
				"session_id":   sessionID,
				"role":         string(msg.Role),
				"tool_call_id": strings.TrimSpace(msg.ToolCallID),
			},
		}
		entry = normalizeEntry(entry)
		entry.Importance = calculateImportance(entry)
		if !entryMatchesQuery(entry, query) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func (s *QueryService) queryMarkdown(query MemoryQuery) ([]MemoryEntry, error) {
	nodes, err := s.cold.ListMarkdownNodes()
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}

	entries := make([]MemoryEntry, 0, len(nodes))
	for _, id := range nodes {
		node, err := s.cold.LoadMarkdownNode(id)
		if err != nil {
			continue
		}

		entry := normalizeEntry(MemoryEntry{
			ID:             "markdown:" + node.ID,
			Content:        strings.TrimSpace(node.Content),
			Type:           MemoryTypeKnowledge,
			Timestamp:      markdownNodeTimestamp(node),
			Importance:     node.Importance,
			RelatedTo:      markdownNodeSourceIDs(node),
			Source:         "markdown",
			Summary:        strings.TrimSpace(node.Summary),
			Anchors:        cloneAnchors(node.Anchors),
			Confidence:     node.Confidence,
			LastAccessedAt: node.LastSeenAt.UTC(),
			EmbeddingID:    node.EmbeddingID,
			Metadata: map[string]any{
				"layer":      "markdown",
				"session_id": strings.TrimSpace(node.SessionID),
				"tags":       append([]string(nil), node.Tags...),
				"node_id":    node.ID,
				"source_ids": markdownNodeSourceIDs(node),
			},
		})
		if !entryMatchesQuery(entry, query) {
			continue
		}
		entries = append(entries, entry)
	}
	if query.Limit > 0 && len(entries) > query.Limit {
		entries = entries[:query.Limit]
	}
	return entries, nil
}

func compactRecallLine(entry MemoryEntry) string {
	text := strings.TrimSpace(entry.Summary)
	if text == "" {
		text = strings.TrimSpace(entry.Content)
	}
	if text == "" {
		return ""
	}
	return summarizeLine(text, 180)
}

func markdownNodeTimestamp(node MarkdownNode) time.Time {
	if !node.LastSeenAt.IsZero() {
		return node.LastSeenAt.UTC()
	}
	return node.CreatedAt.UTC()
}

func markdownNodeSourceIDs(node MarkdownNode) []string {
	if len(node.SourceIDs) > 0 {
		return append([]string(nil), node.SourceIDs...)
	}
	if len(node.RelatedTo) > 0 {
		return append([]string(nil), node.RelatedTo...)
	}
	return nil
}
