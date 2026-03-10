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
	warm     warmQueryStore
	cold     coldQueryStore
	graph    graphQueryStore
	decision decisionQueryStore
	hygiene  hygieneQueryStore
	planner  intentQueryPlanner
	buckets  bucketQueryPlanner
	vector   vectorQueryStore
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

func NewQueryService(config MemoryConfig, warm *WarmMemory, cold *ColdMemory, graph *GraphService, decision *DecisionService, hygiene *HygieneService, planner *IntentPlanner, vector *VectorSidecar, truth *TruthReader, metrics *memoryCounters) *QueryService {
	return &QueryService{
		warm:                      warm,
		cold:                      cold,
		graph:                     graph,
		decision:                  decision,
		hygiene:                   hygiene,
		planner:                   planner,
		buckets:                   NewBucketPlanner(config, cold, metrics),
		vector:                    vector,
		truth:                     truth,
		autoRecallEnabled:         config.Warm.AutoRecallEnabled,
		autoRecallLimit:           config.Warm.AutoRecallLimit,
		shadowEnabled:             config.Recall.ShadowEnabled,
		truthReadEnabled:          config.Truth.ReadEnabled,
		hybridEnabled:             config.Recall.HybridRerankEnabled,
		rerankDebugEnabled:        config.Recall.RerankDebugEnabled,
		recallInjectMinConfidence: clamp01(config.Recall.RecallInjectMinConfidence),
		truthMinSupportRefs:       max(config.Truth.MinSupportRefs, 2),
		conflictPenalty:           maxFloat(config.Recall.ConflictPenalty, 0.12),
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
		AutoInject:        true,
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
	entries = s.filterEntriesByHygiene(entries, query)
	if len(entries) == 0 {
		return nil, nil
	}
	if s.hybridEnabled {
		filtered := make([]MemoryEntry, 0, len(entries))
		for _, entry := range entries {
			if metadataString(entry.Metadata, "memory_kind") == "hygiene_rule" {
				filtered = append(filtered, entry)
				continue
			}
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
	ruleLines := make([]string, 0, s.autoRecallLimit)
	contextLines := make([]string, 0, s.autoRecallLimit)
	for _, entry := range entries {
		if len(ruleLines)+len(contextLines) >= s.autoRecallLimit {
			break
		}
		if recallEntryVisible(entry, visible) {
			continue
		}
		if metadataString(entry.Metadata, "memory_kind") == "hygiene_rule" {
			if line := compactHygieneRuleLine(entry); line != "" {
				ruleLines = append(ruleLines, "- "+line)
			}
			continue
		}
		if line := compactRecallLine(entry); line != "" {
			contextLines = append(contextLines, "- "+line)
		}
	}
	advisoryLines := make([]string, 0, s.autoRecallLimit)
	if result.RecipeAdvisory != nil {
		for _, advisoryLine := range recipeContextLines(*result.RecipeAdvisory) {
			if advisoryLine == "" {
				continue
			}
			advisoryLines = append(advisoryLines, "- "+advisoryLine)
			if len(ruleLines)+len(advisoryLines)+len(contextLines) >= s.autoRecallLimit {
				break
			}
		}
	}
	if len(ruleLines) == 0 && len(contextLines) == 0 {
		return nil, nil
	}
	sections := make([]string, 0, 2+s.autoRecallLimit)
	if len(ruleLines) > 0 {
		sections = append(sections, "Rules to follow:")
		sections = append(sections, ruleLines...)
	}
	if len(contextLines) > 0 {
		if len(sections) > 0 {
			sections = append(sections, "")
		}
		sections = append(sections, "Recalled context:")
		sections = append(sections, advisoryLines...)
		sections = append(sections, contextLines...)
	} else if len(advisoryLines) > 0 {
		if len(sections) > 0 {
			sections = append(sections, "")
		}
		sections = append(sections, "Recalled context:")
		sections = append(sections, advisoryLines...)
	}

	return []llm.Message{{
		Role: llm.RoleSystem,
		Text: strings.Join(sections, "\n"),
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
	query = normalizeMemoryQueryHygiene(query)
	result, err := s.queryResultHybridWithScope(query, scope)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	result = s.decorateRecipeSelection(result, query, scope)
	result.Entries = stripEmbeddingIDs(result.Entries)
	return result, nil
}

func (s *QueryService) recordResultAccess(entries []MemoryEntry, now time.Time) {
	warmHits := make([]string, 0, len(entries))
	decisionMemoHits := make([]string, 0, len(entries))
	for _, entry := range entries {
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
}

func (s *QueryService) truthReadRequested(query MemoryQuery) bool {
	return s.truthReadEnabled || query.IncludeTruth || query.TruthDebug
}

func (s *QueryService) queryHygieneCards(query MemoryQuery, scope SessionScope) ([]MemoryEntry, error) {
	if s == nil || s.hygiene == nil || !s.hygiene.Enabled() {
		return nil, nil
	}
	return s.hygiene.QueryCards(query, scope)
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
	return s.queryMarkdownWithPlan(query, nil)
}

func (s *QueryService) queryMarkdownWithPlan(query MemoryQuery, plan *BucketPlan) ([]MemoryEntry, error) {
	if s == nil || s.cold == nil {
		return nil, nil
	}
	return s.cold.QueryMarkdown(query, plan, s.truth, s.hybridEnabled || s.truthReadRequested(query))
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
