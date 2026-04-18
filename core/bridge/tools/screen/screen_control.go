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
	screenControlToolName = "screen_control"
	screenControlModeAtom = "atomic"
)

const screenControlDescription = "Unified screen control entrypoint for direct screenshot/OCR/click actions (atomic only)."

const screenControlSchema = `{
	"type":"object",
	"properties":{
		"mode":{"type":"string","enum":["atomic"],"description":"Optional; when provided it must be atomic."},
		"action":{"type":"string","enum":["screenshot","ocr_scan","click_text","find_icon","click_icon"],"description":"Required atomic action."},
		"params":{"type":"object","description":"Optional parameters for atomic actions."},
		"display_id":{"type":"integer","minimum":0,"description":"Optional display id forwarded into params.display_id."}
	},
	"required":["action"],
	"additionalProperties":false
}`

type ScreenControlTool struct {
	screenAction Tool
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
