// Memory use cases for multi-level query, archive, and promotion operations.

package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/memory"
	"ghost-os/bridge/session"
)

type memoryQueryResponse struct {
	Entries      []memory.MemoryEntry `json:"entries"`
	GraphHits    []memory.GraphHit    `json:"graph_hits,omitempty"`
	DecisionHits []memory.DecisionHit `json:"decision_hits,omitempty"`
}

type memoryDecisionQueryResponse struct {
	Entries      []memory.MemoryEntry `json:"entries,omitempty"`
	DecisionHits []memory.DecisionHit `json:"decision_hits,omitempty"`
}

// executeMemoryQueryAction 统一入口查询 L1/L2/Graph/L3 记忆，并返回可序列化结果。
func (s *bridgeService) executeMemoryQueryAction(_ context.Context, params memoryQueryParams, traceID string) (any, int, error) {
	if s.memoryManager == nil {
		return nil, http.StatusInternalServerError, errors.New("memory manager is not configured")
	}

	query, err := buildMemoryQuery(params)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	scope, err := s.buildMemoryQueryScope(query)
	if err != nil {
		logAction(traceID, busActionMemoryQuery, "error", err)
		return nil, http.StatusInternalServerError, err
	}

	logAction(traceID, busActionMemoryQuery, "running", nil)
	result, err := s.memoryManager.QueryResultWithScope(query, scope)
	if err != nil {
		logAction(traceID, busActionMemoryQuery, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	if err := markSessionsMemoryAccessed(s.sessionStore, result.Entries); err != nil {
		log.Printf(
			"trace_id=%s action=MEMORY_QUERY_MARK_ACCESSED status=error error=%v",
			strings.TrimSpace(traceID),
			err,
		)
	}
	logAction(traceID, busActionMemoryQuery, "success", nil)
	return memoryQueryResponse{Entries: result.Entries, GraphHits: result.GraphHits, DecisionHits: result.DecisionHits}, http.StatusOK, nil
}

func (s *bridgeService) buildMemoryQueryScope(query memory.MemoryQuery) (memory.SessionScope, error) {
	scope := memory.SessionScope{Environment: query.Environment}
	if s == nil || s.sessionStore == nil || query.Metadata == nil {
		return scope, nil
	}

	rawSessionID, ok := query.Metadata["session_id"]
	if !ok {
		return scope, nil
	}
	sessionID, ok := rawSessionID.(string)
	if !ok || strings.TrimSpace(sessionID) == "" {
		return scope, nil
	}

	sess, err := s.sessionStore.Load(strings.TrimSpace(sessionID))
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return scope, nil
		}
		return memory.SessionScope{}, err
	}
	scope.SessionID = strings.TrimSpace(sess.ID)
	scope.History = agent.NewHistoryFromMessages(sess.Messages)
	return scope, nil
}

// executeMemoryArchiveAction 将指定会话归档到冷存储，常用于长会话收敛。
func (s *bridgeService) executeMemoryArchiveAction(_ context.Context, params memoryArchiveParams, traceID string) (any, int, error) {
	if s.memoryManager == nil {
		return nil, http.StatusInternalServerError, errors.New("memory manager is not configured")
	}

	sessionID := strings.TrimSpace(params.SessionID)
	if sessionID == "" {
		return nil, http.StatusBadRequest, errors.New("session_id is required")
	}

	logAction(traceID, busActionMemoryArchive, "running", nil)
	if err := s.memoryManager.ArchiveToCold(sessionID); err != nil {
		logAction(traceID, busActionMemoryArchive, "error", err)
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil, http.StatusNotFound, err
		}
		return nil, http.StatusInternalServerError, err
	}
	if err := markSessionMemoryArchived(s.sessionStore, sessionID); err != nil {
		logAction(traceID, busActionMemoryArchive, "error", err)
		return nil, mapSessionStorageError(err), err
	}
	logAction(traceID, busActionMemoryArchive, "success", nil)
	return memoryArchiveResponse{
		SessionID: sessionID,
		Archived:  true,
	}, http.StatusOK, nil
}

