package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

const (
	workflowScreenControlAtomicMode      = "atomic"
	workflowScreenControlStepsKey        = "workflow_steps"
	workflowScreenControlActionKey       = "action"
	workflowScreenControlParamsKey       = "params"
	workflowScreenControlStepDelayMinMS  = 200
	workflowScreenControlStepDelayMaxMS  = 300
	workflowFindIconDataURLParam         = "workflow_template_data_url"
	workflowLegacyFindIconDataURLParam   = "template_data_url"
	workflowLegacyFindIconDataURLAlias   = "data_url"
	workflowFindIconDefaultTemplateName  = "workflow-find-icon-template.png"
	workflowFindIconLegacyDataURLMessage = "legacy find_icon data_url keys are not supported; use params.workflow_template_data_url"
)

type workflowScreenControlStep struct {
	Action     string
	ToolAction string
	Params     map[string]any
}

type workflowToolCallResult struct {
	outputText   string
	outputValue  any
	awaitingText string
}

func executeWorkflowToolNode(
	ctx context.Context,
	deps agentRuntimeDependencies,
	node WorkflowNode,
	traceID string,
) workflowNodeOutcome {
	if deps.registry == nil {
		return workflowNodeOutcome{err: fmt.Errorf("workflow tool runtime is not configured")}
	}
	toolName := strings.TrimSpace(node.Tool.ToolName)
	tool := deps.registry.Get(toolName)
	if tool == nil {
		return workflowNodeOutcome{err: fmt.Errorf("workflow tool %q is not available", toolName)}
	}
	preparedArgs, err := prepareWorkflowToolArguments(toolName, node.Tool.Arguments)
	if err != nil {
		return workflowNodeOutcome{err: err}
	}
	if toolName != screenControlToolID {
		return executeWorkflowPreparedToolCall(ctx, tool, node.ID, traceID, toolName, preparedArgs)
	}
	steps, baseArgs, err := parseWorkflowScreenControlSteps(preparedArgs)
	if err != nil {
		return workflowNodeOutcome{err: err}
	}
	if len(steps) == 0 {
		return executeWorkflowPreparedToolCall(ctx, tool, node.ID, traceID, toolName, preparedArgs)
	}
	return executeWorkflowScreenControlStepSequence(ctx, tool, node.ID, traceID, baseArgs, steps)
}

func executeWorkflowPreparedToolCall(
	ctx context.Context,
	tool tools.Tool,
	nodeID string,
	traceID string,
	toolName string,
	arguments map[string]any,
) workflowNodeOutcome {
	result, err := executeWorkflowToolCall(ctx, tool, nodeID, traceID, arguments)
	if err != nil {
		return workflowNodeOutcome{err: err}
	}
	if result.awaitingText != "" {
		prompt := truncateRunes(result.awaitingText, maxTaskResponsePreviewRunes)
		return workflowNodeOutcome{
			status:      taskRunStatusAwaitingHuman,
			preview:     prompt,
			outputText:  prompt,
			outputValue: prompt,
		}
	}
	return workflowNodeOutcome{
		status:      taskRunStatusSuccess,
		preview:     fmt.Sprintf("tool %s executed", toolName),
		outputText:  result.outputText,
		outputValue: result.outputValue,
	}
}

func executeWorkflowToolCall(
	ctx context.Context,
	tool tools.Tool,
	nodeID string,
	traceID string,
	arguments map[string]any,
) (workflowToolCallResult, error) {
	args, err := encodeWorkflowToolArguments(arguments)
	if err != nil {
		return workflowToolCallResult{}, err
	}
	output, err := tool.Execute(tools.WithToolCallID(ctx, workflowNodeToolCallID(nodeID)), args, traceID)
	if err != nil {
		return workflowToolCallResult{}, err
	}
	processed, meta, err := tools.PostProcessExecuteResult(tool, output, traceID)
	if err != nil {
		return workflowToolCallResult{}, err
	}
	result := workflowToolCallResult{
		outputText:  strings.TrimSpace(processed),
		outputValue: decodeWorkflowNodeOutput(strings.TrimSpace(processed)),
	}
	if meta.AwaitingHuman != nil {
		result.awaitingText = strings.TrimSpace(meta.AwaitingHuman.Prompt)
	}
	return result, nil
}

