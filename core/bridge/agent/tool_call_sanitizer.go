package agent

import (
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

type invalidToolCallIssue struct {
	index int
	call  llm.ToolCall
	err   error
}

type indexedToolCall struct {
	index int
	call  llm.ToolCall
}

// sanitizeAssistantToolCalls 在 assistant tool_calls 落历史前做协议校验。
// 非法调用会被剔除，避免污染后续 provider request。
func sanitizeAssistantToolCalls(msg llm.Message) (llm.Message, []indexedToolCall, []invalidToolCallIssue) {
	cloned := llm.CloneMessages([]llm.Message{msg})
	if len(cloned) != 1 {
		return llm.Message{}, nil, nil
	}
	out := cloned[0]
	if len(out.ToolCalls) == 0 {
		return out, nil, nil
	}

	validCalls := make([]llm.ToolCall, 0, len(out.ToolCalls))
	indexedCalls := make([]indexedToolCall, 0, len(out.ToolCalls))
	issues := make([]invalidToolCallIssue, 0, len(out.ToolCalls))
	for index, call := range out.ToolCalls {
		if _, _, _, err := validateToolCall(call); err != nil {
			issues = append(issues, invalidToolCallIssue{
				index: index,
				call:  call,
				err:   err,
			})
			continue
		}
		validCalls = append(validCalls, call)
		indexedCalls = append(indexedCalls, indexedToolCall{
			index: index,
			call:  call,
		})
	}
	out.ToolCalls = validCalls
	return out, indexedCalls, issues
}

func invalidToolCallAssistantMessage(original llm.Message, issues []invalidToolCallIssue) llm.Message {
	parts := make([]string, 0, 2)
	if text := strings.TrimSpace(original.Text); text != "" {
		parts = append(parts, text)
	}
	if notice := invalidToolCallNotice(issues); notice != "" {
		parts = append(parts, notice)
	}
	return llm.Message{
		Role: llm.RoleAssistant,
		Text: strings.Join(parts, "\n\n"),
	}
}

func invalidToolCallNotice(issues []invalidToolCallIssue) string {
	if len(issues) == 0 {
		return ""
	}

	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		if issue.err == nil {
			continue
		}
		label := fmt.Sprintf("tool_call[%d]", issue.index)
		if name := strings.TrimSpace(issue.call.Name); name != "" {
			label += fmt.Sprintf(" name=%q", name)
		}
		if id := strings.TrimSpace(issue.call.ID); id != "" {
			label += fmt.Sprintf(" id=%q", id)
		}
		parts = append(parts, fmt.Sprintf("%s: %v", label, issue.err))
	}
	if len(parts) == 0 {
		return "Tool calls could not be executed because the model returned invalid tool call payloads."
	}
	return "Tool calls could not be executed because the model returned invalid tool call payloads: " + strings.Join(parts, "; ")
}
