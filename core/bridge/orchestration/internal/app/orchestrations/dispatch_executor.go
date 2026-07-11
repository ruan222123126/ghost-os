package orchestrations

import (
	"context"
	"strings"

	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeTasks "ghost-os/bridge/tasks"
)

type DispatchExecutor struct {
	Dispatcher RoundDispatcher
}

type DispatchExecuteCommand struct {
	GroupNode        bridgeTasks.OrchestrationNode
	MemberNodes      map[string]bridgeTasks.OrchestrationNode
	PublicTranscript group.Transcript
	MemberSessions   map[string]string
	PrivateInboxes   map[string][]group.PrivateMessage
	Dispatch         group.DispatchCommand
	Round            int
	TraceID          string
}

type DispatchExecuteResult struct {
	Dispatch         DispatchResult
	PublicTranscript group.Transcript
	MemberSessions   map[string]string
	PrivateInboxes   map[string][]group.PrivateMessage
}

type PrivateDelivery struct {
	ParticipantID string `json:"participant_id"`
	Content       string `json:"content"`
}

type DispatchResult struct {
	Round             int                  `json:"round"`
	Action            string               `json:"action"`
	Order             string               `json:"order,omitempty"`
	Instruction       string               `json:"instruction,omitempty"`
	ParticipantIDs    []string             `json:"participant_ids,omitempty"`
	MemberResults     []ports.MemberResult `json:"member_results,omitempty"`
	PrivateTranscript group.Transcript     `json:"private_transcript,omitempty"`
	PrivateDeliveries []PrivateDelivery    `json:"private_deliveries,omitempty"`
	OwnerVisible      bool                 `json:"owner_visible"`
}

func (e DispatchExecutor) Execute(
	ctx context.Context,
	cmd DispatchExecuteCommand,
) DispatchExecuteResult {
	result := NewDispatchResult(cmd.Dispatch, cmd.Round, cmd.GroupNode.Group.OwnerAgentID)
	sessions := CloneSessionIDs(cmd.MemberSessions)
	inboxes := clonePrivateInboxes(cmd.PrivateInboxes)
	switch result.Action {
	case group.DispatchActionPublicOnce:
		return e.executePublic(ctx, dispatchRunCommand{parent: cmd, result: result, sessions: sessions, inboxes: inboxes})
	case group.DispatchActionPrivateOnce:
		return e.executePrivate(ctx, dispatchRunCommand{parent: cmd, result: result, sessions: sessions, inboxes: inboxes})
	case group.DispatchActionPrivateSend:
		return executePrivateSend(dispatchRunCommand{parent: cmd, result: result, sessions: sessions, inboxes: inboxes})
	default:
		return DispatchExecuteResult{Dispatch: result, PublicTranscript: cmd.PublicTranscript.Clone(), MemberSessions: sessions, PrivateInboxes: inboxes}
	}
}

func NewDispatchResult(
	dispatch group.DispatchCommand,
	round int,
	ownerAgentID string,
) DispatchResult {
	participants := append([]string(nil), dispatch.ParticipantIDs...)
	return DispatchResult{
		Round:             round,
		Action:            strings.TrimSpace(dispatch.Action),
		Order:             strings.TrimSpace(dispatch.Order),
		Instruction:       strings.TrimSpace(dispatch.Instruction),
		ParticipantIDs:    participants,
		PrivateDeliveries: privateDeliveries(dispatch.PrivateMessages),
		OwnerVisible:      containsString(participants, strings.TrimSpace(ownerAgentID)),
	}
}

type dispatchRunCommand struct {
	parent   DispatchExecuteCommand
	result   DispatchResult
	sessions map[string]string
	inboxes  map[string][]group.PrivateMessage
}

