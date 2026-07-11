package owner

import (
	"fmt"
	"strings"

	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/taskdefs"
)

type ControlPromptRequest struct {
	BasePrompt       string
	OwnerNode        taskdefs.OrchestrationNode
	GroupNode        taskdefs.OrchestrationNode
	MemberNodes      map[string]taskdefs.OrchestrationNode
	MemberOrder      []string
	PublicTranscript group.Transcript
	LastDispatch     group.DispatchCommand
	Round            int
	DispatchToolName string
}

func BuildControlPrompt(req ControlPromptRequest) string {
	toolName := dispatchToolName(req.DispatchToolName)
	memberLines := memberLines(req.MemberOrder, req.MemberNodes)
	ownerPrelude := promptPrelude(req.BasePrompt, req.OwnerNode.Agent.Message)
	return strings.TrimSpace(fmt.Sprintf(
		"%s\n\n你是当前群组的群主。你可以像普通 agent 一样自由分析、使用当前可见工具，并为本轮群组决策做准备。\n\n当前群组成员：\n%s\n\n当前公开 transcript：\n%s\n\n上一轮 dispatch 结果：\n%s\n\n本轮是第 %d 次群主调度。当你准备推进本轮编排时，调用一次 `%s`，选择 public_once、private_once、private_send 或 end_group。\n\nprivate_send 用于向一个或多个成员投递不同的私聊内容：只填写 private_messages，不要填写 participant_ids、order 或 instruction。",
		ownerPrelude,
		strings.Join(memberLines, "\n"),
		req.PublicTranscript.Format(),
		formatLastDispatch(req.LastDispatch),
		req.Round,
		toolName,
	))
}

func BuildControlUserPrompt(round int, dispatchToolName string) string {
	return fmt.Sprintf("开始第 %d 次群主调度。你可以先自由行动；当准备好推进群组时，再调用 %s。", round, dispatchToolNameOrDefault(dispatchToolName))
}

func memberLines(
	memberOrder []string,
	memberNodes map[string]taskdefs.OrchestrationNode,
) []string {
	memberLines := make([]string, 0, len(memberOrder))
	for _, memberID := range memberOrder {
		memberLines = append(memberLines, memberLine(memberID, memberNodes[memberID]))
	}
	return memberLines
}

func memberLine(memberID string, memberNode taskdefs.OrchestrationNode) string {
	title := memberID
	if memberNode.Agent != nil && strings.TrimSpace(memberNode.Agent.Title) != "" {
		title = memberNode.Agent.Title
	}
	return fmt.Sprintf("- %s: %s", memberID, title)
}

func promptPrelude(basePrompt string, ownerMessage string) string {
	parts := make([]string, 0, 2)
	if prompt := strings.TrimSpace(basePrompt); prompt != "" {
		parts = append(parts, prompt)
	}
	if message := strings.TrimSpace(ownerMessage); message != "" {
		parts = append(parts, "群主节点开场指令：\n"+message)
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func formatLastDispatch(dispatch group.DispatchCommand) string {
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

func dispatchToolName(name string) string {
	return dispatchToolNameOrDefault(name)
}

func dispatchToolNameOrDefault(name string) string {
	if strings.TrimSpace(name) == "" {
		return "orchestration_dispatch"
	}
	return strings.TrimSpace(name)
}