func parseWorkflowScreenControlSteps(arguments map[string]any) ([]workflowScreenControlStep, map[string]any, error) {
	rawSteps, hasWorkflowSteps := arguments[workflowScreenControlStepsKey]
	if !hasWorkflowSteps {
		return nil, nil, nil
	}
	if _, hasAction := arguments[workflowScreenControlActionKey]; hasAction {
		return nil, nil, fmt.Errorf("screen_control workflow_steps conflicts with action/params")
	}
	if _, hasParams := arguments[workflowScreenControlParamsKey]; hasParams {
		return nil, nil, fmt.Errorf("screen_control workflow_steps conflicts with action/params")
	}
	stepsRaw, ok := normalizeWorkflowScreenControlStepList(rawSteps)
	if !ok {
		return nil, nil, fmt.Errorf("screen_control workflow_steps must be an array")
	}
	if len(stepsRaw) == 0 {
		return nil, nil, fmt.Errorf("screen_control workflow_steps must contain at least 1 step")
	}
	steps := make([]workflowScreenControlStep, 0, len(stepsRaw))
	for index := range stepsRaw {
		rawStep, ok := stepsRaw[index].(map[string]any)
		if !ok {
			return nil, nil, fmt.Errorf("screen_control workflow_steps[%d] must be an object", index+1)
		}
		step, err := decodeWorkflowScreenControlStep(rawStep, index)
		if err != nil {
			return nil, nil, err
		}
		steps = append(steps, step)
	}
	baseArgs := cloneTaskActionParams(arguments)
	delete(baseArgs, workflowScreenControlStepsKey)
	return steps, baseArgs, nil
}

func normalizeWorkflowScreenControlStepList(input any) ([]any, bool) {
	if steps, ok := input.([]any); ok {
		return steps, true
	}
	if typed, ok := input.([]map[string]any); ok {
		steps := make([]any, 0, len(typed))
		for index := range typed {
			steps = append(steps, typed[index])
		}
		return steps, true
	}
	return nil, false
}

func decodeWorkflowScreenControlStep(rawStep map[string]any, index int) (workflowScreenControlStep, error) {
	action := strings.ToLower(strings.TrimSpace(workflowMapString(rawStep, workflowScreenControlActionKey)))
	if action == "" {
		return workflowScreenControlStep{}, fmt.Errorf("screen_control workflow_steps[%d] requires action", index+1)
	}
	toolAction, err := mapWorkflowScreenControlStepAction(action)
	if err != nil {
		return workflowScreenControlStep{}, fmt.Errorf("screen_control workflow_steps[%d]: %w", index+1, err)
	}
	params, err := workflowMapObject(rawStep, workflowScreenControlParamsKey)
	if err != nil {
		return workflowScreenControlStep{}, fmt.Errorf("screen_control workflow_steps[%d]: %w", index+1, err)
	}
	return workflowScreenControlStep{
		Action:     action,
		ToolAction: toolAction,
		Params:     params,
	}, nil
}

func mapWorkflowScreenControlStepAction(action string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "screenshot":
		return "screenshot", nil
	case "find_text":
		return "find_text", nil
	case "find_icon":
		return "find_icon", nil
	case "click":
		return "click_icon", nil
	default:
		return "", fmt.Errorf("unsupported action %q", action)
	}
}

func workflowMapObject(record map[string]any, key string) (map[string]any, error) {
	rawValue, exists := record[key]
	if !exists || rawValue == nil {
		return map[string]any{}, nil
	}
	value, ok := rawValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an object", key)
	}
	return cloneTaskActionParams(value), nil
}

func executeWorkflowScreenControlStepSequence(
	ctx context.Context,
	tool tools.Tool,
	nodeID string,
	traceID string,
	baseArgs map[string]any,
	steps []workflowScreenControlStep,
) workflowNodeOutcome {
	stepResults := make([]map[string]any, 0, len(steps))
	var finalOutput any
	var lastFindIconOutput any
	for index := range steps {
		step := steps[index]
		stepArgs, err := prepareWorkflowScreenControlStepArguments(baseArgs, step, lastFindIconOutput)
		if err != nil {
			return workflowNodeOutcome{err: fmt.Errorf("workflow screen_control step %d (%s) failed: %w", index+1, step.Action, err)}
		}
		result, err := executeWorkflowToolCall(ctx, tool, nodeID, traceID, stepArgs)
		if err != nil {
			return workflowNodeOutcome{err: fmt.Errorf("workflow screen_control step %d (%s) failed: %w", index+1, step.Action, err)}
		}
		finalOutput = result.outputValue
		stepResults = append(stepResults, map[string]any{
			"step_index":  index + 1,
			"action":      step.Action,
			"tool_action": step.ToolAction,
			"output":      result.outputValue,
		})
		if step.Action == "find_icon" {
			lastFindIconOutput = result.outputValue
		}
		if result.awaitingText != "" {
			return workflowNodeOutcome{
				status:      taskRunStatusAwaitingHuman,
				preview:     truncateRunes(result.awaitingText, maxTaskResponsePreviewRunes),
				outputText:  result.awaitingText,
				outputValue: buildWorkflowScreenControlStepSequenceOutput(stepResults, finalOutput),
			}
		}
		if err := waitWorkflowScreenControlStepTransition(ctx, index, len(steps)); err != nil {
			return workflowNodeOutcome{
				err: fmt.Errorf("workflow screen_control step %d (%s) transition delay interrupted: %w", index+1, step.Action, err),
			}
		}
	}
	outputValue := buildWorkflowScreenControlStepSequenceOutput(stepResults, finalOutput)
	return workflowNodeOutcome{
		status:      taskRunStatusSuccess,
		preview:     fmt.Sprintf("tool %s executed %d workflow steps", screenControlToolID, len(stepResults)),
		outputText:  encodeWorkflowNodeOutputText(outputValue),
		outputValue: outputValue,
	}
}

