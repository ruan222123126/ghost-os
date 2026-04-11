package orchestration

import (
	"context"
	"fmt"
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
)

type workflowNodeOutcome struct {
	status     string
	sessionID  string
	preview    string
	outputText string
	err        error
}

type workflowRunState struct {
	lastOutputText string
	nodeOutputs    map[string]string
	loopIterations map[string]int
}

func newWorkflowRunState(nodeCount int) workflowRunState {
	return workflowRunState{
		lastOutputText: "",
		nodeOutputs:    make(map[string]string, nodeCount),
		loopIterations: make(map[string]int, nodeCount),
	}
}

func (s *workflowRunState) recordNode(nodeID string, outcome workflowNodeOutcome) {
	output := strings.TrimSpace(outcome.outputText)
	if output == "" {
		output = strings.TrimSpace(outcome.preview)
	}
	s.lastOutputText = output
	s.nodeOutputs[nodeID] = output
}

func (s workflowRunState) sourceText(sourceNodeID string) string {
	if strings.TrimSpace(sourceNodeID) == "" {
		return s.lastOutputText
	}
	return strings.TrimSpace(s.nodeOutputs[sourceNodeID])
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
	if a.service == nil && plan.needsService() {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "task executor service is not configured"}
	}
	if a.service != nil {
		cfg, err := loadTaskRuntimeConfig(a.service.configStore)
		if err != nil {
			return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
		}
		if err := validateWorkflowTaskRuntime(task.Workflow, cfg); err != nil {
			return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
		}
	}
	return newWorkflowTaskRunner(a, plan, traceID).execute(ctx)
}

type workflowTaskRunner struct {
	adapter taskExecutorAdapter
	plan    workflowExecutionPlan
	traceID string
}

func newWorkflowTaskRunner(adapter taskExecutorAdapter, plan workflowExecutionPlan, traceID string) workflowTaskRunner {
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
	if !r.plan.needsRuntimeDependencies() {
		return agentRuntimeDependencies{}, nil
	}
	if r.adapter.service == nil {
		return agentRuntimeDependencies{}, fmt.Errorf("workflow runtime service is not configured")
	}
	return r.adapter.service.runtimeFactory.Build(r.adapter.service.configStore)
}

func (r workflowTaskRunner) executePlan(ctx context.Context, deps agentRuntimeDependencies) bridgeTasks.ExecutionResult {
	state := newWorkflowRunState(len(r.plan.nodes))
	lastSessionID := ""
	lastNode := WorkflowNode{}
	lastPreview := ""
	currentNodeID := r.plan.startID
	for {
		node, ok := r.plan.node(currentNodeID)
		if !ok {
			return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: fmt.Sprintf("workflow node %q is missing", currentNodeID)}
		}
		if node.Type == workflowNodeTypeEnd {
			return buildWorkflowSuccessResult(lastNode, lastPreview, lastSessionID, r.plan)
		}
		step := r.executeStep(ctx, deps, node, &state)
		if step.err != nil {
			return bridgeTasks.ExecutionResult{
				Status: taskRunStatusError,
				Error:  fmt.Sprintf("workflow node %s (%s) failed: %v", node.ID, node.Type, step.err),
			}
		}
		if strings.TrimSpace(step.outcome.sessionID) != "" {
			lastSessionID = strings.TrimSpace(step.outcome.sessionID)
		}
		if step.outcome.status == taskRunStatusAwaitingHuman {
			return bridgeTasks.ExecutionResult{
				Status:          taskRunStatusAwaitingHuman,
				SessionIDOutput: lastSessionID,
				ResponsePreview: fmt.Sprintf("workflow awaiting human at %s: %s", node.ID, step.outcome.preview),
			}
		}
		if step.executed {
			state.recordNode(node.ID, step.outcome)
			lastNode = node
			lastPreview = step.outcome.preview
		}
		currentNodeID = step.nextNodeID
	}
}

type workflowStepResult struct {
	nextNodeID string
	outcome    workflowNodeOutcome
	executed   bool
	err        error
}

