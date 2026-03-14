// Session use cases for create/list/get/update/delete operations.

package orchestration

import (
	"net/http"
)

// executeSessionsListAction 汇总全部会话元数据，并映射为 API 返回结构。
func (s *bridgeService) executeSessionsListAction(traceID string) (any, int, error) {
	store, code, err := s.requireSessionStore()
	if err != nil {
		return nil, code, err
	}

	summaries, err := store.ListMetadata()
	if err != nil {
		logAction(traceID, "SESSIONS_LIST", "error", err)
		return nil, http.StatusInternalServerError, err
	}

	metadata := make([]sessionMetadata, 0, len(summaries))
	for _, summary := range summaries {
		metadata = append(metadata, buildSessionMetadataPayload(summary))
	}

	logAction(traceID, "SESSIONS_LIST", "success", nil)
	return metadata, http.StatusOK, nil
}

// executeSessionGetAction 读取并返回单会话详情（含完整消息与统计信息）。
func (s *bridgeService) executeSessionGetAction(params sessionIDParams, traceID string) (any, int, error) {
	store, code, err := s.requireSessionStore()
	if err != nil {
		return nil, code, err
	}

	id, code, err := requireSessionID(params.ID)
	if err != nil {
		return nil, code, err
	}

	sess, err := store.Load(id)
	if err != nil {
		logAction(traceID, "SESSION_GET", "error", err)
		return nil, mapSessionStorageError(err), err
	}

	logAction(traceID, "SESSION_GET", "success", nil)
	return buildSessionDetailPayload(sess), http.StatusOK, nil
}

// executeSessionDeleteAction 删除指定会话，并返回幂等友好的删除结果结构。
func (s *bridgeService) executeSessionDeleteAction(params sessionIDParams, traceID string) (any, int, error) {
	store, code, err := s.requireSessionStore()
	if err != nil {
		return nil, code, err
	}

	id, code, err := requireSessionID(params.ID)
	if err != nil {
		return nil, code, err
	}

	if err := store.Delete(id); err != nil {
		logAction(traceID, "SESSION_DELETE", "error", err)
		return nil, mapSessionStorageError(err), err
	}

	logAction(traceID, "SESSION_DELETE", "success", nil)
	return sessionDeleteResponse{
		ID:      id,
		Deleted: true,
	}, http.StatusOK, nil
}
