package guiagent

import "strings"

const (
	ActionClick       = "click"
	ActionDoubleClick = "double_click"
	ActionRightClick  = "right_click"
	ActionTypeText    = "type"
	ActionHotkey      = "hotkey"
	ActionScroll      = "scroll"
	ActionDrag        = "drag"
	ActionWait        = "wait"
	ActionFinished    = "finished"
	ActionCallUser    = "call_user"
)

const (
	StatusSuccess      = "success"
	StatusAwaitingUser = "awaiting_human"
)

const (
	ErrorScreenshotFailed      = "screenshot_failed"
	ErrorActiveWindowInfo      = "active_window_info_failed"
	ErrorModelOutputParse      = "model_output_parse_error"
	ErrorInvalidTargetBox      = "invalid_target_box"
	ErrorOperatorExecute       = "operator_execute_error"
	ErrorVerificationFailed    = "verification_failed"
	ErrorStepLimitExceeded     = "step_limit_exceeded"
	ErrorModelCompletionFailed = "model_completion_failed"
)

type Request struct {
	Goal      string       `json:"goal"`
	Mode      string       `json:"mode"`
	Target    WindowTarget `json:"target,omitempty"`
	DisplayID *int         `json:"display_id,omitempty"`
}

type WindowTarget struct {
	WindowTitle string `json:"window_title,omitempty"`
	WindowClass string `json:"window_class,omitempty"`
}

type StoredArtifact struct {
	Type      string `json:"type"`
	Path      string `json:"path"`
	MimeType  string `json:"mime_type,omitempty"`
	SHA256    string `json:"sha256,omitempty"`
	Bytes     int    `json:"bytes,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
	StepIndex int    `json:"step_index,omitempty"`
}

type Observation struct {
	ImageWidth        int             `json:"image_width"`
	ImageHeight       int             `json:"image_height"`
	DisplayID         int             `json:"display_id"`
	ScaleX            float64         `json:"scale_x"`
	ScaleY            float64         `json:"scale_y"`
	OriginX           int             `json:"origin_x"`
	OriginY           int             `json:"origin_y"`
	ActiveWindowTitle string          `json:"active_window_title,omitempty"`
	ActiveWindowClass string          `json:"active_window_class,omitempty"`
	ImageBytes        []byte          `json:"-"`
	Artifact          *StoredArtifact `json:"artifact,omitempty"`
}

type NormalizedBox [4]float64

type ActionTarget struct {
	Box *NormalizedBox `json:"box,omitempty"`
}

type Action struct {
	Type        string        `json:"type"`
	Target      *ActionTarget `json:"target,omitempty"`
	Destination *ActionTarget `json:"destination,omitempty"`
	Text        string        `json:"text,omitempty"`
	Keys        []string      `json:"keys,omitempty"`
	DeltaX      int           `json:"delta_x,omitempty"`
	DeltaY      int           `json:"delta_y,omitempty"`
	DurationMs  int           `json:"duration_ms,omitempty"`
	Prompt      string        `json:"prompt,omitempty"`
	Reason      string        `json:"reason,omitempty"`
}

type Decision struct {
	Thought string `json:"thought"`
	Action  Action `json:"action"`
}

type VerificationResult struct {
	VisibleEffect     bool   `json:"visible_effect"`
	ScreenshotChanged bool   `json:"screenshot_changed"`
	WindowChanged     bool   `json:"window_changed"`
	TargetChanged     bool   `json:"target_changed"`
	Message           string `json:"message,omitempty"`
}

type StepRecord struct {
	Index                 int                `json:"index"`
	Thought               string             `json:"thought"`
	Action                Action             `json:"action"`
	UserAnswer            string             `json:"user_answer,omitempty"`
	BeforeArtifact        *StoredArtifact    `json:"before_artifact,omitempty"`
	AfterArtifact         *StoredArtifact    `json:"after_artifact,omitempty"`
	ModelOutputArtifact   *StoredArtifact    `json:"model_output_artifact,omitempty"`
	ParsedActionArtifact  *StoredArtifact    `json:"parsed_action_artifact,omitempty"`
	ExecuteResultArtifact *StoredArtifact    `json:"execute_result_artifact,omitempty"`
	Verification          VerificationResult `json:"verification"`
}

type State struct {
	RunID   string       `json:"run_id"`
	Request Request      `json:"request"`
	Steps   []StepRecord `json:"steps,omitempty"`
}

type AwaitingUser struct {
	Prompt string `json:"prompt"`
	Reason string `json:"reason,omitempty"`
}

type StepSummary struct {
	Index         int             `json:"index"`
	ActionType    string          `json:"action_type"`
	VisibleEffect bool            `json:"visible_effect"`
	Message       string          `json:"message,omitempty"`
	Before        *StoredArtifact `json:"before,omitempty"`
	After         *StoredArtifact `json:"after,omitempty"`
}

type Result struct {
	Status        string          `json:"status"`
	RunID         string          `json:"run_id"`
	Goal          string          `json:"goal"`
	Completed     bool            `json:"completed"`
	StepsTaken    int             `json:"steps_taken"`
	Summary       string          `json:"summary,omitempty"`
	AwaitingUser  *AwaitingUser   `json:"awaiting_user,omitempty"`
	LastArtifact  *StoredArtifact `json:"last_artifact,omitempty"`
	StepSummaries []StepSummary   `json:"step_summaries,omitempty"`
	State         *State          `json:"state,omitempty"`
}

type RunError struct {
	Code    string
	Message string
}

func (e *RunError) Error() string {
	if e == nil {
		return ""
	}
	code := strings.TrimSpace(e.Code)
	message := strings.TrimSpace(e.Message)
	if code == "" {
		return message
	}
	if message == "" {
		return code
	}
	return code + ": " + message
}

func CloneState(state State) State {
	cloned := state
	if len(state.Steps) == 0 {
		return cloned
	}
	cloned.Steps = make([]StepRecord, len(state.Steps))
	for index, step := range state.Steps {
		cloned.Steps[index] = cloneStep(step)
	}
	return cloned
}

func cloneStep(step StepRecord) StepRecord {
	cloned := step
	cloned.Action = cloneAction(step.Action)
	cloned.BeforeArtifact = cloneArtifact(step.BeforeArtifact)
	cloned.AfterArtifact = cloneArtifact(step.AfterArtifact)
	cloned.ModelOutputArtifact = cloneArtifact(step.ModelOutputArtifact)
	cloned.ParsedActionArtifact = cloneArtifact(step.ParsedActionArtifact)
	cloned.ExecuteResultArtifact = cloneArtifact(step.ExecuteResultArtifact)
	return cloned
}

func cloneAction(action Action) Action {
	cloned := action
	if len(action.Keys) > 0 {
		cloned.Keys = append([]string(nil), action.Keys...)
	}
	cloned.Target = cloneTarget(action.Target)
	cloned.Destination = cloneTarget(action.Destination)
	return cloned
}

func cloneTarget(target *ActionTarget) *ActionTarget {
	if target == nil {
		return nil
	}
	cloned := *target
	if target.Box != nil {
		box := *target.Box
		cloned.Box = &box
	}
	return &cloned
}

func cloneArtifact(artifact *StoredArtifact) *StoredArtifact {
	if artifact == nil {
		return nil
	}
	cloned := *artifact
	return &cloned
}
