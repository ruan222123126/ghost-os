package orchestration

import (
	"fmt"

	"ghost-os/bridge/session"
)

const supportedSessionSidebarPartitionVersion = 1

func (s *bridgeService) executeSessionSidebarPartitionsGetAction(traceID string) (ServiceResult, error) {
	store, err := s.requireSessionStore()
	if err != nil {
		return ServiceResult{}, err
	}

	state, err := store.LoadSidebarPartitionState()
	if err != nil {
		logAction(traceID, "SESSION_PARTITIONS_GET", "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}

	logAction(traceID, "SESSION_PARTITIONS_GET", "success", nil)
	return serviceResultSuccess(buildSessionSidebarPartitionStatePayload(state)), nil
}

func (s *bridgeService) executeSessionSidebarPartitionsPutAction(
	req sessionSidebarPartitionPutRequest,
	traceID string,
) (ServiceResult, error) {
	store, err := s.requireSessionStore()
	if err != nil {
		return ServiceResult{}, err
	}
	if err := requireSessionSidebarPartitionVersion(req.Version); err != nil {
		return ServiceResult{}, err
	}

	state, err := store.SaveSidebarPartitionState(buildSessionSidebarPartitionStateInput(req))
	if err != nil {
		logAction(traceID, "SESSION_PARTITIONS_PUT", "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}

	logAction(traceID, "SESSION_PARTITIONS_PUT", "success", nil)
	return serviceResultSuccess(buildSessionSidebarPartitionStatePayload(state)), nil
}

func requireSessionSidebarPartitionVersion(version int) error {
	if version == supportedSessionSidebarPartitionVersion {
		return nil
	}
	return wrapServiceError(
		ServiceErrorInvalidInput,
		fmt.Errorf("session sidebar partition version must be %d", supportedSessionSidebarPartitionVersion),
	)
}

func buildSessionSidebarPartitionStatePayload(
	state session.SessionSidebarPartitionState,
) sessionSidebarPartitionState {
	return sessionSidebarPartitionState{
		Version:     supportedSessionSidebarPartitionVersion,
		Partitions:  buildSessionSidebarPartitionPayloads(state.Partitions),
		Assignments: cloneSessionSidebarAssignments(state.Assignments),
	}
}

func buildSessionSidebarPartitionPayloads(
	partitions []session.SessionSidebarPartition,
) []sessionSidebarPartition {
	payloads := make([]sessionSidebarPartition, 0, len(partitions))
	for _, partition := range partitions {
		payloads = append(payloads, sessionSidebarPartition{
			ID:   partition.ID,
			Name: partition.Name,
		})
	}
	return payloads
}

func buildSessionSidebarPartitionStateInput(
	req sessionSidebarPartitionPutRequest,
) session.SessionSidebarPartitionState {
	return session.SessionSidebarPartitionState{
		Version:     supportedSessionSidebarPartitionVersion,
		Partitions:  buildSessionSidebarPartitionInputs(req.Partitions),
		Assignments: cloneSessionSidebarAssignments(req.Assignments),
	}
}

func buildSessionSidebarPartitionInputs(
	partitions []sessionSidebarPartition,
) []session.SessionSidebarPartition {
	inputs := make([]session.SessionSidebarPartition, 0, len(partitions))
	for _, partition := range partitions {
		inputs = append(inputs, session.SessionSidebarPartition{
			ID:   partition.ID,
			Name: partition.Name,
		})
	}
	return inputs
}

func cloneSessionSidebarAssignments(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(source))
	for sessionID, partitionID := range source {
		cloned[sessionID] = partitionID
	}
	return cloned
}
