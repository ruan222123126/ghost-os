package orchestration

import (
	"context"
	"fmt"

	agentadapter "ghost-os/bridge/orchestration/internal/adapters/agent"
	apporchestrations "ghost-os/bridge/orchestration/internal/app/orchestrations"
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeTasks "ghost-os/bridge/tasks"
)

type orchestrationMemberActionInvoker struct {
	service *bridgeService
}

func (i orchestrationMemberActionInvoker) ExecuteAgentAction(
	ctx context.Context,
	req ports.AgentActionRequest,
) (ports.AgentActionPayload, error) {
	if i.service == nil {
		return ports.AgentActionPayload{}, fmt.Errorf("task executor service is not configured")
	}
	result, err := i.service.agentTurnService().Execute(
		ctx,
		agentParams{Message: req.Message, SessionID: req.SessionID},
		bridgeTasks.CloneTaskRuntimeOverrides(req.RuntimeOverrides),
		req.TraceID,
	)
	if err != nil {
		return ports.AgentActionPayload{}, err
	}
	return orchestrationAgentActionPayload(result.Payload), nil
}

func orchestrationAgentActionPayload(payload any) ports.AgentActionPayload {
	switch typed := payload.(type) {
	case agentResponse:
		return ports.AgentActionPayload{
			Kind:      ports.AgentActionPayloadSuccess,
			SessionID: typed.SessionID,
			Message:   typed.Message,
		}
	case askHumanAwaitingResponse:
		return ports.AgentActionPayload{
			Kind:      ports.AgentActionPayloadAwaiting,
			SessionID: typed.SessionID,
			Prompt:    typed.Prompt,
		}
	default:
		return ports.AgentActionPayload{
			Kind:     ports.AgentActionPayloadUnsupported,
			TypeName: fmt.Sprintf("%T", payload),
		}
	}
}

func (r orchestrationTaskRunner) memberRunner() ports.MemberAgentRunner {
	return apporchestrations.MemberRunner{
		Executor: agentadapter.MemberAgentRunner{
			Invoker: orchestrationMemberActionInvoker{service: r.adapter.service},
		},
	}
}
