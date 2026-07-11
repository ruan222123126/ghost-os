package taskrun

import (
	"context"
	"errors"
	"strings"

	appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"
	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	bridgeTasks "ghost-os/bridge/tasks"
)

type Planner interface {
	Build(definition *bridgeTasks.WorkflowDefinition) (workflowdomain.Plan, error)
}

type RuntimeLoader interface {
	LoadWorkflowRuntime(plan workflowdomain.Plan) (appworkflows.RuntimeDependencies, error)
}

type RuntimeValidator interface {
	ValidateWorkflowRuntime(definition *bridgeTasks.WorkflowDefinition) error
}

type Command struct {
	Definition       *bridgeTasks.WorkflowDefinition
	TraceID          string
	ServiceAvailable bool
	Planner          Planner
	Runtime          RuntimeLoader
	Validator        RuntimeValidator
	Agent            appworkflows.AgentExecutor
	Cards            appworkflows.CardObserver
	TemplateUploader appworkflows.TemplateUploader
}

type Runner struct{}

func (Runner) Execute(ctx context.Context, cmd Command) bridgeTasks.ExecutionResult {
	plan, err := buildPlan(cmd.Planner, cmd.Definition)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: err.Error()}
	}
	if !cmd.ServiceAvailable && plan.NeedsService() {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: "task executor service is not configured"}
	}
	if err := validateRuntime(cmd.Validator, cmd.Definition); err != nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: err.Error()}
	}
	runtime, err := loadRuntime(cmd.Runtime, plan)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: bridgeTasks.RunStatusError, Error: err.Error()}
	}
	if runtime.Cleanup != nil {
		defer runtime.Close()
	}
	return appworkflows.Runner{}.Execute(ctx, appworkflows.ExecuteCommand{
		Plan:             plan,
		Runtime:          runtime,
		TraceID:          strings.TrimSpace(cmd.TraceID),
		Agent:            cmd.Agent,
		Cards:            cmd.Cards,
		TemplateUploader: cmd.TemplateUploader,
	})
}

func buildPlan(
	planner Planner,
	definition *bridgeTasks.WorkflowDefinition,
) (workflowdomain.Plan, error) {
	if planner == nil {
		return workflowdomain.Plan{}, errors.New("workflow planner is not configured")
	}
	return planner.Build(definition)
}

func validateRuntime(
	validator RuntimeValidator,
	definition *bridgeTasks.WorkflowDefinition,
) error {
	if validator == nil {
		return nil
	}
	return validator.ValidateWorkflowRuntime(definition)
}

func loadRuntime(
	loader RuntimeLoader,
	plan workflowdomain.Plan,
) (appworkflows.RuntimeDependencies, error) {
	if !plan.NeedsRuntimeDependencies() {
		return appworkflows.RuntimeDependencies{}, nil
	}
	if loader == nil {
		return appworkflows.RuntimeDependencies{}, errors.New("workflow runtime service is not configured")
	}
	return loader.LoadWorkflowRuntime(plan)
}
