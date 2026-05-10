package workflows

import (
	"context"

	bridgeTasks "ghost-os/bridge/tasks"
	"ghost-os/bridge/tools"
)

func executeScreenControlStepSequence(
	ctx context.Context,
	tool tools.Tool,
	nodeID string,
	traceID string,
	baseArgs map[string]any,
	steps []ScreenControlStep,
	uploader TemplateUploader,
) NodeOutcome {
	state := screenSequenceState{stepResults: make([]map[string]any, 0, len(steps))}
	for index := range steps {
		outcome := executeScreenControlStep(ctx, stepCommand{
			tool: tool, nodeID: nodeID, traceID: traceID, baseArgs: baseArgs,
			step: steps[index], index: index, total: len(steps), state: &state,
			uploader: uploader,
		})
		if outcome.Err != nil || outcome.Status == bridgeTasks.RunStatusAwaitingHuman {
			return outcome
		}
	}
	return screenSequenceSuccess(state, len(steps))
}

type screenSequenceState struct {
	stepResults        []map[string]any
	finalOutput        any
	lastFindIconOutput any
}

type stepCommand struct {
	tool     tools.Tool
	nodeID   string
	traceID  string
	baseArgs map[string]any
	step     ScreenControlStep
	index    int
	total    int
	state    *screenSequenceState
	uploader TemplateUploader
}

func executeScreenControlStep(ctx context.Context, cmd stepCommand) NodeOutcome {
	stepArgs, err := PrepareScreenControlStepArguments(cmd.baseArgs, cmd.step, cmd.state.lastFindIconOutput, cmd.uploader)
	if err != nil {
		return screenStepError(cmd.index, cmd.step.Action, err)
	}
	result, err := executeToolCall(ctx, cmd.tool, cmd.nodeID, cmd.traceID, stepArgs)
	if err != nil {
		return screenStepError(cmd.index, cmd.step.Action, err)
	}
	cmd.state.record(cmd.step, cmd.index, result.outputValue)
	if result.awaitingText != "" {
		return screenStepAwaiting(result.awaitingText, cmd.state)
	}
	if err := waitScreenControlStepTransition(ctx, cmd.index, cmd.total); err != nil {
		return screenStepDelayError(cmd.index, cmd.step.Action, err)
	}
	return NodeOutcome{Status: bridgeTasks.RunStatusSuccess}
}

func (s *screenSequenceState) record(step ScreenControlStep, index int, output any) {
	s.finalOutput = output
	s.stepResults = append(s.stepResults, map[string]any{
		"step_index":  index + 1,
		"action":      step.Action,
		"tool_action": step.ToolAction,
		"output":      output,
	})
	if step.Action == "find_icon" {
		s.lastFindIconOutput = output
	}
}
