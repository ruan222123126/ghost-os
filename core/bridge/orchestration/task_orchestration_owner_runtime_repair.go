package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

const ownerDispatchRepairTraceSuffix = "-repair"

func runOwnerDispatchWithRepair(
	ctx context.Context,
	req ownerDispatchRunRequest,
	runAgent *agent.Agent,
) (string, error) {
	output, runErr := runAgent.RunMessageWithTraceID(ctx, ownerUserMessage(req.request), req.request.TraceID)
	handoffErr, err := ownerDispatchHandoffFromRunErr(runErr)
	if err != nil {
		return "", err
	}
	if handoffErr != nil {
		return output, runErr
	}
	return repairOwnerDispatchRound(ctx, runAgent, req.request.TraceID, output)
}

func repairOwnerDispatchRound(
	ctx context.Context,
	runAgent *agent.Agent,
	traceID string,
	previousOutput string,
) (string, error) {
	repairMessage := llm.Message{
		Role: llm.RoleUser,
		Text: buildOwnerDispatchRepairPrompt(previousOutput),
	}
	return runAgent.RunMessageWithTraceID(ctx, repairMessage, ownerDispatchRepairTraceID(traceID))
}

func buildOwnerDispatchRepairPrompt(previousOutput string) string {
	output := strings.TrimSpace(previousOutput)
	if output == "" {
		output = "(empty response)"
	}
	return strings.TrimSpace(fmt.Sprintf(
		"Your previous reply did not advance the owner-led orchestration yet because it ended without calling `orchestration_dispatch`.\n\nPrevious reply:\n%s\n\nContinue this owner turn. You may keep working freely and use other visible tools if needed. When you are ready to advance the orchestration, call `orchestration_dispatch` once with one action: public_once, private_once, private_send, or end_group.",
		output,
	))
}

func ownerDispatchRepairTraceID(traceID string) string {
	trimmed := strings.TrimSpace(traceID)
	if trimmed == "" {
		return "owner-dispatch" + ownerDispatchRepairTraceSuffix
	}
	return trimmed + ownerDispatchRepairTraceSuffix
}

func ownerDispatchHandoffFromRunErr(runErr error) (*agent.ErrIterationHandoff, error) {
	if runErr == nil {
		return nil, nil
	}
	var handoffErr *agent.ErrIterationHandoff
	if errors.As(runErr, &handoffErr) {
		return handoffErr, nil
	}
	return nil, runErr
}
