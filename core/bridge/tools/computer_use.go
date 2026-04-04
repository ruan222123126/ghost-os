package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/artifacts"
	"ghost-os/bridge/guiagent"
	"ghost-os/bridge/llm"
)

const computerUseToolName = "computer_use"

type ComputerUseTool struct {
	execution     ExecutionClient
	model         llm.Completer
	artifactStore *artifacts.SessionArtifactStore
	setupErr      error
}

type computerUseArgs struct {
	Goal      string                `json:"goal"`
	Mode      string                `json:"mode,omitempty"`
	Target    guiagent.WindowTarget `json:"target,omitempty"`
	DisplayID *int                  `json:"display_id,omitempty"`
}

func NewComputerUseTool(
	client ExecutionClient,
	model llm.Completer,
	artifactStore *artifacts.SessionArtifactStore,
) Tool {
	tool := &ComputerUseTool{
		execution:     client,
		model:         model,
		artifactStore: artifactStore,
	}
	switch {
	case client == nil:
		tool.setupErr = fmt.Errorf("execution client is not configured")
	case model == nil:
		tool.setupErr = fmt.Errorf("computer_use model is not configured")
	case artifactStore == nil:
		tool.setupErr = fmt.Errorf("artifact store is not configured")
	}
	return tool
}

func (ComputerUseTool) Name() string {
	return computerUseToolName
}

func (ComputerUseTool) Description() string {
	return "Run a desktop GUI executor loop for high-level visual tasks. Use this only after script/API/browser-native options are insufficient."
}

func (ComputerUseTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"goal":{"type":"string","minLength":1,"description":"High-level desktop task goal."},
			"mode":{"type":"string","enum":["desktop"],"description":"Execution mode. Only desktop is supported in v1."},
			"target":{
				"type":"object",
				"properties":{
					"window_title":{"type":"string"},
					"window_class":{"type":"string"}
				},
				"additionalProperties":false
			},
			"display_id":{"type":"integer","minimum":0,"description":"Optional display id for screenshot capture and coordinate mapping."}
		},
		"required":["goal"],
		"additionalProperties":false
	}`)
}

func (t *ComputerUseTool) Execute(
	ctx context.Context,
	argsJSON json.RawMessage,
	traceID string,
) (string, error) {
	if t.setupErr != nil {
		return "", t.setupErr
	}
	request, err := decodeComputerUseArgs(argsJSON)
	if err != nil {
		return "", err
	}
	runID, err := guiagent.NewRunID()
	if err != nil {
		return "", err
	}
	state := guiagent.State{RunID: runID, Request: request}
	return t.runState(ctx, state, "", traceID)
}

func (ComputerUseTool) InterpretResult(output string) ExecuteMeta {
	return AskHumanTool{}.InterpretResult(output)
}

func (t *ComputerUseTool) ResumeFromHumanAnswer(
	ctx context.Context,
	questionID string,
	answer string,
	traceID string,
) (string, ExecuteMeta, bool, error) {
	if t.setupErr != nil {
		return "", ExecuteMeta{}, false, t.setupErr
	}
	sess := SessionFromContext(ctx)
	if sess == nil {
		return "", ExecuteMeta{}, false, fmt.Errorf("computer_use requires an active session")
	}
	state, ok := sess.PendingComputerUseRun(strings.TrimSpace(questionID))
	if !ok {
		return "", ExecuteMeta{}, false, fmt.Errorf("computer_use pending run %q not found", questionID)
	}
	sess.RemovePendingComputerUseRun(questionID)
	output, err := t.runState(ctx, state, answer, traceID)
	if err != nil {
		return "", ExecuteMeta{}, true, err
	}
	return output, ComputerUseTool{}.InterpretResult(output), true, nil
}

func (t *ComputerUseTool) runState(
	ctx context.Context,
	state guiagent.State,
	answer string,
	traceID string,
) (string, error) {
	runner, err := t.newRunner(ctx, state.RunID)
	if err != nil {
		return "", err
	}
	result, err := runner.Continue(ctx, state, answer, traceID)
	if err != nil {
		return "", err
	}
	if result.Status != guiagent.StatusAwaitingUser {
		return guiagent.EncodeResult(result)
	}
	return t.persistAwaitingState(ctx, result, traceID)
}

func (t *ComputerUseTool) newRunner(ctx context.Context, runID string) (*guiagent.Runner, error) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return nil, fmt.Errorf("computer_use requires an active session")
	}
	writer, err := guiagent.NewArtifactWriter(t.artifactStore, sess.ID, runID)
	if err != nil {
		return nil, err
	}
	return guiagent.NewRunner(
		t.model,
		guiagent.NewDesktopOperator(t.execution),
		writer,
		0,
	), nil
}

func (t *ComputerUseTool) persistAwaitingState(
	ctx context.Context,
	result guiagent.Result,
	traceID string,
) (string, error) {
	sess := SessionFromContext(ctx)
	if sess == nil {
		return "", fmt.Errorf("computer_use requires an active session")
	}
	if result.State == nil || result.AwaitingUser == nil {
		return "", fmt.Errorf("computer_use awaiting state is incomplete")
	}
	questionID, err := newQuestionID()
	if err != nil {
		return "", fmt.Errorf("generate question id: %w", err)
	}
	sess.SetPendingComputerUseRun(questionID, *result.State)
	output, err := registerAwaitingHumanQuestion(ctx, questionID, askHumanArgs{
		Prompt: result.AwaitingUser.Prompt,
	}, traceID, computerUseToolName)
	if err != nil {
		sess.RemovePendingComputerUseRun(questionID)
		return "", err
	}
	return output, nil
}

func decodeComputerUseArgs(argsJSON json.RawMessage) (guiagent.Request, error) {
	var args computerUseArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return guiagent.Request{}, fmt.Errorf("decode args: %w", err)
	}
	goal := strings.TrimSpace(args.Goal)
	if goal == "" {
		return guiagent.Request{}, fmt.Errorf("goal is required")
	}
	mode := strings.TrimSpace(args.Mode)
	if mode == "" {
		mode = "desktop"
	}
	if mode != "desktop" {
		return guiagent.Request{}, fmt.Errorf("mode must be %q", "desktop")
	}
	return guiagent.Request{
		Goal:      goal,
		Mode:      mode,
		Target:    args.Target,
		DisplayID: args.DisplayID,
	}, nil
}
