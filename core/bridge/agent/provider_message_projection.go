package agent

import (
	"context"
	"encoding/json"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

func projectMessagesForProvider(messages []llm.Message, catalog ToolCatalog) []llm.Message {
	projected := llm.ProjectMessagesForCompletion(messages)
	if len(messages) < 2 || len(projected) != len(messages) || catalog == nil {
		return projected
	}

	executor := tools.NewGraphQLTextExecutor(catalog)
	for index := 1; index < len(messages); index++ {
		repairProjectedGraphQLTextTurn(messages, projected, executor, index)
	}
	return projected
}

func repairProjectedGraphQLTextTurn(
	source []llm.Message,
	projected []llm.Message,
	executor tools.GraphQLTextExecutor,
	index int,
) {
	if executor == nil {
		return
	}
	current := source[index]
	if current.Role != llm.RoleTool || strings.TrimSpace(current.ToolCallID) == "" {
		return
	}

	assistantIndex := findProjectedGraphQLAssistantIndex(source, index)
	if assistantIndex < 0 {
		return
	}
	prev := source[assistantIndex]
	if prev.Role != llm.RoleAssistant || len(prev.ToolCalls) != 0 || strings.TrimSpace(prev.Text) == "" {
		return
	}
	if hasToolCallID(projected[assistantIndex].ToolCalls, current.ToolCallID) {
		return
	}

	result, err := executor.Execute(context.Background(), prev.Text, "")
	if err != nil && !canRepairProjectedLegacyGraphQLTextTurn(err, result) {
		return
	}
	call, ok := projectedGraphQLToolCallForIndex(result, len(projected[assistantIndex].ToolCalls))
	if !result.Recognized || !ok || strings.TrimSpace(call.ToolName) == "" {
		return
	}

	if envelope, ok := ParseToolResultEnvelope(current.Text); ok {
		if envelopeTool := strings.TrimSpace(envelope.Tool); envelopeTool != "" && envelopeTool != call.ToolName {
			return
		}
	}

	projected[assistantIndex].ToolCalls = append(projected[assistantIndex].ToolCalls, llm.ToolCall{
		ID:        strings.TrimSpace(current.ToolCallID),
		Name:      strings.TrimSpace(call.ToolName),
		Arguments: cloneToolArguments(call.Arguments),
	})
}

func findProjectedGraphQLAssistantIndex(messages []llm.Message, toolIndex int) int {
	for index := toolIndex - 1; index >= 0; index-- {
		msg := messages[index]
		switch msg.Role {
		case llm.RoleInternal, llm.RoleTool:
			continue
		case llm.RoleAssistant:
			if len(msg.ToolCalls) == 0 && strings.TrimSpace(msg.Text) != "" {
				return index
			}
			return -1
		default:
			return -1
		}
	}
	return -1
}

func canRepairProjectedLegacyGraphQLTextTurn(
	err error,
	result tools.GraphQLTextExecutionResult,
) bool {
	if err == nil || !result.Recognized {
		return false
	}
	protocolErr, ok := tools.AsGraphQLTextProtocolError(err)
	if !ok {
		return false
	}
	return protocolErr.Feedback().Kind == "wrong_operation"
}

func projectedGraphQLToolCallForIndex(
	result tools.GraphQLTextExecutionResult,
	index int,
) (tools.GraphQLTextToolCall, bool) {
	if index < 0 {
		return tools.GraphQLTextToolCall{}, false
	}
	if len(result.Calls) > index {
		return result.Calls[index], true
	}
	if index == 0 && strings.TrimSpace(result.ToolName) != "" {
		return tools.GraphQLTextToolCall{
			Operation: result.Operation,
			ToolName:  strings.TrimSpace(result.ToolName),
			Arguments: cloneToolArguments(result.Arguments),
		}, true
	}
	return tools.GraphQLTextToolCall{}, false
}

func hasToolCallID(calls []llm.ToolCall, toolCallID string) bool {
	trimmed := strings.TrimSpace(toolCallID)
	if trimmed == "" {
		return false
	}
	for _, call := range calls {
		if strings.TrimSpace(call.ID) == trimmed {
			return true
		}
	}
	return false
}

func cloneToolArguments(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	cloned := make([]byte, len(raw))
	copy(cloned, raw)
	return json.RawMessage(cloned)
}
