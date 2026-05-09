package orchestration

import (
	"fmt"
	"strings"
)

func buildOwnerControlPrompt(
	basePrompt string,
	ownerNode OrchestrationNode,
	groupNode OrchestrationNode,
	plan orchestrationExecutionPlan,
	publicTranscript orchestrationTranscript,
	lastDispatch orchestrationDispatchResult,
	round int,
) string {
	memberLines := make([]string, 0, len(plan.groupMember[groupNode.ID]))
	for _, memberID := range plan.groupMember[groupNode.ID] {
		memberLines = append(memberLines, ownerMemberLine(memberID, plan.nodes[memberID]))
	}
	ownerPrelude := buildOwnerPromptPrelude(basePrompt, ownerNode.Agent.Message)
	return strings.TrimSpace(fmt.Sprintf(
		"%s\n\n你是当前群组的群主，只能通过工具 `%s` 做调度，不允许自由聊天。\n\n当前群组成员：\n%s\n\n当前公开 transcript：\n%s\n\n上一轮 dispatch 结果：\n%s\n\n本轮是第 %d 次群主指派。你必须调用一次 `%s`，选择 public_once、private_once 或 end_group。",
		ownerPrelude,
		orchestrationDispatchToolName,
		strings.Join(memberLines, "\n"),
		publicTranscript.Format(),
		formatOwnerLastDispatch(lastDispatch),
		round,
		orchestrationDispatchToolName,
	))
}

func ownerMemberLine(memberID string, memberNode OrchestrationNode) string {
	title := memberID
	if memberNode.Agent != nil && strings.TrimSpace(memberNode.Agent.Title) != "" {
		title = memberNode.Agent.Title
	}
	return fmt.Sprintf("- %s: %s", memberID, title)
}

func buildOwnerPromptPrelude(basePrompt string, ownerMessage string) string {
	parts := make([]string, 0, 2)
	if prompt := strings.TrimSpace(basePrompt); prompt != "" {
		parts = append(parts, prompt)
	}
	if message := strings.TrimSpace(ownerMessage); message != "" {
		parts = append(parts, "群主节点开场指令：\n"+message)
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func buildOwnerControlUserPrompt(round int) string {
	return fmt.Sprintf("开始第 %d 次群主调度。必须调用 orchestration_dispatch。", round)
}

func formatOwnerLastDispatch(dispatch orchestrationDispatchResult) string {
	if strings.TrimSpace(dispatch.Action) == "" {
		return "(none)"
	}
	return fmt.Sprintf("action=%s order=%s participants=%s instruction=%s",
		dispatch.Action,
		dispatch.Order,
		strings.Join(dispatch.ParticipantIDs, ","),
		dispatch.Instruction,
	)
}
