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

func validateToolRef(tool AssistantTextToolRef, contextPrefix string) error {
	if strings.TrimSpace(tool.Name) == "" {
		return fmt.Errorf("%s empty tool name", contextPrefix)
	}
	if strings.TrimSpace(tool.CallID) == "" {
		return fmt.Errorf("%s empty tool_call_id", contextPrefix)
	}
	return nil
}

func validateAssistantTextBatchInvocations(entries []AssistantTextToolInvocationEntry) error {
	for index, entry := range entries {
		contextPrefix := fmt.Sprintf("recognized assistant text handler returned in invocations[%d]", index)
		if err := validateToolRef(entry.Tool, contextPrefix); err != nil {
			return err
		}
		if err := validateAssistantTextArguments(
			entry.Invocation.Arguments,
			fmt.Sprintf("%s empty tool arguments", contextPrefix),
			fmt.Sprintf("%s invalid tool arguments", contextPrefix),
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
	contextPrefix := "recognized assistant text handler returned"
	if err := validateToolRef(tool, contextPrefix); err != nil {
		return err
	}
	if invocation == nil {
		return nil
	}
	return validateAssistantTextArguments(
		invocation.Arguments,
		contextPrefix+" empty tool arguments",
		contextPrefix+" invalid tool arguments",
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
	contextPrefix := "recognized assistant text handler returned awaiting-human"
	return validateToolRef(awaiting.Tool, contextPrefix)
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
