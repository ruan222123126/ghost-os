package orchestrations

import (
	"fmt"
	"strings"

	"ghost-os/bridge/orchestration/internal/domain/group"
	bridgeTasks "ghost-os/bridge/tasks"
)

type OwnerControlPromptRequest struct {
	BasePrompt       string
	OwnerNode        bridgeTasks.OrchestrationNode
	GroupNode        bridgeTasks.OrchestrationNode
	MemberNodes      map[string]bridgeTasks.OrchestrationNode
	MemberOrder      []string
	PublicTranscript group.Transcript
	LastDispatch     group.DispatchCommand
	Round            int
	DispatchToolName string
}

func BuildOwnerControlPrompt(req OwnerControlPromptRequest) string {
	toolName := ownerDispatchToolName(req.DispatchToolName)
	memberLines := ownerMemberLines(req.MemberOrder, req.MemberNodes)
	ownerPrelude := buildOwnerPromptPrelude(req.BasePrompt, req.OwnerNode.Agent.Message)
	return strings.TrimSpace(fmt.Sprintf(
		"%s\n\n你是当前群组的群主。你可以像普通 agent 一样自由分析、使用当前可见工具，并为本轮群组决策做准备。\n\n当前群组成员：\n%s\n\n当前公开 transcript：\n%s\n\n上一轮 dispatch 结果：\n%s\n\n本轮是第 %d 次群主调度。当你准备推进本轮编排时，调用一次 `%s`，选择 public_once、private_once、private_send 或 end_group。\n\nprivate_send 用于向一个或多个成员投递不同的私聊内容：只填写 private_messages，不要填写 participant_ids、order 或 instruction。",
		ownerPrelude,
		strings.Join(memberLines, "\n"),
		req.PublicTranscript.Format(),
		formatOwnerLastDispatch(req.LastDispatch),
		req.Round,
		toolName,
	))
}

func BuildOwnerControlUserPrompt(round int, dispatchToolName string) string {
	return fmt.Sprintf("开始第 %d 次群主调度。你可以先自由行动；当准备好推进群组时，再调用 %s。", round, ownerDispatchToolName(dispatchToolName))
}

func ownerMemberLines(
	memberOrder []string,
	memberNodes map[string]bridgeTasks.OrchestrationNode,
) []string {
	memberLines := make([]string, 0, len(memberOrder))
	for _, memberID := range memberOrder {
		memberLines = append(memberLines, ownerMemberLine(memberID, memberNodes[memberID]))
	}
	return memberLines
}

func ownerMemberLine(memberID string, memberNode bridgeTasks.OrchestrationNode) string {
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

func formatOwnerLastDispatch(dispatch group.DispatchCommand) string {
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

func ownerDispatchToolName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "orchestration_dispatch"
	}
	return strings.TrimSpace(name)
}
