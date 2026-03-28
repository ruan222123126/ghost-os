package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

const (
	graphQLToolCallProtocolName = "graphql_tool_call"
	graphQLToolResultPrefix     = "[GRAPHQL_TOOL_RESULT]\n"
)

type graphQLTextTurnHandler struct {
	executor tools.GraphQLTextExecutor
}

func NewGraphQLTextTurnHandler(executor tools.GraphQLTextExecutor) AssistantTextHandler {
	if executor == nil {
		return nil
	}
	return graphQLTextTurnHandler{executor: executor}
}

func (h graphQLTextTurnHandler) HandleAssistantText(
	ctx context.Context,
	req AssistantTextRequest,
) (AssistantTextResult, error) {
	result, err := h.executor.Execute(ctx, req.Text, req.TraceID)
	if !result.Recognized {
		return AssistantTextResult{}, nil
	}
	toolName := graphQLFallbackToolName(result)
	handled := AssistantTextResult{
		Recognized: true,
		Tool: AssistantTextToolRef{
			Name:   toolName,
			CallID: tools.NewGraphQLTextToolCallID(),
		},
	}
	if err != nil {
		if feedback, ok := newGraphQLToolErrorFeedbackMessage(toolName, err); ok {
			handled.Feedback = []llm.Message{feedback}
		}
		return handled, err
	}
	if len(result.Calls) == 0 {
		return handled, nil
	}
	invocations := make([]AssistantTextToolInvocationEntry, 0, len(result.Calls))
	for _, call := range result.Calls {
		name := strings.TrimSpace(call.ToolName)
		if name == "" {
			name = graphQLToolCallProtocolName
		}
		invocations = append(invocations, AssistantTextToolInvocationEntry{
			Tool: AssistantTextToolRef{
				Name:   name,
				CallID: tools.NewGraphQLTextToolCallID(),
			},
			Invocation: AssistantTextToolInvocation{
				Arguments:       append(json.RawMessage(nil), call.Arguments...),
				FeedbackBuilder: graphQLToolResultFeedbackBuilder,
			},
		})
	}
	handled.Invocations = invocations
	handled.Tool = invocations[0].Tool
	handled.Invocation = &AssistantTextToolInvocation{
		Arguments:       append(json.RawMessage(nil), invocations[0].Invocation.Arguments...),
		FeedbackBuilder: invocations[0].Invocation.FeedbackBuilder,
	}
	return handled, nil
}

func graphQLFallbackToolName(result tools.GraphQLTextExecutionResult) string {
	if len(result.Calls) > 0 {
		if name := strings.TrimSpace(result.Calls[0].ToolName); name != "" {
			return name
		}
	}
	if name := strings.TrimSpace(result.ToolName); name != "" {
		return name
	}
	return graphQLToolCallProtocolName
}

func graphQLToolResultFeedbackBuilder(result AssistantTextToolExecutionResult) []llm.Message {
	feedback, ok := newGraphQLToolResultFeedbackMessage(result.Tool.Name, result.Output)
	if !ok {
		return nil
	}
	return []llm.Message{feedback}
}

func newGraphQLToolResultFeedbackMessage(toolName string, output string) (llm.Message, bool) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return llm.Message{}, false
	}
	return llm.Message{
		Role: llm.RoleInternal,
		Text: formatGraphQLToolResultFeedback(strings.TrimSpace(toolName), trimmed),
	}, true
}

func newGraphQLToolErrorFeedbackMessage(toolName string, err error) (llm.Message, bool) {
	protocolErr, ok := tools.AsGraphQLTextProtocolError(err)
	if !ok {
		return llm.Message{}, false
	}
	payload := protocolErr.Feedback()
	if payload.Tool == "" {
		payload.Tool = strings.TrimSpace(toolName)
	}
	normalized, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return llm.Message{
			Role: llm.RoleInternal,
			Text: graphQLToolResultPrefix + err.Error(),
		}, true
	}
	return llm.Message{
		Role: llm.RoleInternal,
		Text: fmt.Sprintf("%s%s", graphQLToolResultPrefix, string(normalized)),
	}, true
}

func formatGraphQLToolResultFeedback(toolName string, output string) string {
	payload := map[string]any{
		"tool":   toolName,
		"output": output,
	}
	var decoded any
	if err := json.Unmarshal([]byte(output), &decoded); err == nil {
		payload["output"] = decoded
	}
	normalized, err := json.Marshal(payload)
	if err != nil {
		return graphQLToolResultPrefix + output
	}
	return fmt.Sprintf("%s%s", graphQLToolResultPrefix, string(normalized))
}
