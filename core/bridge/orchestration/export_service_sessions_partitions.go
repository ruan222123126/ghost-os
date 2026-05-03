package orchestration

func (s *Service) ExecuteSessionSidebarPartitionsGetAction(traceID string) (ServiceResult, error) {
	return s.inner.executeSessionSidebarPartitionsGetAction(traceID)
}

func (s *Service) ExecuteSessionSidebarPartitionsPutAction(
	req SessionSidebarPartitionPutRequest,
	traceID string,
) (ServiceResult, error) {
	return s.inner.executeSessionSidebarPartitionsPutAction(req, traceID)
}
