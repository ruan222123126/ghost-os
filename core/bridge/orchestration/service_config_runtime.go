// Runtime config snapshot/update use cases exposed to the transport layer.

package orchestration

// executeConfigGetAction 返回当前可编辑配置快照，不暴露敏感明文字段。
func (s *bridgeService) executeConfigGetAction(traceID string) (ServiceResult, error) {
	response, err := s.publicConfigResponse()
	if err != nil {
		logAction(traceID, busActionConfigGet, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}
	logAction(traceID, busActionConfigGet, "success", nil)
	return serviceResultSuccess(response), nil
}

// executeConfigUpdateAction 按请求局部更新运行态配置，并返回更新后可编辑快照。
func (s *bridgeService) executeConfigUpdateAction(req configUpdateRequest, traceID string) (ServiceResult, error) {
	logAction(traceID, busActionConfigUpdate, "running", nil)
	if err := s.configStore.Update(configUpdateRequestToStoreRequest(req)); err != nil {
		logAction(traceID, busActionConfigUpdate, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInvalidInput, err)
	}
	if s.rssActionHandler() != nil {
		_ = s.reloadRSSInboxRuntime()
	}
	if err := s.BootstrapSystemTasks(); err != nil {
		logAction(traceID, busActionConfigUpdate, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}

	response, err := s.publicConfigResponse()
	if err != nil {
		logAction(traceID, busActionConfigUpdate, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}
	logAction(traceID, busActionConfigUpdate, "success", nil)
	return serviceResultSuccess(response), nil
}

func (s *bridgeService) publicConfigResponse() (configResponse, error) {
	snapshot, err := s.configStore.PublicSnapshot()
	if err != nil {
		return configResponse{}, err
	}
	return configResponseFromSnapshot(snapshot), nil
}
