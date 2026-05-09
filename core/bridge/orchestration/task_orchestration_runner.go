package orchestration

import (
	"context"

	apporchestrations "ghost-os/bridge/orchestration/internal/app/orchestrations"
	groupdomain "ghost-os/bridge/orchestration/internal/domain/group"
	bridgeTasks "ghost-os/bridge/tasks"
)

func (a taskExecutorAdapter) executeOrchestrationTask(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
) bridgeTasks.ExecutionResult {
	if _, err := buildOrchestrationExecutionPlan(task.Orchestration); err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	if a.service == nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "task executor service is not configured"}
	}
	return newOrchestrationTaskRunner(a).execute(ctx, task.Orchestration, traceID)
}

type orchestrationTaskRunner struct {
	adapter taskExecutorAdapter
}

func newOrchestrationTaskRunner(adapter taskExecutorAdapter) orchestrationTaskRunner {
	return orchestrationTaskRunner{adapter: adapter}
}

func (r orchestrationTaskRunner) groupExecutor() apporchestrations.ModeGroupExecutor {
	return apporchestrations.ModeGroupExecutor{
		Standard: r.standardGroupExecutor(),
		Owner:    r.ownerGroupExecutor(),
	}
}

func (r orchestrationTaskRunner) execute(
	ctx context.Context,
	definition *OrchestrationDefinition,
	traceID string,
) bridgeTasks.ExecutionResult {
	runner := apporchestrations.Runner{
		Planner: groupdomain.PlanBuilder{},
		Groups:  r.groupExecutor(),
		Mapper:  apporchestrations.ResultMapper{},
	}
	return runner.Execute(ctx, apporchestrations.ExecuteCommand{
		Definition: definition,
		TraceID:    traceID,
	})
}