func waitWorkflowScreenControlStepTransition(ctx context.Context, index int, total int) error {
	if index >= total-1 {
		return nil
	}
	timer := time.NewTimer(randomWorkflowScreenControlStepDelay())
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func randomWorkflowScreenControlStepDelay() time.Duration {
	span := workflowScreenControlStepDelayMaxMS - workflowScreenControlStepDelayMinMS
	delayMS := workflowScreenControlStepDelayMinMS
	if span > 0 {
		delayMS += rand.IntN(span + 1)
	}
	return time.Duration(delayMS) * time.Millisecond
}

func prepareWorkflowScreenControlStepArguments(
	baseArgs map[string]any,
	step workflowScreenControlStep,
	lastFindIconOutput any,
) (map[string]any, error) {
	args := cloneTaskActionParams(baseArgs)
	if args == nil {
		args = map[string]any{}
	}
	args["mode"] = workflowScreenControlAtomicMode
	args[workflowScreenControlActionKey] = step.ToolAction
	params, err := resolveWorkflowScreenControlStepParams(step, lastFindIconOutput)
	if err != nil {
		return nil, err
	}
	if params == nil {
		params = map[string]any{}
	}
	args[workflowScreenControlParamsKey] = params
	return prepareWorkflowToolArguments(screenControlToolID, args)
}

func buildWorkflowScreenControlStepSequenceOutput(
	steps []map[string]any,
	finalOutput any,
) map[string]any {
	return map[string]any{
		"step_count":   len(steps),
		"steps":        steps,
		"final_output": finalOutput,
	}
}

func encodeWorkflowNodeOutputText(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
	return strings.TrimSpace(string(encoded))
}

func encodeWorkflowToolArguments(arguments map[string]any) (json.RawMessage, error) {
	if len(arguments) == 0 {
		return json.RawMessage(`{}`), nil
	}
	encoded, err := json.Marshal(arguments)
	if err != nil {
		return nil, fmt.Errorf("encode workflow tool arguments: %w", err)
	}
	return encoded, nil
}

func prepareWorkflowToolArguments(toolName string, arguments map[string]any) (map[string]any, error) {
	cloned := cloneTaskActionParams(arguments)
	if len(cloned) == 0 {
		return map[string]any{}, nil
	}
	if strings.TrimSpace(toolName) != screenControlToolID {
		return cloned, nil
	}
	return prepareWorkflowScreenControlFindIconArguments(cloned)
}

func prepareWorkflowScreenControlFindIconArguments(arguments map[string]any) (map[string]any, error) {
	if !isWorkflowScreenFindIconAction(arguments) {
		return arguments, nil
	}
	rawParams, ok := arguments["params"].(map[string]any)
	if !ok {
		return arguments, nil
	}
	params, changed, err := ensureWorkflowFindIconTemplatePath(rawParams)
	if err != nil {
		return nil, err
	}
	if !changed {
		return arguments, nil
	}
	arguments["params"] = params
	return arguments, nil
}

func isWorkflowScreenFindIconAction(arguments map[string]any) bool {
	action := strings.ToLower(workflowMapString(arguments, "action"))
	if action != "find_icon" && action != "click_icon" {
		return false
	}
	mode := strings.ToLower(workflowMapString(arguments, "mode"))
	return mode == "" || mode == workflowScreenControlAtomicMode
}

func ensureWorkflowFindIconTemplatePath(params map[string]any) (map[string]any, bool, error) {
	cloned := cloneTaskActionParams(params)
	if err := rejectLegacyWorkflowFindIconDataURL(cloned); err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(workflowMapString(cloned, "template_path")) != "" {
		return cloned, false, nil
	}
	req, hasUpload, err := buildWorkflowFindIconUploadRequest(cloned)
	if err != nil {
		return nil, false, err
	}
	if !hasUpload {
		return cloned, false, nil
	}
	uploaded, err := executeFindIconTemplateUpload(req)
	if err != nil {
		return nil, false, err
	}
	cloned["template_path"] = uploaded.TemplatePath
	if strings.TrimSpace(workflowMapString(cloned, "template_name")) == "" {
		cloned["template_name"] = uploaded.TemplateName
	}
	removeWorkflowFindIconUploadFields(cloned)
	return cloned, true, nil
}

func buildWorkflowFindIconUploadRequest(params map[string]any) (findIconTemplateUploadRequest, bool, error) {
	dataURL := firstWorkflowNonEmptyString(params, workflowFindIconDataURLParam)
	if dataURL == "" {
		return findIconTemplateUploadRequest{}, false, nil
	}
	mimeType := firstWorkflowNonEmptyString(params, "template_mime_type", "mime_type")
	filename := firstWorkflowNonEmptyString(params, "template_filename", "filename", "template_name")
	if filename == "" {
		filename = workflowFindIconDefaultTemplateName
	}
	if mimeType == "" {
		mimeType = workflowFindIconDataURLMimeType(dataURL)
	}
	req := findIconTemplateUploadRequest{
		Filename: filename,
		MimeType: mimeType,
		DataURL:  dataURL,
	}
	if _, _, _, err := normalizeFindIconTemplateUpload(req); err != nil {
		return findIconTemplateUploadRequest{}, false, err
	}
	return req, true, nil
}

func rejectLegacyWorkflowFindIconDataURL(params map[string]any) error {
	if strings.TrimSpace(workflowMapString(params, workflowLegacyFindIconDataURLParam)) != "" {
		return fmt.Errorf(workflowFindIconLegacyDataURLMessage)
	}
	if strings.TrimSpace(workflowMapString(params, workflowLegacyFindIconDataURLAlias)) != "" {
		return fmt.Errorf(workflowFindIconLegacyDataURLMessage)
	}
	return nil
}

func workflowFindIconDataURLMimeType(dataURL string) string {
	if !strings.HasPrefix(dataURL, "data:") {
		return ""
	}
	prefix, _, found := strings.Cut(dataURL, ",")
	if !found {
		return ""
	}
	raw := strings.TrimPrefix(prefix, "data:")
	if strings.HasSuffix(raw, ";base64") {
		raw = strings.TrimSuffix(raw, ";base64")
	}
	return strings.ToLower(strings.TrimSpace(raw))
}

func removeWorkflowFindIconUploadFields(params map[string]any) {
	delete(params, workflowFindIconDataURLParam)
	delete(params, workflowLegacyFindIconDataURLParam)
	delete(params, "template_filename")
	delete(params, "template_mime_type")
	delete(params, workflowLegacyFindIconDataURLAlias)
	delete(params, "filename")
	delete(params, "mime_type")
}

func firstWorkflowNonEmptyString(record map[string]any, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(workflowMapString(record, key))
		if value != "" {
			return value
		}
	}
	return ""
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

func workflowNodeToolCallID(nodeID string) string {
	return "workflow-" + strings.TrimSpace(nodeID)
}

func decodeWorkflowNodeOutput(output string) any {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return ""
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return trimmed
	}
	return decoded
}

