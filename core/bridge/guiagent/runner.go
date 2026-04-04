package guiagent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

const defaultMaxSteps = 8

type Runner struct {
	model    llm.Completer
	operator Operator
	writer   *ArtifactWriter
	maxSteps int
}

func NewRunner(
	model llm.Completer,
	operator Operator,
	writer *ArtifactWriter,
	maxSteps int,
) *Runner {
	if maxSteps <= 0 {
		maxSteps = defaultMaxSteps
	}
	return &Runner{
		model:    model,
		operator: operator,
		writer:   writer,
		maxSteps: maxSteps,
	}
}

func (r *Runner) Continue(ctx context.Context, state State, answer string, traceID string) (Result, error) {
	return r.runLoop(ctx, state, answer, traceID)
}

func (r *Runner) runLoop(ctx context.Context, state State, answer string, traceID string) (Result, error) {
	if err := r.validateSetup(); err != nil {
		return Result{}, err
	}
	nextState, err := normalizeState(state)
	if err != nil {
		return Result{}, err
	}
	if answer != "" {
		nextState, err = applyHumanAnswer(nextState, answer)
		if err != nil {
			return Result{}, err
		}
	}
	for len(nextState.Steps) < r.maxSteps {
		result, updated, err := r.runStep(ctx, nextState, traceID)
		if err != nil {
			return Result{}, err
		}
		if result.Status != "" {
			return result, nil
		}
		nextState = updated
	}
	return Result{}, &RunError{Code: ErrorStepLimitExceeded, Message: fmt.Sprintf("exceeded max steps (%d)", r.maxSteps)}
}

func (r *Runner) runStep(ctx context.Context, state State, traceID string) (Result, State, error) {
	stepIndex := len(state.Steps) + 1
	before, err := r.observeAndPersist(ctx, state.Request, stepIndex, "before", traceID)
	if err != nil {
		return Result{}, State{}, err
	}
	decision, modelArtifacts, err := r.planStep(ctx, state, before, stepIndex, traceID)
	if err != nil {
		return Result{}, State{}, err
	}
	if isTerminalDecision(decision.Action.Type) {
		return r.handleTerminalDecision(state, before.Artifact, decision, modelArtifacts), state, nil
	}
	executeArtifact, err := r.executeAction(ctx, state.Request, before, decision.Action, stepIndex, traceID)
	if err != nil {
		return Result{}, State{}, err
	}
	after, err := r.observeAndPersist(ctx, state.Request, stepIndex, "after", traceID)
	if err != nil {
		return Result{}, State{}, err
	}
	verification, err := VerifyAction(before, after, decision.Action)
	if err != nil {
		return Result{}, State{}, err
	}
	step := StepRecord{
		Index:                 stepIndex,
		Thought:               strings.TrimSpace(decision.Thought),
		Action:                decision.Action,
		BeforeArtifact:        cloneArtifact(before.Artifact),
		AfterArtifact:         cloneArtifact(after.Artifact),
		ModelOutputArtifact:   cloneArtifact(modelArtifacts.raw),
		ParsedActionArtifact:  cloneArtifact(modelArtifacts.parsed),
		ExecuteResultArtifact: cloneArtifact(executeArtifact),
		Verification:          verification,
	}
	nextState := CloneState(state)
	nextState.Steps = append(nextState.Steps, step)
	return Result{}, nextState, nil
}

func (r *Runner) validateSetup() error {
	switch {
	case r == nil:
		return &RunError{Code: ErrorOperatorExecute, Message: "runner is nil"}
	case r.model == nil:
		return &RunError{Code: ErrorModelCompletionFailed, Message: "model completer is not configured"}
	case r.operator == nil:
		return &RunError{Code: ErrorOperatorExecute, Message: "operator is not configured"}
	case r.writer == nil:
		return &RunError{Code: ErrorOperatorExecute, Message: "artifact writer is not configured"}
	default:
		return nil
	}
}

func normalizeState(state State) (State, error) {
	if strings.TrimSpace(state.Request.Goal) == "" {
		return State{}, &RunError{Code: ErrorModelOutputParse, Message: "goal is required"}
	}
	if mode := strings.TrimSpace(state.Request.Mode); mode == "" {
		state.Request.Mode = "desktop"
	} else if mode != "desktop" {
		return State{}, &RunError{Code: ErrorModelOutputParse, Message: fmt.Sprintf("unsupported mode %q", mode)}
	}
	if strings.TrimSpace(state.RunID) != "" {
		return CloneState(state), nil
	}
	runID, err := NewRunID()
	if err != nil {
		return State{}, err
	}
	cloned := CloneState(state)
	cloned.RunID = runID
	return cloned, nil
}

func (r *Runner) observeAndPersist(
	ctx context.Context,
	request Request,
	stepIndex int,
	phase string,
	traceID string,
) (Observation, error) {
	observation, err := r.operator.Observe(ctx, request, traceID)
	if err != nil {
		return Observation{}, err
	}
	artifact, err := r.writer.WritePNG(
		stepIndex,
		phase,
		traceID,
		observation.ImageBytes,
		observation.ImageWidth,
		observation.ImageHeight,
	)
	if err != nil {
		return Observation{}, err
	}
	observation.Artifact = artifact
	return observation, nil
}

type modelArtifacts struct {
	raw    *StoredArtifact
	parsed *StoredArtifact
}

func (r *Runner) planStep(
	ctx context.Context,
	state State,
	before Observation,
	stepIndex int,
	traceID string,
) (Decision, modelArtifacts, error) {
	response, err := r.model.Complete(ctx, llm.CompletionRequest{Messages: BuildMessages(state.Request, state, before)})
	if err != nil {
		return Decision{}, modelArtifacts{}, &RunError{Code: ErrorModelCompletionFailed, Message: err.Error()}
	}
	if response.FinishReason == llm.FinishToolCalls {
		return Decision{}, modelArtifacts{}, &RunError{Code: ErrorModelOutputParse, Message: "computer_use model must not emit tool calls"}
	}
	rawOutput := strings.TrimSpace(response.Message.Text)
	rawArtifact, err := r.writer.WriteText(stepIndex, "model-output", traceID, rawOutput)
	if err != nil {
		return Decision{}, modelArtifacts{}, err
	}
	decision, err := ParseDecision(rawOutput)
	if err != nil {
		return Decision{}, modelArtifacts{}, err
	}
	parsedArtifact, err := r.writer.WriteJSON(stepIndex, "parsed-action", traceID, decision)
	if err != nil {
		return Decision{}, modelArtifacts{}, err
	}
	return decision, modelArtifacts{raw: rawArtifact, parsed: parsedArtifact}, nil
}

func (r *Runner) executeAction(
	ctx context.Context,
	request Request,
	observation Observation,
	action Action,
	stepIndex int,
	traceID string,
) (*StoredArtifact, error) {
	result := map[string]any{}
	switch action.Type {
	case ActionWait:
		duration := action.DurationMs
		if duration <= 0 {
			duration = 1000
		}
		time.Sleep(time.Duration(duration) * time.Millisecond)
		result["waited"] = true
		result["duration_ms"] = duration
	default:
		payload, err := r.operator.Execute(ctx, request, observation, action, traceID)
		if err != nil {
			return nil, err
		}
		result = payload
	}
	result["action"] = action.Type
	return r.writer.WriteJSON(stepIndex, "execute-result", traceID, result)
}

func isTerminalDecision(actionType string) bool {
	switch actionType {
	case ActionFinished, ActionCallUser:
		return true
	default:
		return false
	}
}