func (r workflowTaskRunner) executeStep(
	ctx context.Context,
	deps agentRuntimeDependencies,
	node WorkflowNode,
	state *workflowRunState,
) workflowStepResult {
	switch node.Type {
	case workflowNodeTypeStart:
		nextID, err := r.plan.singleNextNodeID(node.ID)
		return workflowStepResult{nextNodeID: nextID, err: err}
	case workflowNodeTypeTool, workflowNodeTypeLLM, workflowNodeTypeAgent:
		return r.executeActionStep(ctx, deps, node)
	case workflowNodeTypeIf:
		nextID, err := selectWorkflowIfNextNode(node, *state)
		return workflowStepResult{nextNodeID: nextID, err: err}
	case workflowNodeTypeLoop:
		nextID, err := selectWorkflowLoopNextNode(node, state)
		return workflowStepResult{nextNodeID: nextID, err: err}
	default:
		return workflowStepResult{err: fmt.Errorf("unsupported workflow node type %q", node.Type)}
	}
}

func (r workflowTaskRunner) executeActionStep(
	ctx context.Context,
	deps agentRuntimeDependencies,
	node WorkflowNode,
) workflowStepResult {
	outcome := r.executeActionNode(ctx, deps, node)
	if outcome.err != nil || outcome.status == taskRunStatusAwaitingHuman {
		return workflowStepResult{outcome: outcome, executed: true, err: outcome.err}
	}
	nextID, err := r.plan.singleNextNodeID(node.ID)
	return workflowStepResult{
		nextNodeID: nextID,
		outcome:    outcome,
		executed:   true,
		err:        err,
	}
}

func (r workflowTaskRunner) executeActionNode(
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

func selectWorkflowIfNextNode(node WorkflowNode, state workflowRunState) (string, error) {
	if node.If == nil {
		return "", fmt.Errorf("workflow if node %q payload is missing", node.ID)
	}
	sourceText := state.sourceText(node.If.SourceNodeID)
	matched, err := evaluateWorkflowIfCondition(node.If.Operator, sourceText, node.If.Value)
	if err != nil {
		return "", fmt.Errorf("workflow if node %q: %w", node.ID, err)
	}
	if matched {
		return strings.TrimSpace(node.If.TrueNodeID), nil
	}
	return strings.TrimSpace(node.If.FalseNodeID), nil
}

func evaluateWorkflowIfCondition(operator string, source string, value string) (bool, error) {
	normalizedOperator := strings.TrimSpace(operator)
	sourceText := strings.TrimSpace(source)
	targetValue := strings.TrimSpace(value)
	switch normalizedOperator {
	case workflowIfOperatorEquals:
		return sourceText == targetValue, nil
	case workflowIfOperatorNotEquals:
		return sourceText != targetValue, nil
	case workflowIfOperatorContains:
		return strings.Contains(sourceText, targetValue), nil
	case workflowIfOperatorNotContains:
		return !strings.Contains(sourceText, targetValue), nil
	case workflowIfOperatorIsEmpty:
		return sourceText == "", nil
	case workflowIfOperatorNotEmpty:
		return sourceText != "", nil
	default:
		return false, fmt.Errorf("unsupported operator %q", operator)
	}
}

func selectWorkflowLoopNextNode(node WorkflowNode, state *workflowRunState) (string, error) {
	if node.Loop == nil {
		return "", fmt.Errorf("workflow loop node %q payload is missing", node.ID)
	}
	count := state.loopIterations[node.ID]
	if count < node.Loop.MaxIterations {
		state.loopIterations[node.ID] = count + 1
		return strings.TrimSpace(node.Loop.BodyNodeID), nil
	}
	return strings.TrimSpace(node.Loop.ExitNodeID), nil
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
			ResponsePreview: fmt.Sprintf("workflow completed: %s -> %s", plan.startID, plan.endID),
		}
	}
	return bridgeTasks.ExecutionResult{
		Status:          taskRunStatusSuccess,
		SessionIDOutput: sessionID,
		ResponsePreview: fmt.Sprintf("workflow completed at %s: %s", lastNode.ID, lastPreview),
	}
}
