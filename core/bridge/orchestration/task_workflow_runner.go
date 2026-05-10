package orchestration

import (
	"context"
	"fmt"
	"strings"

	appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"
	bridgeTasks "ghost-os/bridge/tasks"
)

type workflowNodeOutcome struct {
	status        string
	sessionID     string
	preview       string
	outputText    string
	outputValue   any
	inputSnapshot any
	err           error
}

func (a taskExecutorAdapter) executeWorkflowTask(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
) bridgeTasks.ExecutionResult {
	plan, err := buildWorkflowExecutionPlan(task.Workflow)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	if a.service == nil && plan.NeedsService() {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "task executor service is not configured"}
	}
	if err := a.validateWorkflowRuntime(task.Workflow); err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	return newWorkflowTaskRunner(a, plan, traceID).execute(ctx)
}

func (a taskExecutorAdapter) validateWorkflowRuntime(definition *WorkflowDefinition) error {
	if a.service == nil {
		return nil
	}
	cfg, err := loadTaskRuntimeConfig(a.service.configStore)
	if err != nil {
		return err
	}
	if err := validateWorkflowTaskRuntime(definition, cfg); err != nil {
		return err
	}
	return validateWorkflowAgentRuntime(definition, a.service.configStore)
}

type workflowTaskRunner struct {
	adapter taskExecutorAdapter
	plan    workflowExecutionPlan
	traceID string
}

func newWorkflowTaskRunner(adapter taskExecutorAdapter, plan workflowExecutionPlan, traceID string) workflowTaskRunner {
	return workflowTaskRunner{adapter: adapter, plan: plan, traceID: strings.TrimSpace(traceID)}
}

func (r workflowTaskRunner) execute(ctx context.Context) bridgeTasks.ExecutionResult {
	deps, err := r.loadRuntimeDependencies()
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	if deps.cleanup != nil {
		defer deps.Close()
	}
	return appworkflows.Runner{}.Execute(ctx, r.executeCommand(deps))
}

func (r workflowTaskRunner) loadRuntimeDependencies() (agentRuntimeDependencies, error) {
	if !r.plan.NeedsRuntimeDependencies() {
		return agentRuntimeDependencies{}, nil
	}
	if r.adapter.service == nil {
		return agentRuntimeDependencies{}, fmt.Errorf("workflow runtime service is not configured")
	}
	return r.adapter.service.runtimeFactory.Build(r.adapter.service.configStore)
}

func (r workflowTaskRunner) executeCommand(deps agentRuntimeDependencies) appworkflows.ExecuteCommand {
	return appworkflows.ExecuteCommand{
		Plan:             r.plan,
		Runtime:          toWorkflowRuntimeDependencies(deps),
		TraceID:          r.traceID,
		Agent:            r.executeAgent,
		TemplateUploader: workflowFindIconTemplateUploader{},
	}
}

func (r workflowTaskRunner) executeAgent(ctx context.Context, req appworkflows.AgentRequest) bridgeTasks.ExecutionResult {
	return r.adapter.runAgentAction(ctx, agentParams{Message: req.Message, SessionID: ""}, req.RuntimeOverrides, req.TraceID)
}

func toWorkflowRuntimeDependencies(deps agentRuntimeDependencies) appworkflows.RuntimeDependencies {
	return appworkflows.RuntimeDependencies{
		Client:   deps.client,
		Registry: deps.registry,
		Cleanup:  deps.cleanup,
	}
}

type workflowFindIconTemplateUploader struct{}

func (workflowFindIconTemplateUploader) UploadFindIconTemplate(
	req appworkflows.TemplateUploadRequest,
) (appworkflows.TemplateUploadResult, error) {
	payload, err := executeFindIconTemplateUpload(findIconTemplateUploadRequest{
		Filename: req.Filename,
		MimeType: req.MimeType,
		DataURL:  req.DataURL,
	})
	if err != nil {
		return appworkflows.TemplateUploadResult{}, err
	}
	return appworkflows.TemplateUploadResult{
		TemplatePath: payload.TemplatePath,
		TemplateName: payload.TemplateName,
	}, nil
}