func executeWorkflowLLMNode(
	ctx context.Context,
	deps agentRuntimeDependencies,
	node WorkflowNode,
) workflowNodeOutcome {
	if deps.client == nil {
		return workflowNodeOutcome{err: fmt.Errorf("workflow llm runtime is not configured")}
	}
	response, err := deps.client.Complete(ctx, llm.CompletionRequest{
		Messages: workflowLLMMessages(*node.LLM),
	})
	if err != nil {
		return workflowNodeOutcome{err: err}
	}
	text := workflowResponseText(response)
	if text == "" {
		return workflowNodeOutcome{err: fmt.Errorf("workflow llm node %q returned empty response", node.ID)}
	}
	preview := truncateRunes(text, maxTaskResponsePreviewRunes)
	return workflowNodeOutcome{
		status:      taskRunStatusSuccess,
		preview:     preview,
		outputText:  text,
		outputValue: text,
	}
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

func (r workflowTaskRunner) executeAgentNode(ctx context.Context, node WorkflowNode) workflowNodeOutcome {
	result := r.adapter.runAgentAction(ctx, agentParams{
		Message:   node.Agent.Message,
		SessionID: "",
	}, cloneTaskRuntimeOverrides(node.Agent.RuntimeOverrides), r.traceID)
	if strings.TrimSpace(result.Error) != "" {
		return workflowNodeOutcome{err: fmt.Errorf("%s", result.Error)}
	}
	return workflowNodeOutcome{
		status:      result.Status,
		sessionID:   result.SessionIDOutput,
		preview:     result.ResponsePreview,
		outputText:  result.ResponsePreview,
		outputValue: result.ResponsePreview,
	}
}
