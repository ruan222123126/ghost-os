package workflows

import (
	"context"
	"fmt"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/app/workflows/screen"
	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	"ghost-os/bridge/orchestration/internal/trace/runcards"
	bridgeTasks "ghost-os/bridge/tasks"
	"ghost-os/bridge/tools"
)

const (
	ScreenControlToolID           = screen.ToolID
	ScreenControlCoordinateRefKey = screen.CoordinateRefKey
	ScreenControlFindIconRef      = screen.FindIconRef
	ScreenControlWorkflowStepsKey = screen.WorkflowStepsKey
	ScreenControlActionKey        = screen.ActionKey
	ScreenControlParamsKey        = screen.ParamsKey
	ScreenControlAtomicMode       = screen.AtomicMode
	ScreenControlStepDelayMinMS   = screen.StepDelayMinMS
	ScreenControlStepDelayMaxMS   = screen.StepDelayMaxMS
	FindIconDataURLParam          = screen.FindIconDataURLParam
	LegacyFindIconDataURLParam    = screen.LegacyDataURLParam
	LegacyFindIconDataURLAlias    = screen.LegacyDataURLAlias
	FindIconDefaultTemplateName   = screen.DefaultTemplateName
	LegacyFindIconDataURLMessage  = screen.LegacyDataURLMessage
)

type ScreenControlStep = screen.Step

type RuntimeDependencies struct {
	Client   llm.Completer
	Registry *tools.Registry
	Cleanup  func()
}

func (d RuntimeDependencies) Close() {
	if d.Cleanup != nil {
		d.Cleanup()
	}
}

type AgentRequest struct {
	Message          string
	RuntimeOverrides *bridgeTasks.TaskRuntimeOverrides
	TraceID          string
	NodeID           string
	NodeType         string
	BranchID         string
	Iteration        int
}

type AgentExecutor func(context.Context, AgentRequest) bridgeTasks.ExecutionResult

type TemplateUploadRequest = screen.TemplateUploadRequest
type TemplateUploadResult = screen.TemplateUploadResult
type TemplateUploader = screen.TemplateUploader

type ExecuteCommand struct {
	Plan             workflowdomain.Plan
	Runtime          RuntimeDependencies
	TraceID          string
	Agent            AgentExecutor
	Cards            CardObserver
	TemplateUploader TemplateUploader
}

type Runner struct{}

type NodeOutcome struct {
	Status        string
	SessionID     string
	Preview       string
	OutputText    string
	OutputValue   any
	InputSnapshot any
	Err           error
}

type runState struct {
	lastOutputText string
	nodeOutputs    map[string]string
	findIconOutput any
	loopIterations map[string]int
	iteration      int
}

type CardStartRequest struct {
	Kind      string
	Title     string
	NodeID    string
	NodeType  string
	BranchID  string
	Iteration int
	StartedAt time.Time
}

type CardFinishRequest struct {
	Status     string
	Preview    string
	Error      string
	FinalText  string
	FinishedAt time.Time
}

type CardHandle interface {
	Finish(ctx context.Context, req CardFinishRequest) error
}

type CardObserver interface {
	StartCard(ctx context.Context, req CardStartRequest) (CardHandle, error)
}

func NewRunCardObserverFromContext(ctx context.Context) CardObserver {
	recorder := runcards.RecorderFromContext(ctx)
	if recorder == nil {
		return nil
	}
	return runCardObserver{recorder: recorder}
}

type runCardObserver struct {
	recorder *runcards.Recorder
}

func (o runCardObserver) StartCard(ctx context.Context, req CardStartRequest) (CardHandle, error) {
	handle, err := o.recorder.StartCard(ctx, runcards.StartInput{
		Kind:      req.Kind,
		Title:     req.Title,
		NodeID:    req.NodeID,
		NodeType:  req.NodeType,
		BranchID:  req.BranchID,
		Iteration: req.Iteration,
		StartedAt: req.StartedAt,
	})
	if err != nil {
		return nil, err
	}
	return runCardHandle{inner: handle}, nil
}

type runCardHandle struct {
	inner *runcards.Handle
}

func (h runCardHandle) Finish(ctx context.Context, req CardFinishRequest) error {
	return h.inner.Finish(ctx, runcards.FinishInput{
		Status:     req.Status,
		Preview:    req.Preview,
		ErrorText:  req.Error,
		FinalText:  req.FinalText,
		FinishedAt: req.FinishedAt,
	})
}

type stepResult struct {
	nextNodeID string
	outcome    NodeOutcome
	executed   bool
	err        error
}

func ValidateTaskDefinition(task *bridgeTasks.ScheduledTask) error {
	if task.Workflow == nil {
		return fmt.Errorf("%w: workflow is required for workflow task", bridgeTasks.ErrInvalidTaskConfig)
	}
	task.Name = ""
	task.Orchestration = nil
	if task.Message != "" {
		return fmt.Errorf("%w: workflow task does not allow message", bridgeTasks.ErrInvalidTaskConfig)
	}
	if task.SessionID != "" {
		return fmt.Errorf("%w: workflow task does not allow session_id", bridgeTasks.ErrInvalidTaskConfig)
	}
	_, err := workflowdomain.PlanBuilder{}.Build(task.Workflow)
	return err
}
