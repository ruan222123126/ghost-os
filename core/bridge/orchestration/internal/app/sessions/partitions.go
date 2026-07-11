package sessions

import (
	"fmt"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/session"
)

const SupportedSidebarPartitionVersion = 1

func RequireSidebarPartitionVersion(version int) error {
	if version == SupportedSidebarPartitionVersion {
		return nil
	}
	return unsupportedSidebarPartitionVersionError{}
}

func BuildSidebarPartitionStatePayload(
	state session.SessionSidebarPartitionState,
) api.SessionSidebarPartitionState {
	return api.SessionSidebarPartitionState{
		Version:     SupportedSidebarPartitionVersion,
		Partitions:  buildSidebarPartitionPayloads(state.Partitions),
		Assignments: cloneSidebarAssignments(state.Assignments),
	}
}

func BuildSidebarPartitionStateInput(
	req api.SessionSidebarPartitionPutRequest,
) session.SessionSidebarPartitionState {
	return session.SessionSidebarPartitionState{
		Version:     SupportedSidebarPartitionVersion,
		Partitions:  buildSidebarPartitionInputs(req.Partitions),
		Assignments: cloneSidebarAssignments(req.Assignments),
	}
}

func buildSidebarPartitionPayloads(
	partitions []session.SessionSidebarPartition,
) []api.SessionSidebarPartition {
	payloads := make([]api.SessionSidebarPartition, 0, len(partitions))
	for _, partition := range partitions {
		payloads = append(payloads, api.SessionSidebarPartition{
			ID:   partition.ID,
			Name: partition.Name,
		})
	}
	return payloads
}

func buildSidebarPartitionInputs(
	partitions []api.SessionSidebarPartition,
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

func cloneSidebarAssignments(source map[string]string) map[string]string {
	if len(source) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(source))
	for sessionID, partitionID := range source {
		cloned[sessionID] = partitionID
	}
	return cloned
}

type unsupportedSidebarPartitionVersionError struct{}

func (unsupportedSidebarPartitionVersionError) Error() string {
	return fmt.Sprintf("session sidebar partition version must be %d", SupportedSidebarPartitionVersion)
}

func (unsupportedSidebarPartitionVersionError) Is(target error) bool {
	return target == ErrUnsupportedSidebarPartitionVersion
}
