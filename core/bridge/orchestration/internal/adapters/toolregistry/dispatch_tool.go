package toolregistry

import (
	"context"
	"encoding/json"
	"fmt"

	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/tools"
)

const DispatchToolName = "orchestration_dispatch"

type DispatchTool struct {
	GroupNode   group.Node
	MemberOrder []string
}

func (t DispatchTool) Name() string {
	return DispatchToolName
}

func (t DispatchTool) Description() string {
	return "Dispatch exactly one orchestration sub-round for the current owner-led group."
}

func (t DispatchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"action":{"type":"string","enum":["public_once","private_once","end_group"]},
			"participant_ids":{"type":"array","items":{"type":"string"}},
			"order":{"type":"string","enum":["sequential","parallel"]},
			"instruction":{"type":"string"}
		},
		"required":["action"],
		"additionalProperties":false
	}`)
}

func (t DispatchTool) Execute(
	_ context.Context,
	argsJSON json.RawMessage,
	_ string,
) (string, error) {
	var req group.DispatchCommand
	if err := json.Unmarshal(argsJSON, &req); err != nil {
		return "", fmt.Errorf("decode dispatch args: %w", err)
	}
	normalized, err := group.DispatchValidator{}.Validate(
		req,
		group.NewGroupRef(t.GroupNode),
		t.MemberOrder,
	)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func (t DispatchTool) InterpretResult(output string) tools.ExecuteMeta {
	var req group.DispatchCommand
	if err := json.Unmarshal([]byte(output), &req); err != nil {
		return tools.ExecuteMeta{}
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return tools.ExecuteMeta{}
	}
	return tools.ExecuteMeta{
		Iteration: &tools.IterationHandoffSignal{
			Did:       string(payload),
			Remaining: "owner dispatch submitted",
		},
	}
}
