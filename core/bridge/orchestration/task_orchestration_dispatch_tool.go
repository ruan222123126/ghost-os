package orchestration

import (
	"context"
	"encoding/json"
	"fmt"

	groupdomain "ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/tools"
)

const orchestrationDispatchToolName = "orchestration_dispatch"

type orchestrationDispatchTool struct {
	groupNode   OrchestrationNode
	memberOrder []string
}

func (t orchestrationDispatchTool) Name() string {
	return orchestrationDispatchToolName
}

func (t orchestrationDispatchTool) Description() string {
	return "Dispatch exactly one orchestration sub-round for the current owner-led group."
}

func (t orchestrationDispatchTool) Parameters() json.RawMessage {
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

func (t orchestrationDispatchTool) Execute(_ context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	var req orchestrationDispatchRequest
	if err := json.Unmarshal(argsJSON, &req); err != nil {
		return "", fmt.Errorf("decode dispatch args: %w", err)
	}
	normalized, err := groupdomain.DispatchValidator{}.Validate(
		req,
		groupdomain.NewGroupRef(t.groupNode),
		t.memberOrder,
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

func (t orchestrationDispatchTool) InterpretResult(output string) tools.ExecuteMeta {
	var req orchestrationDispatchRequest
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

func (r orchestrationTaskRunner) newOwnerDispatchTool(
	groupNode OrchestrationNode,
	memberOrder []string,
) tools.Tool {
	return orchestrationDispatchTool{
		groupNode:   groupNode,
		memberOrder: append([]string(nil), memberOrder...),
	}
}
