// Agent turn use cases exposed to the transport layer.

package orchestration

import (
	"context"

	bridgeTasks "ghost-os/bridge/tasks"
)

// executeAgentAction 执行一次 Agent 回合，并处理“等待人工回答”的中断状态。
func (s *bridgeService) executeAgentAction(ctx context.Context, params agentParams, traceID string) (ServiceResult, error) {
	return s.executeAgentActionWithRuntimeOverrides(ctx, params, nil, traceID)
}

func (s *bridgeService) executeAgentActionWithRuntimeOverrides(
	ctx context.Context,
	params agentParams,
	runtimeOverrides *TaskRuntimeOverrides,
	traceID string,
) (ServiceResult, error) {
	return s.agentTurnService().Execute(
		ctx,
		params,
		bridgeTasks.CloneTaskRuntimeOverrides(runtimeOverrides),
		traceID,
	)
}

func (s *bridgeService) executeAgentStopAction(
	ctx context.Context,
	params agentStopParams,
	traceID string,
) (ServiceResult, error) {
	return s.agentTurnService().Stop(ctx, params, traceID)
}
