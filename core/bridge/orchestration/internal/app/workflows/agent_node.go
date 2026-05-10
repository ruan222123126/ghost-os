package workflows

import (
	"context"
	"fmt"
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
)

func executeAgentNode(
	ctx context.Context,
	executor AgentExecutor,
	node bridgeTasks.WorkflowNode,
	traceID string,
) NodeOutcome {
	if executor == nil {
		return NodeOutcome{Err: fmt.Errorf("workflow agent executor is not configured")}
	}
	result := executor(ctx, AgentRequest{
		Message:          node.Agent.Message,
		RuntimeOverrides: bridgeTasks.CloneTaskRuntimeOverrides(node.Agent.RuntimeOverrides),
		TraceID:          traceID,
	})
	if strings.TrimSpace(result.Error) != "" {
		return NodeOutcome{Err: fmt.Errorf("%s", result.Error)}
	}
	return NodeOutcome{
		Status:      result.Status,
		SessionID:   result.SessionIDOutput,
		Preview:     result.ResponsePreview,
		OutputText:  result.ResponsePreview,
		OutputValue: result.ResponsePreview,
	}
}
