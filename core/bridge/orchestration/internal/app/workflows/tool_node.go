package workflows

import (
	"context"
	"fmt"
	"strings"

	"ghost-os/bridge/orchestration/internal/app/workflows/screen"
	sharedtext "ghost-os/bridge/orchestration/internal/shared/text"
	bridgeTasks "ghost-os/bridge/tasks"
	"ghost-os/bridge/tools"
)

type toolCallResult struct {
	outputText   string
	outputValue  any
	awaitingText string
}

func ExecuteToolNode(
	ctx context.Context,
	deps RuntimeDependencies,
	node bridgeTasks.WorkflowNode,
	traceID string,
	uploader TemplateUploader,
) NodeOutcome {
	if deps.Registry == nil {
		return NodeOutcome{Err: fmt.Errorf("workflow tool runtime is not configured")}
	}
	toolName := strings.TrimSpace(node.Tool.ToolName)
	tool := deps.Registry.Get(toolName)
	if tool == nil {
		return NodeOutcome{Err: fmt.Errorf("workflow tool %q is not available", toolName)}
	}
	return executeAvailableTool(ctx, toolCommand{tool: tool, node: node, traceID: traceID, toolName: toolName, uploader: uploader})
}

type toolCommand struct {
	tool     tools.Tool
	node     bridgeTasks.WorkflowNode
	traceID  string
	toolName string
	uploader TemplateUploader
}

func executeAvailableTool(ctx context.Context, cmd toolCommand) NodeOutcome {
	preparedArgs, err := screen.PrepareToolArguments(cmd.toolName, cmd.node.Tool.Arguments, cmd.uploader)
	if err != nil {
		return NodeOutcome{Err: err}
	}
	if cmd.toolName != ScreenControlToolID {
		return executePreparedToolCall(ctx, cmd.tool, cmd.node.ID, cmd.traceID, cmd.toolName, preparedArgs)
	}
	steps, baseArgs, err := screen.ParseSteps(preparedArgs)
	if err != nil {
		return NodeOutcome{Err: err}
	}
	if len(steps) == 0 {
		return executePreparedToolCall(ctx, cmd.tool, cmd.node.ID, cmd.traceID, cmd.toolName, preparedArgs)
	}
	return executeScreenControlStepSequence(ctx, cmd.tool, cmd.node.ID, cmd.traceID, baseArgs, steps, cmd.uploader)
}

func PrepareToolArguments(
	toolName string,
	arguments map[string]any,
	uploader TemplateUploader,
) (map[string]any, error) {
	return screen.PrepareToolArguments(toolName, arguments, uploader)
}

func ParseScreenControlSteps(arguments map[string]any) ([]ScreenControlStep, map[string]any, error) {
	return screen.ParseSteps(arguments)
}

func DecodeScreenControlStep(rawStep map[string]any, index int) (ScreenControlStep, error) {
	return screen.DecodeStep(rawStep, index)
}

func MapScreenControlStepAction(action string) (string, error) {
	return screen.MapStepAction(action)
}

func ResolveScreenControlStepParams(
	step ScreenControlStep,
	lastFindIconOutput any,
) (map[string]any, error) {
	return screen.ResolveStepParams(step, lastFindIconOutput)
}

func PrepareScreenControlStepArguments(
	baseArgs map[string]any,
	step ScreenControlStep,
	lastFindIconOutput any,
	uploader TemplateUploader,
) (map[string]any, error) {
	return screen.PrepareStepArguments(baseArgs, step, lastFindIconOutput, uploader)
}

func executePreparedToolCall(
	ctx context.Context,
	tool tools.Tool,
	nodeID string,
	traceID string,
	toolName string,
	arguments map[string]any,
) NodeOutcome {
	result, err := executeToolCall(ctx, tool, nodeID, traceID, arguments)
	if err != nil {
		return NodeOutcome{Err: err}
	}
	if result.awaitingText != "" {
		prompt := sharedtext.TruncateRunes(result.awaitingText, bridgeTasks.MaxResponsePreviewRunes)
		return NodeOutcome{Status: bridgeTasks.RunStatusAwaitingHuman, Preview: prompt, OutputText: prompt, OutputValue: prompt}
	}
	return NodeOutcome{
		Status:      bridgeTasks.RunStatusSuccess,
		Preview:     fmt.Sprintf("tool %s executed", toolName),
		OutputText:  result.outputText,
		OutputValue: result.outputValue,
	}
}

func executeToolCall(
	ctx context.Context,
	tool tools.Tool,
	nodeID string,
	traceID string,
	arguments map[string]any,
) (toolCallResult, error) {
	args, err := EncodeToolArguments(arguments)
	if err != nil {
		return toolCallResult{}, err
	}
	output, err := tool.Execute(tools.WithToolCallID(ctx, nodeToolCallID(nodeID)), args, traceID)
	if err != nil {
		return toolCallResult{}, err
	}
	return processToolOutput(tool, output, traceID)
}

func processToolOutput(tool tools.Tool, output string, traceID string) (toolCallResult, error) {
	processed, meta, err := tools.PostProcessExecuteResult(tool, output, traceID)
	if err != nil {
		return toolCallResult{}, err
	}
	result := toolCallResult{
		outputText:  strings.TrimSpace(processed),
		outputValue: DecodeNodeOutput(strings.TrimSpace(processed)),
	}
	if meta.AwaitingHuman != nil {
		result.awaitingText = strings.TrimSpace(meta.AwaitingHuman.Prompt)
	}
	return result, nil
}
