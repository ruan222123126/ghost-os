package memory

import (
	"fmt"
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
	planner  *IntentPlanner
	vector   *VectorSidecar
	truth    *TruthReader

	autoRecallEnabled         bool
	autoRecallLimit           int
	shadowEnabled             bool
	truthReadEnabled          bool
	hybridEnabled             bool
	rerankDebugEnabled        bool
	recallInjectMinConfidence float64
	truthMinSupportRefs       int
	conflictPenalty           float64
	scoring                   memoryScoringConfig
	metrics                   *memoryCounters
}

func NewQueryService(config MemoryConfig, warm *WarmMemory, cold *ColdMemory, graph *GraphService, decision *DecisionService, planner *IntentPlanner, vector *VectorSidecar, truth *TruthReader, metrics *memoryCounters) *QueryService {
	return &QueryService{
		warm:                      warm,
		cold:                      cold,
		graph:                     graph,
		decision:                  decision,
		planner:                   planner,
		vector:                    vector,
		truth:                     truth,
		autoRecallEnabled:         config.AutoRecallEnabled,
		autoRecallLimit:           config.AutoRecallLimit,
		shadowEnabled:             config.ShadowRecallEnabled,
		truthReadEnabled:          config.TruthReadEnabled,
		hybridEnabled:             config.HybridRerankEnabled,
		rerankDebugEnabled:        config.RerankDebugEnabled,
		recallInjectMinConfidence: clamp01(config.RecallInjectMinConfidence),
		truthMinSupportRefs:       max(config.TruthMinSupportRefs, 2),
		conflictPenalty:           maxFloat(config.ConflictPenalty, 0.12),
		scoring:                   newMemoryScoringConfig(config),
		metrics:                   metrics,
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
	if s.hybridEnabled {
		filtered := make([]MemoryEntry, 0, len(entries))
		for _, entry := range entries {
			if entry.Confidence < s.recallInjectMinConfidence {
				if s.metrics != nil {
					s.metrics.lowConfidenceFiltered.Add(1)
					s.metrics.contextInjectionFiltered.Add(1)
				}
				continue
			}
			status := normalizeTruthStatus(entry.TruthStatus)
			if status == truthStatusConflicted || status == truthStatusCandidate {
				if s.metrics != nil {
					s.metrics.contextInjectionFiltered.Add(1)
				}
				continue
			}
			filtered = append(filtered, entry)
		}
		entries = filtered
		if len(entries) == 0 {
			return nil, nil
		}
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
	if s.hybridEnabled && s.truth != nil && s.truth.Enabled() {
		return s.queryResultHybridWithScope(query, scope)
	}
	result, err := s.queryResultWeek2WithScope(query, scope)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	if !s.truthReadRequested(query) || s.truth == nil || !s.truth.Enabled() {
		return result, nil
	}
	plan := result.IntentPlan
	if plan == nil && s.planner != nil && s.planner.Enabled() {
		planned, err := s.planner.Plan(query, scope)
		if err == nil {
			planned = normalizeQueryIntentPlan(planned)
			if hasIntentPlan(planned) {
				plan = &planned
				result.IntentPlan = plan
			}
		}
	}
	matches := s.truth.Query(query, plan)
	result.TruthHits = s.truth.DebugHits(matches)
	return result, nil
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
