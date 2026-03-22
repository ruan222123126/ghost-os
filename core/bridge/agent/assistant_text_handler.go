package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type AssistantTextHandler interface {
	HandleAssistantText(context.Context, AssistantTextRequest) (AssistantTextResult, error)
}

type AssistantTextRequest struct {
	Text    string
	TraceID string
}

type AssistantTextToolRef struct {
	Name   string
	CallID string
}

type AssistantTextAwaitingHuman struct {
	Tool          AssistantTextToolRef
	QuestionID    string
	Prompt        string
	SelectionMode string
	Options       []tools.AskHumanOption
}

type AssistantTextToolExecutionResult struct {
	Tool   AssistantTextToolRef
	Output string
	Meta   tools.ExecuteMeta
}

type AssistantTextToolFeedbackBuilder func(AssistantTextToolExecutionResult) []llm.Message

type AssistantTextToolInvocation struct {
	Arguments       json.RawMessage
	FeedbackBuilder AssistantTextToolFeedbackBuilder
}

type AssistantTextResult struct {
	Recognized    bool
	Tool          AssistantTextToolRef
	Invocation    *AssistantTextToolInvocation
	Feedback      []llm.Message
	AwaitingHuman *AssistantTextAwaitingHuman
}

func validateAssistantTextResult(result AssistantTextResult) error {
	if !result.Recognized {
		return nil
	}
	if strings.TrimSpace(result.Tool.Name) == "" {
		return fmt.Errorf("recognized assistant text handler returned empty tool name")
	}
	if strings.TrimSpace(result.Tool.CallID) == "" {
		return fmt.Errorf("recognized assistant text handler returned empty tool_call_id")
	}
	if result.Invocation != nil {
		if len(result.Invocation.Arguments) == 0 {
			return fmt.Errorf("recognized assistant text handler returned empty tool arguments")
		}
		if _, err := normalizedToolArguments(result.Invocation.Arguments); err != nil {
			return fmt.Errorf("recognized assistant text handler returned invalid tool arguments: %w", err)
		}
	}
	if result.AwaitingHuman == nil {
		return nil
	}
	if strings.TrimSpace(result.AwaitingHuman.Tool.Name) == "" {
		return fmt.Errorf("recognized assistant text handler returned empty awaiting-human tool name")
	}
	if strings.TrimSpace(result.AwaitingHuman.Tool.CallID) == "" {
		return fmt.Errorf("recognized assistant text handler returned empty awaiting-human tool_call_id")
	}
	return nil
}

func cloneAssistantTextFeedback(messages []llm.Message) []llm.Message {
	return llm.CloneMessages(messages)
}

func buildAssistantTextToolFeedback(
	invocation *AssistantTextToolInvocation,
	result AssistantTextToolExecutionResult,
) []llm.Message {
	if invocation == nil || invocation.FeedbackBuilder == nil {
		return nil
	}
	return cloneAssistantTextFeedback(invocation.FeedbackBuilder(result))
}

func cloneAssistantTextOptions(options []tools.AskHumanOption) []tools.AskHumanOption {
	return append([]tools.AskHumanOption(nil), options...)
}
