package screen

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/artifacts"
	"ghost-os/bridge/llm"
)

const (
	screenControlToolName        = "screen_control"
	screenControlModeAtom        = "atomic"
	screenControlActionTextInput = "text_input"
)

const screenControlDescription = "Unified screen control entrypoint for direct screenshot/OCR/click/input actions (atomic only)."

const screenControlSchema = `{
	"type":"object",
	"properties":{
		"mode":{"type":"string","enum":["atomic"],"description":"Optional; when provided it must be atomic."},
		"action":{"type":"string","enum":["screenshot","ocr_scan","click_text","find_icon","click_icon","mouse_position","text_input"],"description":"Required atomic action."},
		"params":{"type":"object","description":"Optional parameters for atomic actions."},
		"display_id":{"type":"integer","minimum":0,"description":"Optional display id forwarded into params.display_id."}
	},
	"required":["action"],
	"additionalProperties":false
}`

type ScreenControlTool struct {
	screenAction Tool
	execution    ExecutionClient
}

type screenControlArgs struct {
	Mode      string         `json:"mode,omitempty"`
	Action    string         `json:"action,omitempty"`
	Params    map[string]any `json:"params,omitempty"`
	Goal      string         `json:"goal,omitempty"`
	Target    map[string]any `json:"target,omitempty"`
	DisplayID *int           `json:"display_id,omitempty"`
}

func NewScreenControlTool(
	client ExecutionClient,
	_ llm.Completer,
	_ *artifacts.SessionArtifactStore,
) Tool {
	return &ScreenControlTool{
		screenAction: NewScreenActionTool(client),
		execution:    client,
	}
}

func (ScreenControlTool) Name() string {
	return screenControlToolName
}

func (ScreenControlTool) Description() string {
	return screenControlDescription
}

func (ScreenControlTool) Parameters() json.RawMessage {
	return json.RawMessage(screenControlSchema)
}

func (t *ScreenControlTool) Execute(
	ctx context.Context,
	argsJSON json.RawMessage,
	traceID string,
) (string, error) {
	args, err := decodeScreenControlArgs(argsJSON)
	if err != nil {
		return "", err
	}
	if err := validateScreenControlArgs(args); err != nil {
		return "", err
	}
	return t.executeAtomic(ctx, args, traceID)
}

func decodeScreenControlArgs(argsJSON json.RawMessage) (screenControlArgs, error) {
	var args screenControlArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return screenControlArgs{}, fmt.Errorf("decode args: %w", err)
	}
	return args, nil
}

func validateScreenControlArgs(args screenControlArgs) error {
	mode := strings.ToLower(strings.TrimSpace(args.Mode))
	action := strings.TrimSpace(args.Action)
	goal := strings.TrimSpace(args.Goal)
	if mode != "" && mode != screenControlModeAtom {
		if mode == "agent" {
			return fmt.Errorf("screen_control mode=%q is removed", mode)
		}
		return fmt.Errorf("screen_control only supports mode=%q", screenControlModeAtom)
	}
	if action == "" {
		return fmt.Errorf("action is required")
	}
	if goal != "" {
		return fmt.Errorf("screen_control does not support goal")
	}
	if len(args.Target) > 0 {
		return fmt.Errorf("screen_control does not support target")
	}
	if hasWorkflowOnlyTemplateDataURL(args.Params) {
		return fmt.Errorf("params.workflow_template_data_url is workflow-only")
	}
	return nil
}

func (t *ScreenControlTool) executeAtomic(
	ctx context.Context,
	args screenControlArgs,
	traceID string,
) (string, error) {
	normalizedAction := strings.ToLower(strings.TrimSpace(args.Action))
	if normalizedAction == screenControlActionTextInput {
		return t.executeTextInput(ctx, args, traceID)
	}
	if t == nil || t.screenAction == nil {
		return "", fmt.Errorf("screen_action backend is not configured")
	}
	params, err := buildScreenControlAtomicParams(args.Params, args.DisplayID)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(screenActionArgs{
		Action: strings.TrimSpace(args.Action),
		Params: params,
	})
	if err != nil {
		return "", fmt.Errorf("encode atomic args: %w", err)
	}
	return t.screenAction.Execute(ctx, payload, traceID)
}

func (t *ScreenControlTool) executeTextInput(
	ctx context.Context,
	args screenControlArgs,
	traceID string,
) (string, error) {
	if t == nil || t.execution == nil {
		return "", fmt.Errorf("text_input backend is not configured")
	}
	if args.DisplayID != nil {
		return "", fmt.Errorf("display_id is not supported for action=%q", screenControlActionTextInput)
	}
	text, submit, err := parseScreenControlTextInputParams(args.Params)
	if err != nil {
		return "", err
	}
	params := map[string]any{"text": text}
	if submit {
		params["submit"] = true
	}
	payload, err := t.execution.Call(ctx, "TEXT_INPUT", params, traceID)
	if err != nil {
		return "", fmt.Errorf("execution TEXT_INPUT failed: %w", err)
	}
	result := cloneScreenControlParams(payload)
	result["action"] = screenControlActionTextInput
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("encode payload: %w", err)
	}
	return string(encoded), nil
}

func parseScreenControlTextInputParams(params map[string]any) (string, bool, error) {
	for key := range params {
		if key == "text" || key == "submit" {
			continue
		}
		return "", false, fmt.Errorf("params.%s is not supported for action=%q", key, screenControlActionTextInput)
	}
	value, exists := params["text"]
	if !exists || value == nil {
		return "", false, fmt.Errorf("params.text is required for action=%q", screenControlActionTextInput)
	}
	text, ok := value.(string)
	if !ok {
		return "", false, fmt.Errorf("params.text must be a string")
	}
	if text == "" {
		return "", false, fmt.Errorf("params.text is required for action=%q", screenControlActionTextInput)
	}
	submitValue, exists := params["submit"]
	if !exists {
		return text, false, nil
	}
	submit, ok := submitValue.(bool)
	if !ok {
		return "", false, fmt.Errorf("params.submit must be a boolean")
	}
	return text, submit, nil
}

func (t *ScreenControlTool) InterpretResult(output string) ExecuteMeta {
	if t == nil || t.screenAction == nil {
		return ExecuteMeta{}
	}
	return InterpretExecuteResult(t.screenAction, output)
}

func buildScreenControlAtomicParams(
	raw map[string]any,
	displayID *int,
) (map[string]any, error) {
	params := cloneScreenControlParams(raw)
	if displayID == nil {
		return params, nil
	}
	if _, duplicated := params["display_id"]; duplicated {
		return nil, fmt.Errorf("display_id is duplicated; provide either top-level display_id or params.display_id")
	}
	params["display_id"] = *displayID
	return params, nil
}

func cloneScreenControlParams(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = value
	}
	return cloned
}

func hasWorkflowOnlyTemplateDataURL(params map[string]any) bool {
	if len(params) == 0 {
		return false
	}
	_, exists := params["workflow_template_data_url"]
	return exists
}
