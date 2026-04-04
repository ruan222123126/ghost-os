package guiagent

import (
	"encoding/json"
	"fmt"
	"strings"
)

func applyHumanAnswer(state State, answer string) (State, error) {
	if len(state.Steps) == 0 {
		return State{}, &RunError{Code: ErrorModelOutputParse, Message: "no pending call_user step to resume"}
	}
	index := len(state.Steps) - 1
	last := state.Steps[index]
	if last.Action.Type != ActionCallUser || last.UserAnswer != "" {
		return State{}, &RunError{Code: ErrorModelOutputParse, Message: "last step is not awaiting human input"}
	}
	nextState := CloneState(state)
	nextState.Steps[index].UserAnswer = strings.TrimSpace(answer)
	nextState.Steps[index].Verification = VerificationResult{
		VisibleEffect: true,
		Message:       "user answered call_user prompt",
	}
	return nextState, nil
}

func (r *Runner) handleTerminalDecision(
	state State,
	lastArtifact *StoredArtifact,
	decision Decision,
	artifacts modelArtifacts,
) Result {
	if decision.Action.Type == ActionFinished {
		return buildResult(StatusSuccess, state, lastArtifact, strings.TrimSpace(decision.Thought), nil)
	}
	nextState := CloneState(state)
	nextState.Steps = append(nextState.Steps, StepRecord{
		Index:                len(state.Steps) + 1,
		Thought:              strings.TrimSpace(decision.Thought),
		Action:               decision.Action,
		BeforeArtifact:       cloneArtifact(lastArtifact),
		ModelOutputArtifact:  cloneArtifact(artifacts.raw),
		ParsedActionArtifact: cloneArtifact(artifacts.parsed),
		Verification: VerificationResult{
			VisibleEffect: true,
			Message:       fmt.Sprintf("awaiting user input: %s", strings.TrimSpace(decision.Action.Prompt)),
		},
	})
	return buildResult(StatusAwaitingUser, nextState, lastArtifact, strings.TrimSpace(decision.Thought), &AwaitingUser{
		Prompt: strings.TrimSpace(decision.Action.Prompt),
		Reason: strings.TrimSpace(decision.Action.Reason),
	})
}

func buildResult(
	status string,
	state State,
	lastArtifact *StoredArtifact,
	summary string,
	awaiting *AwaitingUser,
) Result {
	return Result{
		Status:        status,
		RunID:         state.RunID,
		Goal:          strings.TrimSpace(state.Request.Goal),
		Completed:     status == StatusSuccess,
		StepsTaken:    len(state.Steps),
		Summary:       summary,
		AwaitingUser:  awaiting,
		LastArtifact:  cloneArtifact(lastArtifact),
		StepSummaries: summarizeSteps(state.Steps),
		State:         &state,
	}
}

func summarizeSteps(steps []StepRecord) []StepSummary {
	if len(steps) == 0 {
		return nil
	}
	out := make([]StepSummary, 0, len(steps))
	for _, step := range steps {
		out = append(out, StepSummary{
			Index:         step.Index,
			ActionType:    step.Action.Type,
			VisibleEffect: step.Verification.VisibleEffect,
			Message:       step.Verification.Message,
			Before:        cloneArtifact(step.BeforeArtifact),
			After:         cloneArtifact(step.AfterArtifact),
		})
	}
	return out
}

func EncodeResult(result Result) (string, error) {
	buf, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
