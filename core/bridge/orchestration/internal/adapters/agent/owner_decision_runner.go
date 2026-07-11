package agent

import (
	"context"
	"errors"
	"strings"

	"ghost-os/bridge/orchestration/internal/adapters/toolregistry"
	apporchestrations "ghost-os/bridge/orchestration/internal/app/orchestrations"
	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeTasks "ghost-os/bridge/tasks"
	"ghost-os/bridge/tools"
)

type OwnerDecisionRunner struct {
	Executor ports.OwnerDecisionTurnExecutor
}

func (r OwnerDecisionRunner) Decide(
	ctx context.Context,
	req ports.OwnerDecisionRequest,
) (group.DispatchCommand, string, error) {
	if req.OwnerNode.Agent == nil {
		return group.DispatchCommand{}, req.OwnerSessionID, errors.New("owner agent is not configured")
	}
	if r.Executor == nil {
		return group.DispatchCommand{}, req.OwnerSessionID, errors.New("owner dispatch runtime is not configured")
	}
	return r.Executor.RunOwnerDecisionTurn(ctx, ownerDecisionTurnRequest(req))
}

func ownerDecisionTurnRequest(req ports.OwnerDecisionRequest) ports.OwnerDecisionTurnRequest {
	return ports.OwnerDecisionTurnRequest{
		OwnerNode:        req.OwnerNode,
		GroupNode:        req.GroupNode,
		MemberNodes:      cloneNodeMap(req.MemberNodes),
		MemberOrder:      append([]string(nil), req.MemberOrder...),
		PublicTranscript: req.PublicTranscript.Clone(),
		LastDispatch:     cloneDispatchCommand(req.LastDispatch),
		RuntimeOverrides: bridgeTasks.CloneTaskRuntimeOverrides(req.OwnerNode.Agent.RuntimeOverrides),
		UserPrompt:       apporchestrations.BuildOwnerControlUserPrompt(req.Round, toolregistry.DispatchToolName),
		SessionID:        strings.TrimSpace(req.OwnerSessionID),
		Round:            req.Round,
		TraceID:          strings.TrimSpace(req.TraceID),
		Catalog:          ownerDispatchCatalog(req),
	}
}

func ownerDispatchCatalog(req ports.OwnerDecisionRequest) tools.ToolCatalog {
	registry := tools.NewRegistry()
	registry.Register(toolregistry.DispatchTool{
		GroupNode:   req.GroupNode,
		MemberOrder: append([]string(nil), req.MemberOrder...),
	})
	return tools.NewScopedCatalog(registry, []string{toolregistry.DispatchToolName})
}

func cloneNodeMap(
	input map[string]bridgeTasks.OrchestrationNode,
) map[string]bridgeTasks.OrchestrationNode {
	if len(input) == 0 {
		return map[string]bridgeTasks.OrchestrationNode{}
	}
	out := make(map[string]bridgeTasks.OrchestrationNode, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneDispatchCommand(input group.DispatchCommand) group.DispatchCommand {
	return group.DispatchCommand{
		Action:          input.Action,
		ParticipantIDs:  append([]string(nil), input.ParticipantIDs...),
		Order:           input.Order,
		Instruction:     input.Instruction,
		PrivateMessages: append([]group.PrivateMessage(nil), input.PrivateMessages...),
	}
}
