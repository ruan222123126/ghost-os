package orchestration

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	runtimeadapter "ghost-os/bridge/orchestration/internal/adapters/runtime"
	workflowadapter "ghost-os/bridge/orchestration/internal/adapters/workflow"
	appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"
	"ghost-os/bridge/tools"
)

const (
	workflowScreenControlAtomicMode            = appworkflows.ScreenControlAtomicMode
	workflowScreenControlStepsKey              = appworkflows.ScreenControlWorkflowStepsKey
	workflowScreenControlActionKey             = appworkflows.ScreenControlActionKey
	workflowScreenControlParamsKey             = appworkflows.ScreenControlParamsKey
	workflowScreenControlCoordinateRefKey      = appworkflows.ScreenControlCoordinateRefKey
	workflowScreenControlFindIconCoordinateRef = appworkflows.ScreenControlFindIconRef
	workflowScreenControlStepDelayMinMS        = appworkflows.ScreenControlStepDelayMinMS
	workflowScreenControlStepDelayMaxMS        = appworkflows.ScreenControlStepDelayMaxMS
	workflowFindIconDataURLParam               = appworkflows.FindIconDataURLParam
	workflowLegacyFindIconDataURLParam         = appworkflows.LegacyFindIconDataURLParam
	workflowLegacyFindIconDataURLAlias         = appworkflows.LegacyFindIconDataURLAlias
	workflowFindIconDefaultTemplateName        = appworkflows.FindIconDefaultTemplateName
	workflowFindIconLegacyDataURLMessage       = appworkflows.LegacyFindIconDataURLMessage
)

type workflowScreenControlStep struct {
	Action     string
	ToolAction string
	Params     map[string]any
}

type workflowNodeOutcome struct {
	status        string
	sessionID     string
	preview       string
	outputText    string
	outputValue   any
	inputSnapshot any
	err           error
}

func executeWorkflowToolNode(
	ctx context.Context,
	deps agentRuntimeDependencies,
	node WorkflowNode,
	traceID string,
) workflowNodeOutcome {
	outcome := appworkflows.ExecuteToolNode(
		ctx,
		runtimeadapter.ToWorkflowDependencies(deps),
		node,
		traceID,
		workflowadapter.TemplateUploader{},
	)
	return fromAppWorkflowOutcome(outcome)
}

func executeWorkflowPreparedToolCall(
	ctx context.Context,
	tool tools.Tool,
	nodeID string,
	traceID string,
	toolName string,
	arguments map[string]any,
) workflowNodeOutcome {
	outcome := appworkflows.ExecuteToolNode(ctx, appworkflows.RuntimeDependencies{
		Registry: singleWorkflowToolRegistry(toolName, tool),
	}, WorkflowNode{ID: nodeID, Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{
		ToolName:  toolName,
		Arguments: arguments,
	}}, traceID, workflowadapter.TemplateUploader{})
	return fromAppWorkflowOutcome(outcome)
}

func singleWorkflowToolRegistry(toolName string, tool tools.Tool) *tools.Registry {
	registry := tools.NewRegistry()
	if tool != nil {
		registry.Register(tool)
	}
	return registry
}

func parseWorkflowScreenControlSteps(arguments map[string]any) ([]workflowScreenControlStep, map[string]any, error) {
	steps, baseArgs, err := appworkflows.ParseScreenControlSteps(arguments)
	return fromAppScreenControlSteps(steps), baseArgs, err
}

func decodeWorkflowScreenControlStep(rawStep map[string]any, index int) (workflowScreenControlStep, error) {
	step, err := appworkflows.DecodeScreenControlStep(rawStep, index)
	return fromAppScreenControlStep(step), err
}

func mapWorkflowScreenControlStepAction(action string) (string, error) {
	return appworkflows.MapScreenControlStepAction(action)
}

func encodeWorkflowToolArguments(arguments map[string]any) (json.RawMessage, error) {
	return appworkflows.EncodeToolArguments(arguments)
}

func prepareWorkflowToolArguments(toolName string, arguments map[string]any) (map[string]any, error) {
	return appworkflows.PrepareToolArguments(toolName, arguments, workflowadapter.TemplateUploader{})
}

func prepareWorkflowScreenControlStepArguments(
	baseArgs map[string]any,
	step workflowScreenControlStep,
	lastFindIconOutput any,
) (map[string]any, error) {
	return appworkflows.PrepareScreenControlStepArguments(
		baseArgs,
		toAppScreenControlStep(step),
		lastFindIconOutput,
		workflowadapter.TemplateUploader{},
	)
}

func encodeWorkflowNodeOutputText(value any) string {
	return appworkflows.EncodeNodeOutputText(value)
}

func decodeWorkflowNodeOutput(output string) any {
	return appworkflows.DecodeNodeOutput(output)
}

func workflowMapString(record map[string]any, key string) string {
	raw, ok := record[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}

func randomWorkflowScreenControlStepDelay() time.Duration {
	span := workflowScreenControlStepDelayMaxMS - workflowScreenControlStepDelayMinMS
	delayMS := workflowScreenControlStepDelayMinMS
	if span > 0 {
		delayMS += rand.IntN(span + 1)
	}
	return time.Duration(delayMS) * time.Millisecond
}

func workflowLLMMessages(node WorkflowLLMNode) []llm.Message {
	messages := make([]llm.Message, 0, 2)
	if strings.TrimSpace(node.SystemPrompt) != "" {
		messages = append(messages, llm.Message{Role: llm.RoleSystem, Text: node.SystemPrompt})
	}
	messages = append(messages, llm.Message{Role: llm.RoleUser, Text: node.Prompt})
	return messages
}

func workflowResponseText(response *llm.CompletionResponse) string {
	if response == nil {
		return ""
	}
	if text := strings.TrimSpace(response.Message.Text); text != "" {
		return text
	}
	parts := make([]string, 0, len(response.Message.Content))
	for _, part := range response.Message.Content {
		if text := strings.TrimSpace(part.Text); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func fromAppWorkflowOutcome(outcome appworkflows.NodeOutcome) workflowNodeOutcome {
	return workflowNodeOutcome{
		status:        outcome.Status,
		sessionID:     outcome.SessionID,
		preview:       outcome.Preview,
		outputText:    outcome.OutputText,
		outputValue:   outcome.OutputValue,
		inputSnapshot: outcome.InputSnapshot,
		err:           outcome.Err,
	}
}

func toAppScreenControlStep(step workflowScreenControlStep) appworkflows.ScreenControlStep {
	return appworkflows.ScreenControlStep{
		Action:     step.Action,
		ToolAction: step.ToolAction,
		Params:     cloneTaskActionParams(step.Params),
	}
}

func fromAppScreenControlStep(step appworkflows.ScreenControlStep) workflowScreenControlStep {
	return workflowScreenControlStep{
		Action:     step.Action,
		ToolAction: step.ToolAction,
		Params:     cloneTaskActionParams(step.Params),
	}
}

func fromAppScreenControlSteps(steps []appworkflows.ScreenControlStep) []workflowScreenControlStep {
	if steps == nil {
		return nil
	}
	out := make([]workflowScreenControlStep, 0, len(steps))
	for _, step := range steps {
		out = append(out, fromAppScreenControlStep(step))
	}
	return out
}
