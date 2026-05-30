package workflows

import (
	"context"
	"time"

	"ghost-os/bridge/llm"
	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	bridgeTasks "ghost-os/bridge/tasks"
	"ghost-os/bridge/tools"
)

const (
	ScreenControlToolID           = "screen_control"
	ScreenControlCoordinateRefKey = "coordinate_ref"
	ScreenControlFindIconRef      = "${find_icon}"
	ScreenControlWorkflowStepsKey = "workflow_steps"
	ScreenControlActionKey        = "action"
	ScreenControlParamsKey        = "params"
	ScreenControlAtomicMode       = "atomic"
	ScreenControlStepDelayMinMS   = 200
	ScreenControlStepDelayMaxMS   = 300
	FindIconDataURLParam          = "workflow_template_data_url"
	LegacyFindIconDataURLParam    = "template_data_url"
	LegacyFindIconDataURLAlias    = "data_url"
	FindIconDefaultTemplateName   = "workflow-find-icon-template.png"
	LegacyFindIconDataURLMessage  = "legacy find_icon data_url keys are not supported; use params.workflow_template_data_url"
)

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

type TemplateUploadRequest struct {
	Filename string
	MimeType string
	DataURL  string
}

type TemplateUploadResult struct {
	TemplatePath string
	TemplateName string
}

type TemplateUploader interface {
	UploadFindIconTemplate(TemplateUploadRequest) (TemplateUploadResult, error)
}

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

type stepResult struct {
	nextNodeID string
	outcome    NodeOutcome
	executed   bool
	err        error
}
