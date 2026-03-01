// Memory use cases for multi-level query, archive, and promotion operations.

package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"ghost-os/bridge/memory"
	"ghost-os/bridge/session"
)

type memoryQueryResponse struct {
	Entries []memory.MemoryEntry `json:"entries"`
}

// executeMemoryQueryAction 统一入口查询 L1/L2/L3 记忆，并返回可序列化结果。
func (s *bridgeService) executeMemoryQueryAction(_ context.Context, params memoryQueryParams, traceID string) (any, int, error) {
	if s.memoryManager == nil {
		return nil, http.StatusInternalServerError, errors.New("memory manager is not configured")
	}

	query, err := buildMemoryQuery(params)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	logAction(traceID, actionMemoryQuery, "running", nil)
	entries, err := s.memoryManager.Query(query)
	if err != nil {
		logAction(traceID, actionMemoryQuery, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	logAction(traceID, actionMemoryQuery, "success", nil)
	return memoryQueryResponse{Entries: entries}, http.StatusOK, nil
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

	logAction(traceID, actionMemoryArchive, "running", nil)
	if err := s.memoryManager.ArchiveToCold(sessionID); err != nil {
		logAction(traceID, actionMemoryArchive, "error", err)
		if errors.Is(err, session.ErrSessionNotFound) {
			return nil, http.StatusNotFound, err
		}
		return nil, http.StatusInternalServerError, err
	}
	logAction(traceID, actionMemoryArchive, "success", nil)
	return memoryArchiveResponse{
		SessionID: sessionID,
		Archived:  true,
	}, http.StatusOK, nil
}

// buildMemoryQuery 把 API 参数转换为 MemoryQuery，并补齐 metadata/time_range 语义。
func buildMemoryQuery(params memoryQueryParams) (memory.MemoryQuery, error) {
	query := memory.MemoryQuery{
		Limit:    params.Limit,
		Keywords: params.Keywords,
		Metadata: map[string]any{},
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
