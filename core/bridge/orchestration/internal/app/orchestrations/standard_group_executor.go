package orchestrations

import (
	"context"
	"strings"

	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	sharedtext "ghost-os/bridge/orchestration/internal/shared/text"
	bridgeTasks "ghost-os/bridge/tasks"
)

type StandardGroupExecutor struct {
	Dispatcher RoundDispatcher
}

type StandardGroupCommand struct {
	GroupNode          bridgeTasks.OrchestrationNode
	MemberNodes        map[string]bridgeTasks.OrchestrationNode
	MemberOrder        []string
	PreviousGroupID    string
	PreviousTranscript group.Transcript
	MemberSessions     map[string]string
	TraceID            string
}

type StandardGroupResult struct {
	Status          string
	Preview         string
	Error           string
	CompletedRounds int
	Transcript      group.Transcript
	MemberResults   []ports.MemberResult
	MemberSessions  map[string]string
}

func (e StandardGroupExecutor) Execute(
	ctx context.Context,
	cmd StandardGroupCommand,
) StandardGroupResult {
	transcript := group.Initial(cmd.GroupNode, cmd.PreviousGroupID, cmd.PreviousTranscript)
	memberSessions := CloneSessionIDs(cmd.MemberSessions)
	memberResults := make([]ports.MemberResult, 0, resultCapacity(cmd))
	completedRounds := 0
	for round := 1; round <= cmd.GroupNode.Group.MaxRounds; round++ {
		roundResult := e.dispatchRound(ctx, cmd, transcript, memberSessions, round)
		memberResults = append(memberResults, roundResult.MemberResults...)
		transcript = roundResult.Transcript
		applyMemberSessions(memberSessions, roundResult.MemberResults)
		if failure := SelectGroupFailure(roundResult.MemberResults); failure != nil {
			return failedStandardGroup(*failure, completedRounds, transcript, memberResults, memberSessions)
		}
		completedRounds = round
	}
	return successfulStandardGroup(completedRounds, transcript, memberResults, memberSessions)
}

func (e StandardGroupExecutor) dispatchRound(
	ctx context.Context,
	cmd StandardGroupCommand,
	transcript group.Transcript,
	memberSessions map[string]string,
	round int,
) RoundDispatchResult {
	return e.Dispatcher.Dispatch(ctx, RoundDispatchCommand{
		GroupNode:        cmd.GroupNode,
		MemberNodes:      cmd.MemberNodes,
		ParticipantIDs:   append([]string(nil), cmd.MemberOrder...),
		Transcript:       transcript,
		MemberSessionIDs: memberSessions,
		Round:            round,
		Order:            cmd.GroupNode.Group.SpeakingMode,
		TraceID:          cmd.TraceID,
	})
}

func SelectGroupFailure(results []ports.MemberResult) *ports.MemberResult {
	for index := range results {
		if results[index].Status == bridgeTasks.RunStatusAwaitingHuman {
			return &results[index]
		}
	}
	for index := range results {
		if isBusinessError(results[index]) {
			return &results[index]
		}
	}
	for index := range results {
		if results[index].Status != bridgeTasks.RunStatusSuccess {
			return &results[index]
		}
	}
	return nil
}

func isBusinessError(result ports.MemberResult) bool {
	return result.Status == bridgeTasks.RunStatusError &&
		result.Error != context.Canceled.Error()
}

func failedStandardGroup(
	failure ports.MemberResult,
	completedRounds int,
	transcript group.Transcript,
	memberResults []ports.MemberResult,
	memberSessions map[string]string,
) StandardGroupResult {
	errText := strings.TrimSpace(failure.Error)
	return StandardGroupResult{
		Status:          failure.Status,
		Preview:         sharedtext.TruncateRunes(failure.Preview, bridgeTasks.MaxResponsePreviewRunes),
		Error:           errText,
		CompletedRounds: completedRounds,
		Transcript:      transcript,
		MemberResults:   memberResults,
		MemberSessions:  memberSessions,
	}
}

func successfulStandardGroup(
	completedRounds int,
	transcript group.Transcript,
	memberResults []ports.MemberResult,
	memberSessions map[string]string,
) StandardGroupResult {
	return StandardGroupResult{
		Status:          bridgeTasks.RunStatusSuccess,
		Preview:         sharedtext.TruncateRunes(transcript.Format(), bridgeTasks.MaxResponsePreviewRunes),
		CompletedRounds: completedRounds,
		Transcript:      transcript,
		MemberResults:   memberResults,
		MemberSessions:  memberSessions,
	}
}

func applyMemberSessions(
	sessions map[string]string,
	results []ports.MemberResult,
) {
	for _, result := range results {
		if result.SessionID != "" {
			sessions[result.AgentID] = result.SessionID
		}
	}
}

func CloneSessionIDs(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func resultCapacity(cmd StandardGroupCommand) int {
	return len(cmd.MemberOrder) * cmd.GroupNode.Group.MaxRounds
}
