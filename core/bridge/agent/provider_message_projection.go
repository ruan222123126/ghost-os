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
	candidate, ok := newProjectedGraphQLRepairCandidate(source, projected, index)
	if !ok {
		return
	}
	toolCall, ok := buildProjectedGraphQLRepairToolCall(executor, candidate, projected)
	if !ok {
		return
	}
	if !matchesProjectedGraphQLToolEnvelope(candidate.toolResult, toolCall.Name) {
		return
	}
	projected[candidate.assistantIndex].ToolCalls = append(projected[candidate.assistantIndex].ToolCalls, toolCall)
}

type projectedGraphQLRepairCandidate struct {
	assistantIndex int
	assistantText  string
	toolCallID     string
	toolResult     string
}

func newProjectedGraphQLRepairCandidate(
	source []llm.Message,
	projected []llm.Message,
	toolIndex int,
) (projectedGraphQLRepairCandidate, bool) {
	current := source[toolIndex]
	toolCallID := strings.TrimSpace(current.ToolCallID)
	if current.Role != llm.RoleTool || toolCallID == "" {
		return projectedGraphQLRepairCandidate{}, false
	}

	assistantIndex := findProjectedGraphQLAssistantIndex(source, toolIndex)
	if assistantIndex < 0 {
		return projectedGraphQLRepairCandidate{}, false
	}

	assistantMessage := source[assistantIndex]
	if !isProjectedGraphQLAssistantCandidate(assistantMessage) {
		return projectedGraphQLRepairCandidate{}, false
	}
	if hasToolCallID(projected[assistantIndex].ToolCalls, toolCallID) {
		return projectedGraphQLRepairCandidate{}, false
	}
	return projectedGraphQLRepairCandidate{
		assistantIndex: assistantIndex,
		assistantText:  assistantMessage.Text,
		toolCallID:     toolCallID,
		toolResult:     current.Text,
	}, true
}

func isProjectedGraphQLAssistantCandidate(message llm.Message) bool {
	if message.Role != llm.RoleAssistant {
		return false
	}
	if len(message.ToolCalls) > 0 {
		return false
	}
	return strings.TrimSpace(message.Text) != ""
}

func buildProjectedGraphQLRepairToolCall(
	executor tools.GraphQLTextExecutor,
	candidate projectedGraphQLRepairCandidate,
	projected []llm.Message,
) (llm.ToolCall, bool) {
	result, err := executor.Execute(context.Background(), candidate.assistantText, "")
	if err != nil && !canRepairProjectedLegacyGraphQLTextTurn(err, result) {
		return llm.ToolCall{}, false
	}

	call, ok := projectedGraphQLToolCallForIndex(result, len(projected[candidate.assistantIndex].ToolCalls))
	if !result.Recognized || !ok {
		return llm.ToolCall{}, false
	}
	toolName := strings.TrimSpace(call.ToolName)
	if toolName == "" {
		return llm.ToolCall{}, false
	}
	return llm.ToolCall{
		ID:        candidate.toolCallID,
		Name:      toolName,
		Arguments: cloneToolArguments(call.Arguments),
	}, true
}

func matchesProjectedGraphQLToolEnvelope(rawResult string, toolName string) bool {
	envelope, ok := ParseToolResultEnvelope(rawResult)
	if !ok {
		return true
	}
	envelopeTool := strings.TrimSpace(envelope.Tool)
	if envelopeTool == "" {
		return true
	}
	return envelopeTool == strings.TrimSpace(toolName)
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
