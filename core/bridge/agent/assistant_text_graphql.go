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
	graphQLAssistantTextToolName   = "graphql_text"
	graphQLAssistantTextToolCallID = "graphql-text"
	graphQLExecutionResultPrefix   = "[GRAPHQL_EXECUTION_RESULT]\n"
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
	handled := AssistantTextResult{
		Recognized: true,
		Tool: AssistantTextToolRef{
			Name:   graphQLAssistantTextToolName,
			CallID: graphQLAssistantTextToolCallID,
		},
	}
	if err != nil {
		return handled, err
	}
	if awaiting := result.Meta.AwaitingHuman; awaiting != nil {
		handled.AwaitingHuman = &AssistantTextAwaitingHuman{
			Tool: AssistantTextToolRef{
				Name:   tools.GraphQLTextMutationToolName,
				CallID: graphQLAssistantTextToolCallID,
			},
			QuestionID:    strings.TrimSpace(awaiting.QuestionID),
			Prompt:        strings.TrimSpace(awaiting.Prompt),
			SelectionMode: strings.TrimSpace(awaiting.SelectionMode),
			Options:       cloneAssistantTextOptions(awaiting.Options),
		}
		return handled, nil
	}
	feedback, ok := newGraphQLExecutionFeedbackMessage(result.Output)
	if ok {
		handled.Feedback = []llm.Message{feedback}
	}
	return handled, nil
}

func newGraphQLExecutionFeedbackMessage(output string) (llm.Message, bool) {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return llm.Message{}, false
	}
	return llm.Message{
		Role: llm.RoleInternal,
		Text: formatGraphQLExecutionFeedback(trimmed),
	}, true
}

func formatGraphQLExecutionFeedback(output string) string {
	var decoded any
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		return graphQLExecutionResultPrefix + output
	}
	normalized, err := json.Marshal(decoded)
	if err != nil {
		return graphQLExecutionResultPrefix + output
	}
	return fmt.Sprintf("%s%s", graphQLExecutionResultPrefix, string(normalized))
}
