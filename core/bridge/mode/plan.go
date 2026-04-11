package mode

import (
	"errors"
	"strings"

	"ghost-os/bridge/llm"
)

const (
	TemplateIntent      = "【用户意图】"
	TemplateTasks       = "【任务编排】"
	TemplateOrder       = "【执行顺序】"
	TemplateAcceptance  = "【完成判定】"
	toolTagPrefix       = "<t:"
	toolResultMarker    = "[TOOL_TAG_RESULT]"
	MainExecutorLiteral = "main_ai"
)

var (
	ErrToolCallsForbidden = errors.New("plan mode cannot contain tool calls")
	ErrToolTagForbidden   = errors.New("plan mode cannot contain tool-tag syntax")
	ErrEmptyResponse      = errors.New("plan mode response is empty")
)

func ValidateResponse(response *llm.CompletionResponse) (llm.Message, llm.ConversationState, error) {
	if response == nil {
		return llm.Message{}, llm.ConversationState{}, ErrEmptyResponse
	}
	if response.FinishReason == llm.FinishToolCalls || len(response.Message.ToolCalls) > 0 {
		return llm.Message{}, llm.ConversationState{}, ErrToolCallsForbidden
	}
	text := strings.TrimSpace(response.Message.Text)
	if ContainsToolSyntax(text) {
		return llm.Message{}, llm.ConversationState{}, ErrToolTagForbidden
	}
	if text == "" {
		return llm.Message{}, llm.ConversationState{}, ErrEmptyResponse
	}
	return llm.Message{
		Role: llm.RoleAssistant,
		Text: text,
	}, response.ConversationState, nil
}

func ContainsToolSyntax(text string) bool {
	trimmed := strings.TrimSpace(text)
	return strings.Contains(trimmed, toolTagPrefix) || strings.Contains(trimmed, toolResultMarker)
}

func BuildSystemPrompt(basePrompt string) string {
	rules := strings.Join([]string{
		"You are the Ghost-OS planning orchestrator.",
		"In this mode, you must only decompose user intent into executable task orchestration for the main execution AI.",
		"Never execute tools or tasks.",
		"Never output tool calls, tool tags, or internal tool-result markers.",
		"Respond strictly in this template:",
		TemplateIntent,
		"- <one concise paragraph>",
		TemplateTasks,
		"1. task_id=<T1>; objective=<...>; inputs=<...>; depends_on=<...|none>; executor=" + MainExecutorLiteral,
		TemplateOrder,
		"1. <ordered execution steps>",
		TemplateAcceptance,
		"1. <clear pass/fail completion criteria>",
	}, "\n")
	trimmed := strings.TrimSpace(basePrompt)
	if trimmed == "" {
		return rules
	}
	return trimmed + "\n\n" + rules
}