// executeMemoryDecisionQueryAction 只查询 decision sidecar，用于 debug 打分与命中观察。
func (s *bridgeService) executeMemoryDecisionQueryAction(_ context.Context, params memoryDecisionQueryParams, traceID string) (any, int, error) {
	if s.memoryManager == nil {
		return nil, http.StatusInternalServerError, errors.New("memory manager is not configured")
	}
	query, err := buildMemoryDecisionQuery(params)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	logAction(traceID, busActionMemoryDecisionQuery, "running", nil)
	entries, hits, err := s.memoryManager.DecisionQuery(query)
	if err != nil {
		logAction(traceID, busActionMemoryDecisionQuery, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	logAction(traceID, busActionMemoryDecisionQuery, "success", nil)
	return memoryDecisionQueryResponse{Entries: entries, DecisionHits: hits}, http.StatusOK, nil
}

// executeMemoryDecisionStatsAction 返回 decision sidecar 的聚合计数。
func (s *bridgeService) executeMemoryDecisionStatsAction(_ context.Context, params memoryDecisionStatsParams, traceID string) (any, int, error) {
	if s.memoryManager == nil {
		return nil, http.StatusInternalServerError, errors.New("memory manager is not configured")
	}
	logAction(traceID, busActionMemoryDecisionStats, "running", nil)
	stats := s.memoryManager.DecisionStats(params.Namespace)
	logAction(traceID, busActionMemoryDecisionStats, "success", nil)
	return stats, http.StatusOK, nil
}

// executeMemoryDecisionRebuildAction 手动触发 decision memo/recipe rebuild。
func (s *bridgeService) executeMemoryDecisionRebuildAction(_ context.Context, params memoryDecisionRebuildParams, traceID string) (any, int, error) {
	if s.memoryManager == nil {
		return nil, http.StatusInternalServerError, errors.New("memory manager is not configured")
	}
	opts, err := buildMemoryDecisionRebuildOptions(params)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	logAction(traceID, busActionMemoryDecisionRebuild, "running", nil)
	stats, err := s.memoryManager.RebuildDecision(opts)
	if err != nil {
		logAction(traceID, busActionMemoryDecisionRebuild, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	logAction(traceID, busActionMemoryDecisionRebuild, "success", nil)
	return stats, http.StatusOK, nil
}

// buildMemoryQuery 把 API 参数转换为 MemoryQuery，并补齐 metadata/time_range 语义。
func buildMemoryQuery(params memoryQueryParams) (memory.MemoryQuery, error) {
	includeGraph := true
	if params.IncludeGraph != nil {
		includeGraph = *params.IncludeGraph
	}
	includeDecision := false
	if params.IncludeDecision != nil {
		includeDecision = *params.IncludeDecision
	}
	query := memory.MemoryQuery{
		Limit:             params.Limit,
		Keywords:          params.Keywords,
		Metadata:          map[string]any{},
		IncludeMarkdown:   params.IncludeMarkdown,
		IncludeGraph:      includeGraph || params.GraphDebug,
		IncludeDecision:   includeDecision || params.DecisionDebug,
		SemanticQuery:     strings.TrimSpace(params.SemanticQuery),
		GraphHops:         params.GraphHops,
		GraphPredicates:   append([]string(nil), params.GraphPredicates...),
		GraphDebug:        params.GraphDebug,
		DecisionDebug:     params.DecisionDebug,
		DecisionReuseOnly: params.DecisionReuseOnly,
		DecisionTypes:     append([]string(nil), params.DecisionTypes...),
		EnvironmentStrict: params.EnvironmentStrict,
		MinReuseScore:     params.MinReuseScore,
	}

	if params.Metadata != nil {
		for k, v := range params.Metadata {
			query.Metadata[k] = v
		}
	}

	if sessionID := strings.TrimSpace(params.SessionID); sessionID != "" {
		query.Metadata["session_id"] = sessionID
	}
	if len(query.Metadata) == 0 {
		query.Metadata = nil
	}

	if params.TimeRange != nil {
		timeRange, err := parseMemoryTimeRange(*params.TimeRange)
		if err != nil {
			return memory.MemoryQuery{}, err
		}
		query.TimeRange = timeRange
	}

	return query, nil
}

func buildMemoryDecisionQuery(params memoryDecisionQueryParams) (memory.MemoryQuery, error) {
	query := memory.MemoryQuery{
		Limit:             params.Limit,
		Keywords:          append([]string(nil), params.Keywords...),
		SemanticQuery:     strings.TrimSpace(params.SemanticQuery),
		IncludeDecision:   true,
		DecisionReuseOnly: params.DecisionReuseOnly,
		DecisionTypes:     append([]string(nil), params.DecisionTypes...),
		EnvironmentStrict: params.EnvironmentStrict,
		MinReuseScore:     params.MinReuseScore,
	}
	if namespace := strings.TrimSpace(params.Namespace); namespace != "" {
		query.Metadata = map[string]any{"namespace": namespace}
	}
	if params.TimeRange != nil {
		timeRange, err := parseMemoryTimeRange(*params.TimeRange)
		if err != nil {
			return memory.MemoryQuery{}, err
		}
		query.TimeRange = timeRange
	}
	return query, nil
}

func buildMemoryDecisionRebuildOptions(params memoryDecisionRebuildParams) (memory.DecisionRebuildOptions, error) {
	rebuildMemos := true
	if params.RebuildMemos != nil {
		rebuildMemos = *params.RebuildMemos
	}
	includeRecipes := false
	if params.IncludeRecipes != nil {
		includeRecipes = *params.IncludeRecipes
	}
	opts := memory.DecisionRebuildOptions{
		Namespace:      strings.TrimSpace(params.Namespace),
		MaxSessions:    params.MaxSessions,
		DryRun:         params.DryRun,
		IncludeRecipes: includeRecipes,
		RebuildMemos:   rebuildMemos,
		ResetNamespace: params.ResetNamespace,
	}
	if params.TimeRange != nil {
		timeRange, err := parseMemoryTimeRange(*params.TimeRange)
		if err != nil {
			return memory.DecisionRebuildOptions{}, err
		}
		opts.TimeRange = timeRange
	}
	return opts, nil
}

// parseMemoryTimeRange 解析 RFC3339 时间窗，并校验起止顺序合法性。
func parseMemoryTimeRange(raw memoryTimeRangeParams) (*memory.TimeRange, error) {
	startRaw := strings.TrimSpace(raw.Start)
	endRaw := strings.TrimSpace(raw.End)
	if startRaw == "" && endRaw == "" {
		return nil, nil
	}

	timeRange := &memory.TimeRange{}
	if startRaw != "" {
		start, err := time.Parse(time.RFC3339, startRaw)
		if err != nil {
			return nil, errors.New("time_range.start must be RFC3339 format")
		}
		timeRange.Start = start.UTC()
	}
	if endRaw != "" {
		end, err := time.Parse(time.RFC3339, endRaw)
		if err != nil {
			return nil, errors.New("time_range.end must be RFC3339 format")
		}
		timeRange.End = end.UTC()
	}
	if !timeRange.Start.IsZero() && !timeRange.End.IsZero() && timeRange.End.Before(timeRange.Start) {
		return nil, errors.New("time_range.end must be after time_range.start")
	}
	return timeRange, nil
}
