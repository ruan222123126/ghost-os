package workflows

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	sharedtext "ghost-os/bridge/orchestration/internal/shared/text"
	bridgeTasks "ghost-os/bridge/tasks"
)

func screenStepError(index int, action string, err error) NodeOutcome {
	return NodeOutcome{Err: fmt.Errorf("workflow screen_control step %d (%s) failed: %w", index+1, action, err)}
}

func screenStepDelayError(index int, action string, err error) NodeOutcome {
	return NodeOutcome{Err: fmt.Errorf("workflow screen_control step %d (%s) transition delay interrupted: %w", index+1, action, err)}
}

func screenStepAwaiting(awaitingText string, state *screenSequenceState) NodeOutcome {
	preview := sharedtext.TruncateRunes(awaitingText, bridgeTasks.MaxResponsePreviewRunes)
	return NodeOutcome{
		Status:      bridgeTasks.RunStatusAwaitingHuman,
		Preview:     preview,
		OutputText:  awaitingText,
		OutputValue: buildScreenControlStepSequenceOutput(state),
	}
}

func screenSequenceSuccess(state screenSequenceState, stepCount int) NodeOutcome {
	outputValue := buildScreenControlStepSequenceOutput(&state)
	return NodeOutcome{
		Status:      bridgeTasks.RunStatusSuccess,
		Preview:     fmt.Sprintf("tool %s executed %d workflow steps", ScreenControlToolID, stepCount),
		OutputText:  EncodeNodeOutputText(outputValue),
		OutputValue: outputValue,
	}
}

func buildScreenControlStepSequenceOutput(state *screenSequenceState) map[string]any {
	return map[string]any{
		"step_count":   len(state.stepResults),
		"steps":        state.stepResults,
		"final_output": state.finalOutput,
	}
}

func waitScreenControlStepTransition(ctx context.Context, index int, total int) error {
	if index >= total-1 {
		return nil
	}
	timer := time.NewTimer(randomScreenControlStepDelay())
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func randomScreenControlStepDelay() time.Duration {
	span := ScreenControlStepDelayMaxMS - ScreenControlStepDelayMinMS
	delayMS := ScreenControlStepDelayMinMS
	if span > 0 {
		delayMS += rand.IntN(span + 1)
	}
	return time.Duration(delayMS) * time.Millisecond
}
