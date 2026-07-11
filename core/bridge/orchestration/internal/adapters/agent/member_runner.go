package agent

import (
	"context"
	"fmt"
	"strings"

	appagentturn "ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeTasks "ghost-os/bridge/tasks"
)

type MemberAgentRunner struct {
	Invoker ports.AgentActionInvoker
}

func (r MemberAgentRunner) RunMemberTurn(
	ctx context.Context,
	req ports.MemberTurnRequest,
) (ports.MemberResult, error) {
	result := baseMemberResult(req)
	if r.Invoker == nil {
		return memberError(result, "orchestration member action invoker is not configured"), nil
	}
	payload, err := r.Invoker.ExecuteAgentAction(ctx, ports.AgentActionRequest{
		Message:          req.Message,
		SessionID:        strings.TrimSpace(req.SessionID),
		TraceID:          strings.TrimSpace(req.TraceID),
		AgentID:          strings.TrimSpace(req.AgentID),
		Title:            strings.TrimSpace(req.Title),
		Round:            req.Round,
		RuntimeOverrides: bridgeTasks.CloneTaskRuntimeOverrides(req.RuntimeOverrides),
	})
	if err != nil {
		result = memberError(result, err.Error())
		result.SessionID = appagentturn.SessionIDFromError(err)
		return result, nil
	}
	return memberResultFromPayload(result, payload), nil
}

func baseMemberResult(req ports.MemberTurnRequest) ports.MemberResult {
	return ports.MemberResult{
		Round:            req.Round,
		AgentID:          strings.TrimSpace(req.AgentID),
		Title:            strings.TrimSpace(req.Title),
		RuntimeOverrides: runtimeOverrideSnapshot(req.RuntimeOverrides),
	}
}

func memberResultFromPayload(
	result ports.MemberResult,
	payload ports.AgentActionPayload,
) ports.MemberResult {
	switch payload.Kind {
	case ports.AgentActionPayloadSuccess:
		result.Status = bridgeTasks.RunStatusSuccess
		result.SessionID = strings.TrimSpace(payload.SessionID)
		result.Content = payload.Message
		result.Preview = payload.Message
	case ports.AgentActionPayloadAwaiting:
		result.Status = bridgeTasks.RunStatusAwaitingHuman
		result.SessionID = strings.TrimSpace(payload.SessionID)
		result.Preview = payload.Prompt
		result.Error = payload.Prompt
	default:
		result = unsupportedMemberResult(result, payload.TypeName)
	}
	return result
}

func memberError(result ports.MemberResult, message string) ports.MemberResult {
	result.Status = bridgeTasks.RunStatusError
	result.Preview = strings.TrimSpace(message)
	result.Error = strings.TrimSpace(message)
	return result
}

func unsupportedMemberResult(result ports.MemberResult, typeName string) ports.MemberResult {
	if strings.TrimSpace(typeName) == "" {
		typeName = "<nil>"
	}
	result.Status = bridgeTasks.RunStatusError
	result.Preview = "unsupported orchestration member response"
	result.Error = fmt.Sprintf("unsupported orchestration member response %s", typeName)
	return result
}

func runtimeOverrideSnapshot(runtimeOverrides *bridgeTasks.TaskRuntimeOverrides) map[string]any {
	if runtimeOverrides == nil {
		return map[string]any{}
	}
	snapshot := map[string]any{
		"provider_name": strings.TrimSpace(runtimeOverrides.ProviderName),
		"model":         strings.TrimSpace(runtimeOverrides.Model),
		"system_prompt": strings.TrimSpace(runtimeOverrides.SystemPrompt),
		"preset_id":     strings.TrimSpace(runtimeOverrides.PresetID),
	}
	addOptionalRuntimeOverrides(snapshot, runtimeOverrides)
	return snapshot
}

func addOptionalRuntimeOverrides(
	snapshot map[string]any,
	runtimeOverrides *bridgeTasks.TaskRuntimeOverrides,
) {
	if runtimeOverrides.ToolAllowlistOnly != nil {
		snapshot["tool_allowlist_only"] = *runtimeOverrides.ToolAllowlistOnly
	}
	if runtimeOverrides.MaxTurns != nil {
		snapshot["max_turns"] = *runtimeOverrides.MaxTurns
	}
	if runtimeOverrides.ToolAllowlist != nil {
		snapshot["tool_allowlist"] = append([]string(nil), runtimeOverrides.ToolAllowlist...)
	}
}
