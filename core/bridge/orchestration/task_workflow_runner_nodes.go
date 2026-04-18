package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

const (
	workflowScreenControlAtomicMode      = "atomic"
	workflowFindIconDataURLParam         = "workflow_template_data_url"
	workflowLegacyFindIconDataURLParam   = "template_data_url"
	workflowLegacyFindIconDataURLAlias   = "data_url"
	workflowFindIconDefaultTemplateName  = "workflow-find-icon-template.png"
	workflowFindIconLegacyDataURLMessage = "legacy find_icon data_url keys are not supported; use params.workflow_template_data_url"
)

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
	args, err := encodeWorkflowToolArguments(preparedArgs)
	if err != nil {
		return workflowNodeOutcome{err: err}
	}
	output, err := tool.Execute(tools.WithToolCallID(ctx, workflowNodeToolCallID(node.ID)), args, traceID)
	if err != nil {
		return workflowNodeOutcome{err: err}
	}
	_, meta, err := tools.PostProcessExecuteResult(tool, output, traceID)
	if err != nil {
		return workflowNodeOutcome{err: err}
	}
	if meta.AwaitingHuman != nil {
		prompt := truncateRunes(strings.TrimSpace(meta.AwaitingHuman.Prompt), maxTaskResponsePreviewRunes)
		return workflowNodeOutcome{
			status:      taskRunStatusAwaitingHuman,
			preview:     prompt,
			outputText:  prompt,
			outputValue: prompt,
		}
	}
	outputText := strings.TrimSpace(output)
	return workflowNodeOutcome{
		status:      taskRunStatusSuccess,
		preview:     fmt.Sprintf("tool %s executed", toolName),
		outputText:  outputText,
		outputValue: decodeWorkflowNodeOutput(outputText),
	}
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
	}, nil, r.traceID)
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
