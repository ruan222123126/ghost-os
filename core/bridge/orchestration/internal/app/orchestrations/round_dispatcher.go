package orchestrations

import (
	"context"
	"errors"
	"sync"

	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeTasks "ghost-os/bridge/tasks"
)

type RoundDispatcher struct {
	Members ports.MemberAgentRunner
}

type RoundDispatchCommand struct {
	GroupNode        bridgeTasks.OrchestrationNode
	MemberNodes      map[string]bridgeTasks.OrchestrationNode
	ParticipantIDs   []string
	Transcript       group.Transcript
	MemberSessionIDs map[string]string
	PrivateInboxes   map[string][]group.PrivateMessage
	Round            int
	Order            string
	Instruction      string
	Private          bool
	TraceID          string
}

type RoundDispatchResult struct {
	MemberResults []ports.MemberResult
	Transcript    group.Transcript
}

func (d RoundDispatcher) Dispatch(
	ctx context.Context,
	cmd RoundDispatchCommand,
) RoundDispatchResult {
	if cmd.Order == group.SpeakingModeParallel {
		return d.dispatchParallel(ctx, cmd)
	}
	return d.dispatchSequential(ctx, cmd)
}

func (d RoundDispatcher) dispatchSequential(
	ctx context.Context,
	cmd RoundDispatchCommand,
) RoundDispatchResult {
	transcript := cmd.Transcript.Clone()
	results := make([]ports.MemberResult, 0, len(cmd.ParticipantIDs))
	for _, agentID := range cmd.ParticipantIDs {
		result := d.runMember(ctx, cmd, agentID, transcript.Format())
		results = append(results, result)
		if result.Status == bridgeTasks.RunStatusSuccess {
			transcript = transcript.AppendMember(result.Round, result.Title, result.AgentID, result.Content)
		}
		if result.Status != bridgeTasks.RunStatusSuccess {
			break
		}
	}
	return RoundDispatchResult{MemberResults: results, Transcript: transcript}
}

func (d RoundDispatcher) dispatchParallel(
	ctx context.Context,
	cmd RoundDispatchCommand,
) RoundDispatchResult {
	visibleTranscript := cmd.Transcript.Format()
	roundCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	results := make([]ports.MemberResult, len(cmd.ParticipantIDs))
	var wg sync.WaitGroup
	for index, agentID := range cmd.ParticipantIDs {
		wg.Add(1)
		go d.runParallelMember(roundCtx, cancel, &wg, parallelMemberCommand{
			round:   index,
			agentID: agentID,
			parent:  cmd,
			text:    visibleTranscript,
			results: results,
		})
	}
	wg.Wait()
	return RoundDispatchResult{
		MemberResults: results,
		Transcript:    appendSuccessfulMembers(cmd.Transcript, results),
	}
}

type parallelMemberCommand struct {
	round   int
	agentID string
	parent  RoundDispatchCommand
	text    string
	results []ports.MemberResult
}

func (d RoundDispatcher) runParallelMember(
	ctx context.Context,
	cancel context.CancelFunc,
	wg *sync.WaitGroup,
	cmd parallelMemberCommand,
) {
	defer wg.Done()
	cmd.results[cmd.round] = d.runMember(ctx, cmd.parent, cmd.agentID, cmd.text)
	if cmd.results[cmd.round].Status != bridgeTasks.RunStatusSuccess {
		cancel()
	}
}

func (d RoundDispatcher) runMember(
	ctx context.Context,
	cmd RoundDispatchCommand,
	agentID string,
	transcriptText string,
) ports.MemberResult {
	req := memberRunRequest(cmd, agentID, transcriptText)
	if d.Members == nil {
		return memberSetupError(req, errors.New("orchestration member runner is not configured"))
	}
	result, err := d.Members.RunMember(ctx, req)
	if err != nil {
		return memberSetupError(req, err)
	}
	return result
}

func memberRunRequest(
	cmd RoundDispatchCommand,
	agentID string,
	transcriptText string,
) ports.MemberRunRequest {
	return ports.MemberRunRequest{
		GroupNode:       cmd.GroupNode,
		MemberNode:      cmd.MemberNodes[agentID],
		TranscriptText:  transcriptText,
		Round:           cmd.Round,
		SessionID:       cmd.MemberSessionIDs[agentID],
		Instruction:     cmd.Instruction,
		PrivateMessages: append([]group.PrivateMessage(nil), cmd.PrivateInboxes[agentID]...),
		Private:         cmd.Private,
		TraceID:         cmd.TraceID,
	}
}

func appendSuccessfulMembers(
	transcript group.Transcript,
	results []ports.MemberResult,
) group.Transcript {
	out := transcript.Clone()
	for _, result := range results {
		if result.Status == bridgeTasks.RunStatusSuccess {
			out = out.AppendMember(result.Round, result.Title, result.AgentID, result.Content)
		}
	}
	return out
}