func (e DispatchExecutor) executePublic(ctx context.Context, cmd dispatchRunCommand) DispatchExecuteResult {
	roundResult := e.Dispatcher.Dispatch(ctx, RoundDispatchCommand{
		GroupNode:        cmd.parent.GroupNode,
		MemberNodes:      cmd.parent.MemberNodes,
		ParticipantIDs:   append([]string(nil), cmd.result.ParticipantIDs...),
		Transcript:       cmd.parent.PublicTranscript,
		MemberSessionIDs: cmd.sessions,
		PrivateInboxes:   clonePrivateInboxes(cmd.inboxes),
		Round:            cmd.parent.Round,
		Order:            cmd.result.Order,
		Instruction:      cmd.result.Instruction,
		TraceID:          cmd.parent.TraceID,
	})
	result := cmd.result
	result.MemberResults = roundResult.MemberResults
	applyMemberSessions(cmd.sessions, roundResult.MemberResults)
	return DispatchExecuteResult{Dispatch: result, PublicTranscript: roundResult.Transcript, MemberSessions: cmd.sessions, PrivateInboxes: cmd.inboxes}
}

func (e DispatchExecutor) executePrivate(ctx context.Context, cmd dispatchRunCommand) DispatchExecuteResult {
	roundResult := e.Dispatcher.Dispatch(ctx, RoundDispatchCommand{
		GroupNode:      cmd.parent.GroupNode,
		MemberNodes:    cmd.parent.MemberNodes,
		ParticipantIDs: append([]string(nil), cmd.result.ParticipantIDs...),
		Transcript:     cmd.parent.PublicTranscript,
		PrivateInboxes: clonePrivateInboxes(cmd.inboxes),
		Round:          cmd.parent.Round,
		Order:          group.SpeakingModeSequential,
		Instruction:    cmd.result.Instruction,
		Private:        true,
		TraceID:        cmd.parent.TraceID,
	})
	result := cmd.result
	result.MemberResults = roundResult.MemberResults
	result.PrivateTranscript = roundResult.Transcript.Clone()
	return DispatchExecuteResult{Dispatch: result, PublicTranscript: cmd.parent.PublicTranscript.Clone(), MemberSessions: cmd.sessions, PrivateInboxes: cmd.inboxes}
}

func executePrivateSend(cmd dispatchRunCommand) DispatchExecuteResult {
	nextInboxes := addPrivateMessages(cmd.inboxes, cmd.parent.Dispatch.PrivateMessages)
	return DispatchExecuteResult{
		Dispatch:         cmd.result,
		PublicTranscript: cmd.parent.PublicTranscript.Clone(),
		MemberSessions:   cmd.sessions,
		PrivateInboxes:   nextInboxes,
	}
}

func DispatchCommandFromResult(result DispatchResult) group.DispatchCommand {
	return group.DispatchCommand{
		Action:          strings.TrimSpace(result.Action),
		ParticipantIDs:  append([]string(nil), result.ParticipantIDs...),
		Order:           strings.TrimSpace(result.Order),
		Instruction:     strings.TrimSpace(result.Instruction),
		PrivateMessages: nil,
	}
}

func privateDeliveries(messages []group.PrivateMessage) []PrivateDelivery {
	if len(messages) == 0 {
		return nil
	}
	deliveries := make([]PrivateDelivery, 0, len(messages))
	for _, message := range messages {
		deliveries = append(deliveries, PrivateDelivery{
			ParticipantID: message.ParticipantID,
			Content:       strings.TrimSpace(message.Content),
		})
	}
	return deliveries
}

func clonePrivateInboxes(
	input map[string][]group.PrivateMessage,
) map[string][]group.PrivateMessage {
	if len(input) == 0 {
		return map[string][]group.PrivateMessage{}
	}
	out := make(map[string][]group.PrivateMessage, len(input))
	for key, value := range input {
		out[key] = append([]group.PrivateMessage(nil), value...)
	}
	return out
}

func addPrivateMessages(
	input map[string][]group.PrivateMessage,
	messages []group.PrivateMessage,
) map[string][]group.PrivateMessage {
	out := clonePrivateInboxes(input)
	for _, message := range messages {
		out[message.ParticipantID] = append(out[message.ParticipantID], message)
	}
	return out
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}
