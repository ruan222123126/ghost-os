package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	bridgeTasks "ghost-os/bridge/tasks"
	"ghost-os/bridge/tools"
)

type workflowNodeOutcome struct {
	status    string
	sessionID string
	preview   string
	err       error
}

func (a taskExecutorAdapter) executeWorkflowTask(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
) bridgeTasks.ExecutionResult {
	plan, err := buildWorkflowExecutionPlan(task.Workflow)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	if len(plan.executableNodes()) == 0 {
		return bridgeTasks.ExecutionResult{
			Status:          taskRunStatusSuccess,
			ResponsePreview: fmt.Sprintf("workflow completed: %s -> %s", plan.ordered[0].ID, plan.ordered[len(plan.ordered)-1].ID),
		}
	}
	if a.service == nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "task executor service is not configured"}
	}
	cfg, err := loadTaskRuntimeConfig(a.service.configStore)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	if err := validateWorkflowTaskRuntime(task.Workflow, cfg); err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	return newWorkflowTaskRunner(a, plan, traceID).execute(ctx)
}

type workflowTaskRunner struct {
	adapter taskExecutorAdapter
	plan    workflowExecutionPlan
	traceID string
}

func newWorkflowTaskRunner(
	adapter taskExecutorAdapter,
	plan workflowExecutionPlan,
	traceID string,
) workflowTaskRunner {
	return workflowTaskRunner{
		adapter: adapter,
		plan:    plan,
		traceID: strings.TrimSpace(traceID),
	}
}

func (r workflowTaskRunner) execute(ctx context.Context) bridgeTasks.ExecutionResult {
	deps, err := r.loadRuntimeDependencies()
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	if deps.cleanup != nil {
		defer deps.Close()
	}
	return r.executePlan(ctx, deps)
}

func (r workflowTaskRunner) loadRuntimeDependencies() (agentRuntimeDependencies, error) {
	if !workflowNeedsRuntimeDependencies(r.plan.executableNodes()) {
		return agentRuntimeDependencies{}, nil
	}
	return r.adapter.service.runtimeFactory.Build(r.adapter.service.configStore)
}

func workflowNeedsRuntimeDependencies(nodes []WorkflowNode) bool {
	for _, node := range nodes {
		if node.Type == workflowNodeTypeTool || node.Type == workflowNodeTypeLLM {
			return true
		}
	}
	return false
}

func (r workflowTaskRunner) executePlan(
	ctx context.Context,
	deps agentRuntimeDependencies,
) bridgeTasks.ExecutionResult {
	var lastSessionID string
	var lastNode WorkflowNode
	var lastPreview string
	for _, node := range r.plan.executableNodes() {
		outcome := r.executeNode(ctx, deps, node)
		if outcome.err != nil {
			return bridgeTasks.ExecutionResult{
				Status: taskRunStatusError,
				Error:  fmt.Sprintf("workflow node %s (%s) failed: %v", node.ID, node.Type, outcome.err),
			}
		}
		if strings.TrimSpace(outcome.sessionID) != "" {
			lastSessionID = strings.TrimSpace(outcome.sessionID)
		}
		if outcome.status == taskRunStatusAwaitingHuman {
			return bridgeTasks.ExecutionResult{
				Status:          taskRunStatusAwaitingHuman,
				SessionIDOutput: lastSessionID,
				ResponsePreview: fmt.Sprintf("workflow awaiting human at %s: %s", node.ID, outcome.preview),
			}
		}
		lastNode = node
		lastPreview = outcome.preview
	}
	return buildWorkflowSuccessResult(lastNode, lastPreview, lastSessionID, r.plan)
}

func buildWorkflowSuccessResult(
	lastNode WorkflowNode,
	lastPreview string,
	sessionID string,
	plan workflowExecutionPlan,
) bridgeTasks.ExecutionResult {
	if strings.TrimSpace(lastNode.ID) == "" {
		return bridgeTasks.ExecutionResult{
			Status:          taskRunStatusSuccess,
			SessionIDOutput: sessionID,
			ResponsePreview: fmt.Sprintf("workflow completed: %s -> %s", plan.ordered[0].ID, plan.ordered[len(plan.ordered)-1].ID),
		}
	}
	return bridgeTasks.ExecutionResult{
		Status:          taskRunStatusSuccess,
		SessionIDOutput: sessionID,
		ResponsePreview: fmt.Sprintf("workflow completed at %s: %s", lastNode.ID, lastPreview),
	}
}

func (r workflowTaskRunner) executeNode(
	ctx context.Context,
	deps agentRuntimeDependencies,
	node WorkflowNode,
) workflowNodeOutcome {
	switch node.Type {
	case workflowNodeTypeTool:
		return executeWorkflowToolNode(ctx, deps, node, r.traceID)
	case workflowNodeTypeLLM:
		return executeWorkflowLLMNode(ctx, deps, node)
	case workflowNodeTypeAgent:
		return r.executeAgentNode(ctx, node)
	default:
		return workflowNodeOutcome{err: fmt.Errorf("unsupported workflow node type %q", node.Type)}
	}
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
	args, err := encodeWorkflowToolArguments(node.Tool.Arguments)
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
		return workflowNodeOutcome{
			status:  taskRunStatusAwaitingHuman,
			preview: truncateRunes(strings.TrimSpace(meta.AwaitingHuman.Prompt), maxTaskResponsePreviewRunes),
		}
	}
	return workflowNodeOutcome{
		status:  taskRunStatusSuccess,
		preview: fmt.Sprintf("tool %s executed", toolName),
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

func workflowNodeToolCallID(nodeID string) string {
	return "workflow-" + strings.TrimSpace(nodeID)
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
	return workflowNodeOutcome{
		status:  taskRunStatusSuccess,
		preview: truncateRunes(text, maxTaskResponsePreviewRunes),
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

func (r workflowTaskRunner) executeAgentNode(
	ctx context.Context,
	node WorkflowNode,
) workflowNodeOutcome {
	result := r.adapter.runAgentAction(ctx, agentParams{
		Message:   node.Agent.Message,
		SessionID: "",
	}, r.traceID)
	if strings.TrimSpace(result.Error) != "" {
		return workflowNodeOutcome{err: fmt.Errorf("%s", result.Error)}
	}
	return workflowNodeOutcome{
		status:    result.Status,
		sessionID: result.SessionIDOutput,
		preview:   result.ResponsePreview,
	}
}
