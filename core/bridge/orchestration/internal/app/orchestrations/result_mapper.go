package orchestrations

import (
	sharedresult "ghost-os/bridge/orchestration/internal/shared/result"
	bridgeTasks "ghost-os/bridge/tasks"
)

type ResultMapper struct{}

type ResultMapCommand struct {
	GroupNode   bridgeTasks.OrchestrationNode
	MemberOrder []string
	Result      GroupResult
}

func (ResultMapper) GroupRecord(cmd ResultMapCommand) sharedresult.NodeRecord {
	return sharedresult.NodeRecord{
		NodeID:   cmd.GroupNode.ID,
		NodeType: cmd.GroupNode.Type,
		Status:   cmd.Result.Status,
		Input:    groupNodeResultInput(cmd.GroupNode, cmd.MemberOrder),
		Output:   groupNodeResultOutput(cmd.GroupNode, cmd.MemberOrder, cmd.Result),
		Preview:  cmd.Result.Preview,
		Error:    cmd.Result.Error,
	}
}

func groupNodeResultInput(
	node bridgeTasks.OrchestrationNode,
	memberOrder []string,
) map[string]any {
	return map[string]any{
		"title":          node.Group.Title,
		"shared_context": node.Group.SharedContext,
		"speaking_mode":  node.Group.SpeakingMode,
		"owner_agent_id": node.Group.OwnerAgentID,
		"max_rounds":     node.Group.MaxRounds,
		"member_order":   append([]string(nil), memberOrder...),
	}
}

func groupNodeResultOutput(
	node bridgeTasks.OrchestrationNode,
	memberOrder []string,
	result GroupResult,
) map[string]any {
	return map[string]any{
		"completed_rounds":   result.CompletedRounds,
		"speaking_mode":      node.Group.SpeakingMode,
		"owner_agent_id":     result.OwnerAgentID,
		"owner_session_id":   result.OwnerSessionID,
		"member_order":       append([]string(nil), memberOrder...),
		"member_session_ids": result.MemberSessions,
		"shared_transcript":  result.Transcript,
		"member_results":     result.MemberResults,
		"dispatch_results":   result.DispatchResults,
	}
}
