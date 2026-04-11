// Runtime config snapshot/update use cases exposed to the transport layer.

package orchestration

import "net/http"

// executeConfigGetAction 返回当前运行态配置快照，不暴露敏感明文字段。
func (s *bridgeService) executeConfigGetAction(traceID string) (any, int, error) {
	logAction(traceID, busActionConfigGet, "success", nil)
	return configResponseFromSnapshot(s.configStore.Snapshot()), http.StatusOK, nil
}

// executeConfigUpdateAction 按请求局部更新运行态配置，并返回更新后快照。
func (s *bridgeService) executeConfigUpdateAction(req configUpdateRequest, traceID string) (any, int, error) {
	logAction(traceID, busActionConfigUpdate, "running", nil)
	if err := s.configStore.Update(configUpdateRequestToStoreRequest(req)); err != nil {
		logAction(traceID, busActionConfigUpdate, "error", err)
		return nil, http.StatusBadRequest, err
	}
	if s.rssHandler != nil {
		_ = s.reloadRSSInboxRuntime()
	}
	if err := s.BootstrapSystemTasks(); err != nil {
		logAction(traceID, busActionConfigUpdate, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	logAction(traceID, busActionConfigUpdate, "success", nil)
	return configResponseFromSnapshot(s.configStore.Snapshot()), http.StatusOK, nil
}
