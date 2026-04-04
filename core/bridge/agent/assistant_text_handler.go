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

type AssistantTextToolInvocationEntry struct {
	Tool       AssistantTextToolRef
	Invocation AssistantTextToolInvocation
}

type AssistantTextResult struct {
	Recognized    bool
	Tool          AssistantTextToolRef
	Invocation    *AssistantTextToolInvocation
	Invocations   []AssistantTextToolInvocationEntry
	Feedback      []llm.Message
	AwaitingHuman *AssistantTextAwaitingHuman
}

func validateAssistantTextResult(result AssistantTextResult) error {
	if !result.Recognized {
		return nil
	}
	if err := validateAssistantTextInvocations(result); err != nil {
		return err
	}
	return validateAssistantTextAwaitingHuman(result.AwaitingHuman)
}

func validateAssistantTextInvocations(result AssistantTextResult) error {
	if len(result.Invocations) > 0 {
		return validateAssistantTextBatchInvocations(result.Invocations)
	}
	return validateAssistantTextSingleInvocation(result.Tool, result.Invocation)
}

func validateAssistantTextBatchInvocations(entries []AssistantTextToolInvocationEntry) error {
	for index, entry := range entries {
		if strings.TrimSpace(entry.Tool.Name) == "" {
			return fmt.Errorf("recognized assistant text handler returned empty tool name in invocations[%d]", index)
		}
		if strings.TrimSpace(entry.Tool.CallID) == "" {
			return fmt.Errorf("recognized assistant text handler returned empty tool_call_id in invocations[%d]", index)
		}
		if err := validateAssistantTextArguments(
			entry.Invocation.Arguments,
			fmt.Sprintf("recognized assistant text handler returned empty tool arguments in invocations[%d]", index),
			fmt.Sprintf("recognized assistant text handler returned invalid tool arguments in invocations[%d]", index),
		); err != nil {
			return err
		}
	}
	return nil
}

func validateAssistantTextSingleInvocation(
	tool AssistantTextToolRef,
	invocation *AssistantTextToolInvocation,
) error {
	if strings.TrimSpace(tool.Name) == "" {
		return fmt.Errorf("recognized assistant text handler returned empty tool name")
	}
	if strings.TrimSpace(tool.CallID) == "" {
		return fmt.Errorf("recognized assistant text handler returned empty tool_call_id")
	}
	if invocation == nil {
		return nil
	}
	return validateAssistantTextArguments(
		invocation.Arguments,
		"recognized assistant text handler returned empty tool arguments",
		"recognized assistant text handler returned invalid tool arguments",
	)
}

func validateAssistantTextArguments(arguments json.RawMessage, emptyMsg string, invalidMsg string) error {
	if len(arguments) == 0 {
		return fmt.Errorf("%s", emptyMsg)
	}
	if _, err := normalizedToolArguments(arguments); err != nil {
		return fmt.Errorf("%s: %w", invalidMsg, err)
	}
	return nil
}

func validateAssistantTextAwaitingHuman(awaiting *AssistantTextAwaitingHuman) error {
	if awaiting == nil {
		return nil
	}
	if strings.TrimSpace(awaiting.Tool.Name) == "" {
		return fmt.Errorf("recognized assistant text handler returned empty awaiting-human tool name")
	}
	if strings.TrimSpace(awaiting.Tool.CallID) == "" {
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
